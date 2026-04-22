package handler

import (
	"github.com/DigitalTolk/wireguard-ui/audit"
	"github.com/labstack/echo/v4"
)

const auditLoggerKey = "audit_logger"

// WithAuditLogger middleware injects the audit logger into the echo context
func WithAuditLogger(auditLog *audit.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(auditLoggerKey, auditLog)
			return next(c)
		}
	}
}

// getAuditLogger retrieves the audit logger from the echo context
func getAuditLogger(c echo.Context) *audit.Logger {
	if al, ok := c.Get(auditLoggerKey).(*audit.Logger); ok {
		return al
	}
	return nil
}

// auditLog is a convenience function to log an audit event
func auditLogEvent(c echo.Context, action, resourceType, resourceID string, details interface{}) {
	al := getAuditLogger(c)
	if al == nil {
		return
	}
	al.Log(audit.Entry{
		Actor:        currentUser(c),
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    c.RealIP(),
	})
}
