package main

import (
	"context"
	"net/http"

	ctxlog "github.com/adlandh/context-logger"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

type contextKey string

func (c contextKey) String() string {
	return string(c)
}

func (c contextKey) Saver(e echo.Context, id string) {
	ctx := context.WithValue(e.Request().Context(), c, id)
	e.SetRequest(e.Request().WithContext(ctx))
}

var requestID = contextKey("request_id")

func main() {
	// Create zap logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	// Create context logger
	ctxLogger := ctxlog.WithContext(logger, ctxlog.WithValueExtractor(requestID))

	// Then create your app
	app := echo.New()

	// Add middleware for adding request id as value to context
	app.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		RequestIDHandler: requestID.Saver,
	}))

	// Add some endpoints
	app.POST("/", func(c echo.Context) error {
		ctxLogger.Ctx(c.Request().Context()).Info("post request received!")
		return c.String(http.StatusOK, "Hello, World!")
	})

	app.GET("/", func(c echo.Context) error {
		ctxLogger.Ctx(c.Request().Context()).Info("get request received!")
		return c.String(http.StatusOK, "Hello, World!")
	})

	// And run it
	logger.Fatal("error starting server", zap.Error(app.Start(":3000")))
}
