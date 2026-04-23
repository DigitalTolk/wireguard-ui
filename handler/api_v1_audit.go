package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/xuri/excelize/v2"

	"github.com/DigitalTolk/wireguard-ui/audit"
)

// APIListAuditLogs returns paginated audit logs
func APIListAuditLogs(auditLog *audit.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		from := c.QueryParam("from")
		to := c.QueryParam("to")
		actor := c.QueryParam("actor")
		action := c.QueryParam("action")
		search := c.QueryParam("search")
		page, _ := strconv.Atoi(c.QueryParam("page"))
		perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

		entries, total, err := auditLog.Query(from, to, actor, action, search, page, perPage)
		if err != nil {
			return apiInternalError(c, "Cannot query audit logs")
		}

		if entries == nil {
			entries = []audit.LogEntry{}
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"data":     entries,
			"total":    total,
			"page":     page,
			"per_page": perPage,
		})
	}
}

// APIExportAuditLogs exports audit logs as an Excel file
func APIExportAuditLogs(auditLog *audit.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		from := c.QueryParam("from")
		to := c.QueryParam("to")
		actor := c.QueryParam("actor")
		action := c.QueryParam("action")
		search := c.QueryParam("search")

		entries, err := auditLog.QueryAll(from, to, actor, action, search)
		if err != nil {
			return apiInternalError(c, "Cannot query audit logs")
		}

		f := excelize.NewFile()
		sheet := "Audit Logs"
		f.SetSheetName("Sheet1", sheet)

		// headers
		headers := []string{"ID", "Timestamp", "Actor", "Action", "Resource Type", "Resource ID", "Details", "IP Address"}
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheet, cell, h)
		}

		// style headers bold
		style, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true},
		})
		f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s1", string(rune('A'+len(headers)-1))), style)

		// data
		for row, e := range entries {
			r := row + 2
			f.SetCellValue(sheet, fmt.Sprintf("A%d", r), e.ID)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", r), e.Timestamp.Format("2006-01-02 15:04:05"))
			f.SetCellValue(sheet, fmt.Sprintf("C%d", r), e.Actor)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", r), e.Action)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", r), e.ResourceType)
			f.SetCellValue(sheet, fmt.Sprintf("F%d", r), e.ResourceID)
			f.SetCellValue(sheet, fmt.Sprintf("G%d", r), e.Details)
			f.SetCellValue(sheet, fmt.Sprintf("H%d", r), e.IPAddress)
		}

		// auto-width columns
		for i := range headers {
			col, _ := excelize.ColumnNumberToName(i + 1)
			f.SetColWidth(sheet, col, col, 20)
		}

		c.Response().Header().Set("Content-Disposition", "attachment; filename=audit-logs.xlsx")
		c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		return f.Write(c.Response())
	}
}

// APIAuditLogFilters returns distinct actors and actions for filter dropdowns
func APIAuditLogFilters(auditLog *audit.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		actors, actions, err := auditLog.DistinctFilters()
		if err != nil {
			return apiInternalError(c, "Cannot query audit log filters")
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"actors":  actors,
			"actions": actions,
		})
	}
}
