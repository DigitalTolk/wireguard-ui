package main

import (
	"crypto/sha512"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"

	"github.com/DigitalTolk/wireguard-ui/audit"
	"github.com/DigitalTolk/wireguard-ui/emailer"
	"github.com/DigitalTolk/wireguard-ui/handler"
	"github.com/DigitalTolk/wireguard-ui/router"
	"github.com/DigitalTolk/wireguard-ui/store"
	"github.com/DigitalTolk/wireguard-ui/store/sqlitedb"
	"github.com/DigitalTolk/wireguard-ui/telegram"
	"github.com/DigitalTolk/wireguard-ui/util"
)

var (
	appVersion = "development"
	gitCommit  = "N/A"
	gitRef     = "N/A"
	buildTime  = time.Now().UTC().Format("01-02-2006 15:04:05")

	flagDisableLogin             = false
	flagBindAddress              = "0.0.0.0:5000"
	flagSmtpHostname             = "127.0.0.1"
	flagSmtpPort                 = 25
	flagSmtpUsername             string
	flagSmtpPassword             string
	flagSmtpAuthType             = "NONE"
	flagSmtpNoTLSCheck           = false
	flagSmtpEncryption           = "STARTTLS"
	flagSmtpHelo                 = "localhost"
	flagSendgridApiKey           string
	flagEmailFrom                string
	flagEmailFromName            = "WireGuard UI"
	flagTelegramToken            string
	flagTelegramAllowConfRequest = false
	flagTelegramFloodWait        = 60
	flagSessionSecret            = util.RandomString(32)
	flagSessionMaxDuration       = 90
	flagWgConfTemplate           string
	flagBasePath                 string
	flagSubnetRanges             string
)

const (
	defaultEmailSubject = "Your wireguard configuration"
	defaultEmailContent = `Hi,</br>
<p>In this email you can find your personal configuration for our wireguard server.</p>

<p>Best</p>
`
)

// embed the "templates" directory (contains wg.conf for WireGuard config generation)
//
//go:embed templates/wg.conf
var embeddedTemplates embed.FS

// embed the "assets" directory (built React SPA)
//
//go:embed assets/*
var embeddedAssets embed.FS

func init() {
	flag.BoolVar(&flagDisableLogin, "disable-login", util.LookupEnvOrBool("DISABLE_LOGIN", flagDisableLogin), "Disable authentication on the app. This is potentially dangerous.")
	flag.StringVar(&flagBindAddress, "bind-address", util.LookupEnvOrString("BIND_ADDRESS", flagBindAddress), "Address:Port to which the app will be bound.")
	flag.StringVar(&flagSmtpHostname, "smtp-hostname", util.LookupEnvOrString("SMTP_HOSTNAME", flagSmtpHostname), "SMTP Hostname")
	flag.IntVar(&flagSmtpPort, "smtp-port", util.LookupEnvOrInt("SMTP_PORT", flagSmtpPort), "SMTP Port")
	flag.StringVar(&flagSmtpHelo, "smtp-helo", util.LookupEnvOrString("SMTP_HELO", flagSmtpHelo), "SMTP HELO Hostname")
	flag.StringVar(&flagSmtpUsername, "smtp-username", util.LookupEnvOrString("SMTP_USERNAME", flagSmtpUsername), "SMTP Username")
	flag.BoolVar(&flagSmtpNoTLSCheck, "smtp-no-tls-check", util.LookupEnvOrBool("SMTP_NO_TLS_CHECK", flagSmtpNoTLSCheck), "Disable TLS verification for SMTP. This is potentially dangerous.")
	flag.StringVar(&flagSmtpEncryption, "smtp-encryption", util.LookupEnvOrString("SMTP_ENCRYPTION", flagSmtpEncryption), "SMTP Encryption : NONE, SSL, SSLTLS, TLS or STARTTLS (by default)")
	flag.StringVar(&flagSmtpAuthType, "smtp-auth-type", util.LookupEnvOrString("SMTP_AUTH_TYPE", flagSmtpAuthType), "SMTP Auth Type : PLAIN, LOGIN or NONE.")
	flag.StringVar(&flagEmailFrom, "email-from", util.LookupEnvOrString("EMAIL_FROM_ADDRESS", flagEmailFrom), "'From' email address.")
	flag.StringVar(&flagEmailFromName, "email-from-name", util.LookupEnvOrString("EMAIL_FROM_NAME", flagEmailFromName), "'From' email name.")
	flag.StringVar(&flagTelegramToken, "telegram-token", util.LookupEnvOrString("TELEGRAM_TOKEN", flagTelegramToken), "Telegram bot token for distributing configs to clients.")
	flag.BoolVar(&flagTelegramAllowConfRequest, "telegram-allow-conf-request", util.LookupEnvOrBool("TELEGRAM_ALLOW_CONF_REQUEST", flagTelegramAllowConfRequest), "Allow users to get configs from the bot by sending a message.")
	flag.IntVar(&flagTelegramFloodWait, "telegram-flood-wait", util.LookupEnvOrInt("TELEGRAM_FLOOD_WAIT", flagTelegramFloodWait), "Time in minutes before the next conf request is processed.")
	flag.StringVar(&flagWgConfTemplate, "wg-conf-template", util.LookupEnvOrString("WG_CONF_TEMPLATE", flagWgConfTemplate), "Path to custom wg.conf template.")
	flag.StringVar(&flagBasePath, "base-path", util.LookupEnvOrString("BASE_PATH", flagBasePath), "The base path of the URL")
	flag.StringVar(&flagSubnetRanges, "subnet-ranges", util.LookupEnvOrString("SUBNET_RANGES", flagSubnetRanges), "IP ranges to choose from when assigning an IP for a client.")
	flag.IntVar(&flagSessionMaxDuration, "session-max-duration", util.LookupEnvOrInt("SESSION_MAX_DURATION", flagSessionMaxDuration), "Max time in days a remembered session is refreshed and valid.")

	var (
		smtpPasswordLookup   = util.LookupEnvOrString("SMTP_PASSWORD", flagSmtpPassword)
		sendgridApiKeyLookup = util.LookupEnvOrString("SENDGRID_API_KEY", flagSendgridApiKey)
		sessionSecretLookup  = util.LookupEnvOrString("SESSION_SECRET", flagSessionSecret)
	)

	if smtpPasswordLookup != "" {
		flag.StringVar(&flagSmtpPassword, "smtp-password", smtpPasswordLookup, "SMTP Password")
	} else {
		flag.StringVar(&flagSmtpPassword, "smtp-password", util.LookupEnvOrFile("SMTP_PASSWORD_FILE", flagSmtpPassword), "SMTP Password File")
	}

	if sendgridApiKeyLookup != "" {
		flag.StringVar(&flagSendgridApiKey, "sendgrid-api-key", sendgridApiKeyLookup, "Your sendgrid api key.")
	} else {
		flag.StringVar(&flagSendgridApiKey, "sendgrid-api-key", util.LookupEnvOrFile("SENDGRID_API_KEY_FILE", flagSendgridApiKey), "File containing your sendgrid api key.")
	}

	if sessionSecretLookup != "" {
		flag.StringVar(&flagSessionSecret, "session-secret", sessionSecretLookup, "The key used to encrypt session cookies.")
	} else {
		flag.StringVar(&flagSessionSecret, "session-secret", util.LookupEnvOrFile("SESSION_SECRET_FILE", flagSessionSecret), "File containing the key used to encrypt session cookies.")
	}

	flag.Parse()

	util.DisableLogin = flagDisableLogin
	util.BindAddress = flagBindAddress
	util.SmtpHostname = flagSmtpHostname
	util.SmtpPort = flagSmtpPort
	util.SmtpHelo = flagSmtpHelo
	util.SmtpUsername = flagSmtpUsername
	util.SmtpPassword = flagSmtpPassword
	util.SmtpAuthType = flagSmtpAuthType
	util.SmtpNoTLSCheck = flagSmtpNoTLSCheck
	util.SmtpEncryption = flagSmtpEncryption
	util.SendgridApiKey = flagSendgridApiKey
	util.EmailFrom = flagEmailFrom
	util.EmailFromName = flagEmailFromName
	util.SessionSecret = sha512.Sum512([]byte(flagSessionSecret))
	util.SessionMaxDuration = int64(flagSessionMaxDuration) * 86_400
	util.WgConfTemplate = flagWgConfTemplate
	util.BasePath = util.ParseBasePath(flagBasePath)
	util.SubnetRanges = util.ParseSubnetRanges(flagSubnetRanges)

	util.OIDCIssuerURL = util.LookupEnvOrString(util.OIDCIssuerURLEnvVar, "")
	util.OIDCClientID = util.LookupEnvOrString(util.OIDCClientIDEnvVar, "")
	oidcSecret := util.LookupEnvOrString(util.OIDCClientSecretEnvVar, "")
	if oidcSecret == "" {
		oidcSecret = util.LookupEnvOrFile(util.OIDCClientSecretFileVar, "")
	}
	util.OIDCClientSecret = oidcSecret
	util.OIDCRedirectURL = util.LookupEnvOrString(util.OIDCRedirectURLEnvVar, "")
	util.OIDCScopes = util.LookupEnvOrStrings(util.OIDCScopesEnvVar, []string{"openid", "profile", "email"})
	util.OIDCAutoProvision = util.LookupEnvOrBool(util.OIDCAutoProvisionEnvVar, true)
	util.OIDCAdminGroups = util.LookupEnvOrStrings(util.OIDCAdminGroupsEnvVar, []string{})

	lvl, _ := util.ParseLogLevel(util.LookupEnvOrString(util.LogLevel, "INFO"))

	telegram.Token = flagTelegramToken
	telegram.AllowConfRequest = flagTelegramAllowConfRequest
	telegram.FloodWait = flagTelegramFloodWait
	telegram.LogLevel = lvl

	if lvl <= log.INFO {
		fmt.Println("Wireguard UI")
		fmt.Println("App Version\t:", appVersion)
		fmt.Println("Git Commit\t:", gitCommit)
		fmt.Println("Git Ref\t\t:", gitRef)
		fmt.Println("Build Time\t:", buildTime)
		fmt.Println("Git Repo\t:", "https://github.com/DigitalTolk/wireguard-ui")
		fmt.Println("Authentication\t:", !util.DisableLogin)
		fmt.Println("Bind address\t:", util.BindAddress)
		fmt.Println("Email from\t:", util.EmailFrom)
		fmt.Println("Email from name\t:", util.EmailFromName)
		fmt.Println("Custom wg.conf\t:", util.WgConfTemplate)
		fmt.Println("Base path\t:", util.BasePath+"/")
		fmt.Println("Subnet ranges\t:", util.GetSubnetRangesString())
	}
}

func main() {
	dbPath := "./db/wireguard-ui.db"

	// Check for legacy JSON database and migrate if needed
	jsonDBPath := "./db"
	if _, err := os.Stat(filepath.Join(jsonDBPath, "server")); err == nil {
		tmpDBPath := "./db_sqlite_migration/wireguard-ui.db"
		tmpDB, err := sqlitedb.New(tmpDBPath)
		if err != nil {
			panic(fmt.Sprintf("Cannot create SQLite database for migration: %v", err))
		}
		if err := sqlitedb.MigrateFromJSON(tmpDB, jsonDBPath); err != nil {
			panic(fmt.Sprintf("Migration from JSON to SQLite failed: %v", err))
		}
		if err := os.MkdirAll("./db", 0750); err != nil {
			panic(fmt.Sprintf("Cannot create db directory: %v", err))
		}
		if err := os.Rename(tmpDBPath, dbPath); err != nil {
			panic(fmt.Sprintf("Cannot move SQLite database: %v", err))
		}
		os.RemoveAll("./db_sqlite_migration")
	}

	db, err := sqlitedb.New(dbPath)
	if err != nil {
		panic(err)
	}
	if err := db.Init(); err != nil {
		panic(err)
	}

	// wg.conf template for WireGuard config generation
	tmplDir, _ := fs.Sub(fs.FS(embeddedTemplates), "templates")

	// create the wireguard config on start, if it doesn't exist
	initServerConfig(db, tmplDir)

	if err := util.ValidateAndFixSubnetRanges(db); err != nil {
		panic(err)
	}

	if lvl, _ := util.ParseLogLevel(util.LookupEnvOrString(util.LogLevel, "INFO")); lvl <= log.INFO {
		fmt.Println("Valid subnet ranges:", util.GetSubnetRangesString())
	}

	// set up email sender
	var sendmail emailer.Emailer
	if util.SendgridApiKey != "" {
		sendmail = emailer.NewSendgridApiMail(util.SendgridApiKey, util.EmailFromName, util.EmailFrom)
	} else {
		sendmail = emailer.NewSmtpMail(util.SmtpHostname, util.SmtpPort, util.SmtpUsername, util.SmtpPassword, util.SmtpHelo, util.SmtpNoTLSCheck, util.SmtpAuthType, util.EmailFromName, util.EmailFrom, util.SmtpEncryption)
	}

	// set up Echo with session middleware
	app := router.New(util.SessionSecret)

	// audit logger
	auditLog := audit.NewLogger(db.DB())

	// API v1 routes
	apiV1 := app.Group(util.BasePath+"/api/v1", handler.WithAuditLogger(auditLog))
	router.RegisterAPIv1(apiV1, db, sendmail, tmplDir, defaultEmailSubject, defaultEmailContent, auditLog)

	// OIDC SSO routes
	oidcProvider, err := handler.NewOIDCProvider()
	if err != nil {
		log.Warnf("OIDC configuration failed: %v", err)
	}
	if oidcProvider != nil {
		apiV1.GET("/auth/oidc/login", handler.APIStartOIDCLogin(oidcProvider))
		apiV1.GET("/auth/oidc/callback", handler.APIHandleOIDCCallback(oidcProvider, db))
		log.Info("OIDC authentication enabled")
	}

	// health check + favicon (no auth)
	app.GET(util.BasePath+"/_health", handler.Health())
	app.GET(util.BasePath+"/favicon", handler.Favicon())

	// SPA frontend (embedded React app)
	assetsDir, _ := fs.Sub(fs.FS(embeddedAssets), "assets")
	assetHandler := http.FileServer(http.FS(assetsDir))
	indexHTML, _ := fs.ReadFile(assetsDir, "index.html")

	// serve static files (JS, CSS, fonts)
	app.GET(util.BasePath+"/static/*", echo.WrapHandler(http.StripPrefix(util.BasePath+"/", assetHandler)))

	// SPA catch-all: serve index.html for all non-file routes
	serveIndex := func(c echo.Context) error {
		if indexHTML == nil {
			return c.String(http.StatusNotFound, "Not found")
		}
		return c.HTMLBlob(http.StatusOK, indexHTML)
	}
	app.GET(util.BasePath+"/", serveIndex)
	if util.BasePath == "" {
		app.GET("/", serveIndex)
	}
	app.GET(util.BasePath+"/*", serveIndex)

	// Telegram bot
	initDeps := telegram.TgBotInitDependencies{
		DB:                             db,
		SendRequestedConfigsToTelegram: util.SendRequestedConfigsToTelegram,
	}
	initTelegram(initDeps)

	// Start server
	if strings.HasPrefix(util.BindAddress, "unix://") {
		if err := syscall.Unlink(util.BindAddress[6:]); err != nil {
			app.Logger.Fatalf("Cannot unlink unix socket: Error: %v", err)
		}
		l, err := net.Listen("unix", util.BindAddress[6:])
		if err != nil {
			app.Logger.Fatalf("Cannot create unix socket. Error: %v", err)
		}
		app.Listener = l
		app.Logger.Fatal(app.Start(""))
	} else {
		app.Logger.Fatal(app.Start(util.BindAddress))
	}
}

func initServerConfig(db store.IStore, tmplDir fs.FS) {
	settings, err := db.GetGlobalSettings()
	if err != nil {
		log.Fatalf("Cannot get global settings: %v", err)
	}

	if _, err := os.Stat(settings.ConfigFilePath); err == nil {
		return
	}

	server, err := db.GetServer()
	if err != nil {
		log.Fatalf("Cannot get server config: %v", err)
	}

	clients, err := db.GetClients(false)
	if err != nil {
		log.Fatalf("Cannot get client config: %v", err)
	}

	users, err := db.GetUsers()
	if err != nil {
		log.Fatalf("Cannot get user config: %v", err)
	}

	err = util.WriteWireGuardServerConfig(tmplDir, server, clients, users, settings)
	if err != nil {
		log.Fatalf("Cannot create server config: %v", err)
	}
}

func initTelegram(initDeps telegram.TgBotInitDependencies) {
	go func() {
		for {
			err := telegram.Start(initDeps)
			if err == nil {
				break
			}
		}
	}()
}
