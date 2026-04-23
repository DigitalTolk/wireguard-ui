package router

import (
	"io/fs"

	"github.com/labstack/echo/v4"

	"github.com/DigitalTolk/wireguard-ui/audit"
	"github.com/DigitalTolk/wireguard-ui/emailer"
	"github.com/DigitalTolk/wireguard-ui/handler"
	"github.com/DigitalTolk/wireguard-ui/store"
)

// RegisterAPIv1 registers all API v1 routes under the given group
func RegisterAPIv1(g *echo.Group, db store.IStore, mailer emailer.Emailer, tmplDir fs.FS, emailSubject, emailContent, appVersion, gitCommit string, auditLog *audit.Logger) {
	// Auth
	g.GET("/auth/me", handler.APIGetMe(db), handler.APIAuth)
	g.POST("/auth/logout", handler.APILogout(), handler.APIAuth)
	g.GET("/auth/info", handler.APIAppInfo(appVersion, gitCommit))

	// Clients
	clients := g.Group("/clients", handler.APIAuth)
	clients.GET("", handler.APIListClients(db))
	clients.GET("/export", handler.APIExportClients(db))
	clients.GET("/:id", handler.APIGetClient(db))
	clients.POST("", handler.APICreateClient(db), handler.ContentTypeJson)
	clients.PUT("/:id", handler.APIUpdateClient(db), handler.ContentTypeJson)
	clients.PATCH("/:id/status", handler.APIPatchClientStatus(db), handler.ContentTypeJson)
	clients.DELETE("/:id", handler.APIDeleteClient(db))
	clients.GET("/:id/config", handler.APIDownloadClientConfig(db))
	clients.GET("/:id/qrcode", handler.APIGetClientQRCode(db))
	clients.POST("/:id/email", handler.APIEmailClient(db, mailer, emailSubject, emailContent), handler.ContentTypeJson)

	// Server (admin only)
	server := g.Group("/server", handler.APIAuth, handler.APIAdmin)
	server.GET("", handler.APIGetServer(db))
	server.PUT("/interface", handler.APIUpdateServerInterface(db), handler.ContentTypeJson)
	server.POST("/keypair", handler.APIRegenerateServerKeypair(db), handler.ContentTypeJson)
	server.POST("/apply-config", handler.APIApplyServerConfig(db, tmplDir), handler.ContentTypeJson)
	server.GET("/config-status", handler.APIConfigStatus(db))

	// Settings (admin only)
	settings := g.Group("/settings", handler.APIAuth, handler.APIAdmin)
	settings.GET("", handler.APIGetSettings(db))
	settings.PUT("", handler.APIUpdateSettings(db), handler.ContentTypeJson)

	// Users (admin only for list/create/delete)
	// Users (read-only — managed via SSO)
	users := g.Group("/users", handler.APIAuth, handler.APIAdmin)
	users.GET("", handler.APIListUsers(db))
	users.GET("/:username", handler.APIGetUser(db))

	// Wake-on-LAN
	wolGroup := g.Group("/wol-hosts", handler.APIAuth)
	wolGroup.GET("", handler.APIListWolHosts(db))
	wolGroup.POST("", handler.APISaveWolHost(db), handler.ContentTypeJson)
	wolGroup.DELETE("/:mac", handler.APIDeleteWolHost(db))
	wolGroup.POST("/:mac/wake", handler.APIWakeHost(db), handler.ContentTypeJson)

	// Utilities
	utils := g.Group("", handler.APIAuth)
	utils.GET("/machine-ips", handler.APIMachineIPs())
	utils.GET("/subnet-ranges", handler.APISubnetRanges())
	utils.GET("/suggest-client-ips", handler.APISuggestClientIPs(db))

	// Status
	g.GET("/status", handler.APIServerStatus(db), handler.APIAuth)

	// Audit logs (admin only)
	auditGroup := g.Group("/audit-logs", handler.APIAuth, handler.APIAdmin)
	auditGroup.GET("", handler.APIListAuditLogs(auditLog))
	auditGroup.GET("/filters", handler.APIAuditLogFilters(auditLog))
	auditGroup.GET("/export", handler.APIExportAuditLogs(auditLog))
}
