package v3

import (
	"github.com/labstack/echo"

	"go.opentelemetry.io/contrib/plugins/labstack/echo/internal"

	otelglobal "go.opentelemetry.io/otel/api/global"
)

func Middleware(service string, opts ...internal.Option) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		conf := internal.Config{Service: service}
		for _, opt := range opts {
			opt(&conf)
		}

		if conf.Tracer == nil {
			conf.Tracer = otelglobal.Tracer(internal.TracerName)
		}
		if conf.Propagators == nil {
			conf.Propagators = otelglobal.Propagators()
		}

		return func(c echo.Context) error {
			r, span := internal.StartTrace(c.Request(), c.Path(), conf)
			c.SetRequest(r)

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			internal.EndTrace(span, c.Response().Status)
			return err
		}
	}
}
