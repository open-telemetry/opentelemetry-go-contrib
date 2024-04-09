# Migration Plan
- Current release - v0.49.0
    - Support only for v1.20.0 semantic conventions
- v0.50.0
    - Add support for v1.20.0, v1.24.0, and both semantic conventions
    - Default to v1.20.0
    - Warn users when using v1.20.0
- v0.51.0
    - Default to v1.24.0
    - Warn users when using v1.20.0 or both semantic conventions
- v0.52.0
    - Remove support for v1.20.0
    - Remove support for both semantic conventions
    - All users will be using v1.24.0

# User Migration Story
The goal is to allow users to upgrade versions of the `otelhttp` package and enable migrations of dashboards and alerts incrementally.

When the user first upgrades to v0.49 they will receive a warning that they are using v1.20.0 semantic conventions. This is an indication that action needs to be taken or their dashboards and alerts will be affected.

They can then set the environment variable `OTEL_HTTP_CLIENT_COMPATIBILITY_MODE` to `http/dup` and receive both the original v1.20.0 attributes and the new v1.24.0 attributes. This will allow them to update their dashboards and alerts to use the new attributes and validate they are not missing any data.
Once the user has completed the migration, they can then set `OTEL_HTTP_CLIENT_COMPATIBILITY_MODE` to `http` to complete the migration to the stable http metric names.
