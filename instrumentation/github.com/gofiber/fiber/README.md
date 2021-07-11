### OpenTelemetry for [Fiber](https://gofiber.io/)



### Example
```go

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
```
