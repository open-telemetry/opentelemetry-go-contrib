package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	echotrace "go.opentelemetry.io/contrib/plugins/labstack/echo/v4"

	otelglobal "go.opentelemetry.io/otel/api/global"
	oteltracestdout "go.opentelemetry.io/otel/exporters/trace/stdout"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	initTracer()

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(echotrace.Middleware("trace-demo"))

	// Route => handler
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, world\n")
	})

	e.GET("/greeting/:name", func(c echo.Context) error {
		name := c.Param("name")
		return c.String(http.StatusOK, "Hello, "+name+"\n")
	})

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}

func initTracer() {
	exporter, err := oteltracestdout.NewExporter(oteltracestdout.Options{PrettyPrint: true})
	if err != nil {
		log.Fatal(err)
	}
	cfg := sdktrace.Config{
		DefaultSampler: sdktrace.AlwaysSample(),
	}
	tp, err := sdktrace.NewProvider(
		sdktrace.WithConfig(cfg),
		sdktrace.WithSyncer(exporter),
	)
	if err != nil {
		log.Fatal(err)
	}
	otelglobal.SetTraceProvider(tp)
}
