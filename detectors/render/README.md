# Render Resource Detector for Go

<!--[![Go Reference][goref-image]][goref-url]-->
<!--[goref-image]: https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/render.svg-->
<!--[goref-url]: https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/render-->

A resource detector for the [Render](https://render.com) cloud platform.

It reads the `RENDER_SERVICE_ID`, `RENDER_SERVICE_NAME`, and
`RENDER_GIT_COMMIT` environment variables that Render sets automatically on
every running service, and maps them to the `service.name` and
`service.version` resource attributes.
