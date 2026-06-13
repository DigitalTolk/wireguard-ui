package handler

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/model"
)

func saveClientForDeleteTest(t *testing.T, env *testEnv, name, email string) string {
	t.Helper()
	id := xid.New().String()
	now := time.Now().UTC()
	require.NoError(t, env.db.SaveClient(model.Client{
		ID: id, Name: name, Email: email,
		PublicKey: "pk-" + id, PrivateKey: "priv-" + id,
		AllocatedIPs:    []string{"10.252.1." + id[len(id)-3:] + "/32"},
		AllowedIPs:      []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	}))
	return id
}

func TestDeleteByEmail_RejectsMissingEmail(t *testing.T) {
	env := setupTestEnv(t)
	req, rec := jsonRequest(http.MethodDelete, "/api/v1/clients/by-email", nil)
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APIDeleteClientsByEmail(env.db, env.cw)(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteByEmail_RejectsMalformedEmail(t *testing.T) {
	env := setupTestEnv(t)
	req, rec := jsonRequest(http.MethodDelete, "/api/v1/clients/by-email?email=not-an-email", nil)
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APIDeleteClientsByEmail(env.db, env.cw)(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteByEmail_NoMatches_404(t *testing.T) {
	env := setupTestEnv(t)
	saveClientForDeleteTest(t, env, "alice", "alice@example.com")

	req, rec := jsonRequest(http.MethodDelete, "/api/v1/clients/by-email?email=ghost@example.com", nil)
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APIDeleteClientsByEmail(env.db, env.cw)(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteByEmail_SingleMatch_Deletes(t *testing.T) {
	env := setupTestEnv(t)
	id := saveClientForDeleteTest(t, env, "alice", "alice@example.com")

	req, rec := jsonRequest(http.MethodDelete, "/api/v1/clients/by-email?email=alice@example.com", nil)
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APIDeleteClientsByEmail(env.db, env.cw)(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, float64(1), resp["deleted"])
	ids, _ := resp["ids"].([]interface{})
	require.Len(t, ids, 1)
	assert.Equal(t, id, ids[0])

	all, _ := env.db.GetClients(false)
	assert.Empty(t, all, "the client should be gone")
}

func TestDeleteByEmail_MultiMatch_WithoutConfirm_409(t *testing.T) {
	env := setupTestEnv(t)
	saveClientForDeleteTest(t, env, "bob-laptop", "bob@example.com")
	saveClientForDeleteTest(t, env, "bob-phone", "bob@example.com")

	req, rec := jsonRequest(http.MethodDelete, "/api/v1/clients/by-email?email=bob@example.com", nil)
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APIDeleteClientsByEmail(env.db, env.cw)(c))
	require.Equal(t, http.StatusConflict, rec.Code)

	all, _ := env.db.GetClients(false)
	assert.Len(t, all, 2, "no clients should be deleted without confirm_all")

	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	errObj, _ := resp["error"].(map[string]interface{})
	require.NotNil(t, errObj)
	assert.Equal(t, "CONFIRM_REQUIRED", errObj["code"])
	assert.Equal(t, float64(2), errObj["matched"])
}

func TestDeleteByEmail_MultiMatch_WithConfirm_DeletesAll(t *testing.T) {
	env := setupTestEnv(t)
	saveClientForDeleteTest(t, env, "bob-laptop", "bob@example.com")
	saveClientForDeleteTest(t, env, "bob-phone", "bob@example.com")
	// And one that shouldn't be touched
	keeperID := saveClientForDeleteTest(t, env, "carol-laptop", "carol@example.com")

	req, rec := jsonRequest(http.MethodDelete, "/api/v1/clients/by-email?email=bob@example.com&confirm_all=true", nil)
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APIDeleteClientsByEmail(env.db, env.cw)(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, float64(2), resp["deleted"])

	all, _ := env.db.GetClients(false)
	require.Len(t, all, 1, "only carol should remain")
	assert.Equal(t, keeperID, all[0].Client.ID)
}

func TestDeleteByEmail_IsCaseInsensitive(t *testing.T) {
	env := setupTestEnv(t)
	saveClientForDeleteTest(t, env, "dave", "Dave@Example.COM")

	req, rec := jsonRequest(http.MethodDelete, "/api/v1/clients/by-email?email=dave@example.com", nil)
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APIDeleteClientsByEmail(env.db, env.cw)(c))
	require.Equal(t, http.StatusOK, rec.Code)

	all, _ := env.db.GetClients(false)
	assert.Empty(t, all)
}

func TestDeleteByEmail_BehindTokenAuth_RequiresValidToken(t *testing.T) {
	env := setupTestEnv(t)
	saveClientForDeleteTest(t, env, "guarded", "guarded@example.com")

	env.echo.DELETE("/api/v1/clients/by-email", APIDeleteClientsByEmail(env.db, env.cw), APITokenAuth(env.db))

	req, rec := jsonRequest(http.MethodDelete, "/api/v1/clients/by-email?email=guarded@example.com", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Still there.
	all, _ := env.db.GetClients(false)
	assert.Len(t, all, 1)
}
