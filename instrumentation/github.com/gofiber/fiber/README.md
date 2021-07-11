### OpenTelemetry for [Fiber](https://gofiber.io/)



### Example
```go
package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gofiber/fiber/otelfiber"
	"oteltrace "go.opentelemetry.io/otel/trace"

)

var tracer = otel.Tracer("fiber-server")

func main() {
	// your exporter setup 
	
	app := fiber.New()
	app.Use(otelfiber.Middleware("my-server"))

	// trace from parent tracer
	app.Get("/users/:id", func(ctx *fiber.Ctx) error {
		id := c.Params("id")
		// use ctx.UserContext() for tracer's context
		_, span := tracer.Start(ctx.UserContext(), "getUser", oteltrace.WithAttributes(attribute.String("id", id)))
		defer span.End()
		return c.JSON(fiber.Map{"id": id})
	})
	
	log.Fatal(app.Listen(":3000"))

}
```
