package handler

import (
	"net/http"
	"net/mail"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"

	"github.com/DigitalTolk/wireguard-ui/store"
)

// APIDeleteClientsByEmail removes every client whose email matches the query
// param (case-insensitive). To prevent a typo from wiping multiple device
// entries belonging to one person, multi-match deletes require the explicit
// `confirm_all=true` query parameter.
//
//   - 0 matches → 404
//   - 1 match → delete, 200 with {deleted: 1, ids: [...]}
//   - >1 matches without confirm_all → 409 with {matched, ids, message}
//   - >1 matches with confirm_all=true → delete all, 200 with the full list
func APIDeleteClientsByEmail(db store.IStore, cw *ConfigWriter) echo.HandlerFunc {
	return func(c echo.Context) error {
		email := strings.TrimSpace(c.QueryParam("email"))
		if email == "" {
			return apiBadRequest(c, "email query parameter is required")
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return apiBadRequest(c, "Valid email is required")
		}
		confirmAll := strings.EqualFold(c.QueryParam("confirm_all"), "true")

		allClients, err := db.GetClients(false)
		if err != nil {
			return apiInternalError(c, "Cannot list clients")
		}

		var matchIDs []string
		var matchNames []string
		for _, cd := range allClients {
			if strings.EqualFold(cd.Client.Email, email) {
				matchIDs = append(matchIDs, cd.Client.ID)
				matchNames = append(matchNames, cd.Client.Name)
			}
		}

		switch {
		case len(matchIDs) == 0:
			return apiNotFound(c, "No clients found for that email")
		case len(matchIDs) > 1 && !confirmAll:
			// 409 with enough info for the caller to retry knowingly.
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "CONFIRM_REQUIRED",
					"message": "Multiple clients match this email. Re-send with ?confirm_all=true to delete them all.",
					"matched": len(matchIDs),
					"ids":     matchIDs,
					"names":   matchNames,
				},
			})
		}

		// Delete all matches. We don't short-circuit on the first error
		// because partial-delete state is worse than full-delete state; record
		// what failed and surface it in the response so the caller can retry.
		deleted := make([]string, 0, len(matchIDs))
		failed := make([]string, 0)
		for _, id := range matchIDs {
			if err := db.DeleteClient(id); err != nil {
				log.Warnf("delete-by-email: failed to delete %s: %v", id, err)
				failed = append(failed, id)
				continue
			}
			deleted = append(deleted, id)
		}
		if len(deleted) > 0 {
			cw.Trigger()
		}

		auditLogEvent(c, "client.delete.by_email", "client", "", map[string]interface{}{
			"email": email, "deleted": deleted, "failed": failed,
		})

		if len(failed) > 0 && len(deleted) == 0 {
			return apiInternalError(c, "Failed to delete matching clients")
		}
		resp := map[string]interface{}{
			"deleted": len(deleted),
			"ids":     deleted,
		}
		if len(failed) > 0 {
			resp["failed"] = failed
		}
		return c.JSON(http.StatusOK, resp)
	}
}
