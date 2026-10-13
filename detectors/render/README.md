# Render Resource Detector

A resource detector for the [Render](https://render.com) cloud platform.

It reads the `RENDER_SERVICE_ID`, `RENDER_INSTANCE_ID`, `RENDER_SERVICE_NAME`,
and `RENDER_GIT_COMMIT` environment variables that Render sets automatically
on every running service, and maps them to `service.instance.id`,
`service.name`, and `service.version` resource attributes.
