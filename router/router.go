package router

import (
	"github.com/DigitalTolk/wireguard-ui/util"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

// New creates a configured Echo instance with session middleware and logging
func New(secret [64]byte) *echo.Echo {
	e := echo.New()

	cookiePath := util.GetCookiePath()

	cookieStore := sessions.NewCookieStore(secret[:32], secret[32:])
	cookieStore.Options.Path = cookiePath
	cookieStore.Options.HttpOnly = true
	cookieStore.MaxAge(86400 * 7)

	e.Use(session.Middleware(cookieStore))

	lvl, err := util.ParseLogLevel(util.LookupEnvOrString(util.LogLevel, "INFO"))
	if err != nil {
		log.Fatal(err)
	}
	e.Logger.SetLevel(lvl)
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:      true,
		LogStatus:   true,
		LogMethod:   true,
		LogLatency:  true,
		LogRemoteIP: true,
		LogError:    true,
		Skipper: func(c echo.Context) bool {
			resp := c.Response()
			if resp.Status >= 500 && lvl > log.ERROR {
				return true
			} else if resp.Status >= 400 && lvl > log.WARN {
				return true
			} else if lvl > log.DEBUG {
				return true
			}
			return false
		},
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			e.Logger.Infof("%s %s %d %v", v.Method, v.URI, v.Status, v.Latency)
			return nil
		},
	}))
	e.HideBanner = true
	e.HidePort = lvl > log.INFO
	e.Validator = NewValidator()

	return e
}
