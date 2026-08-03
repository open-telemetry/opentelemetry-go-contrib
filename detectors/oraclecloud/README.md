# Oracle Cloud Infrastructure Resource Detector for Go

<!--[![Go Reference][goref-image]][goref-url]-->
<!--[goref-image]: https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/oraclecloud.svg-->
<!--[goref-url]: https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/oraclecloud-->

The detector reports the OCI instance identifier as both `host.id` and the
semantic-convention `cloud.resource_id`. Use `WithAttributeFilter` when only
one representation is required.
