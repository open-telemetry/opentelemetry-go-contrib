## `go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego`

This package provides tracing and metrics to the `github.com/astaxie/beego` package.

To enable tracing and metrics in your beego application:

```go
package main

import (
        "github.com/astaxie/beego"

        otelBeego "go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego"
)

func main() {
        // Init the tracer and metric export pipelines

        // Create your routes
        
        // Create the MiddleWare
        mware := otelBeego.NewOTelBeegoMiddleWare("example-server")

        // Run the server with the MiddleWare
        beego.RunWithMiddleWares("localhost:8080", mware)
}
```

You can customize instrumentation by passing any of the following options to `NewOtelBeegoMiddleWare`:

| Function | Description |
| -------- | ----------- |
| `WithTracer(trace.Tracer)` | The tracer to be used to create spans for the beego server. If not specified, `global.Tracer("go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego")` will be used. |
| `WithMeter(metric.Meter)` | The meter to be used to create the instruments. If not specified, `global.Meter("go.opentelemery.io/contrib/instrumentation/github.com/astaxie/beego")` will be used. |
| `WithPropagators(propagation.Propagators)` | The propagators to be used. If not specified, `global.Propagators()` will be used. |
| `WithFilter(Filter)` | Adds an additional filter function to the configuration. Defaults to no filters. |
| `WithSpanNameFormatter(SpanNameFormatter)` | The formatter used to format span names. If not specified, the route will be used instead. |

You can also trace the `Render`, `RenderString`, and `RenderBytes` functions. You should disable the `AutoRender` setting either programmatically or in the config file, so you can explicitly call the traced implementations:

```go
package main

import (
        "github.com/astaxie/beego"

        otelBeego "go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego"
)

type ExampleController struct {
        beego.Controller
}

func (c *ExampleController) Get() {
        c.TplName = "index.ptl"

        // explicit call to traced Render function
        otelBeego.Render(&c.Controller)
}

func main() {
        // Init the tracer and metric export pipelines

        // Disable autorender
        beego.BConfig.WebConfig.AutoRender = false

        // Create your routes
        beego.Router("/", &ExampleController{})

        // Create the MiddleWare
        mware := otelBeego.NewOTelBeegoMiddleWare("example-server")

        // Run the server with the MiddleWare
        beego.RunWithMiddleWares("localhost:8080", mware)
}
```

| Function | Description |
| -------- | ----------- |
| `Render(*beego.Controller) error` | Provides tracing to `beego.Controller.Render`. |
| `RenderString(*beego.Controller) (string, error)` | Provides tracing to `beego.Controller.RenderString`. |
| `RenderBytes(*beego.Controller) ([]byte, error)` | Provides tracing to `beego.Controller.RenderBytes`. |
