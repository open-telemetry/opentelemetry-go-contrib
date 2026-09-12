// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package dynatrace provides a [resource.Detector] which supports detecting
attributes of hosts monitored by the Dynatrace OneAgent.

The detector reads the host enrichment file dt_host_metadata.properties which
the OneAgent writes to /var/lib/dynatrace/enrichment on non-Windows systems and
to %ProgramData%\dynatrace\enrichment on Windows. Each of the following
attributes is added if it is present in that file:

  - dt.entity.host
  - host.name
  - dt.smartscape.host

The dt.entity.host and dt.smartscape.host attributes identify the host entity
in Dynatrace and have no equivalent in the OpenTelemetry [semantic
conventions].

No cloud.provider or cloud.platform attribute is reported: the OneAgent runs on
hosts across any infrastructure, so there is no platform to identify.

All three attributes are optional, because which of them the OneAgent writes
depends on its version and configuration. A resource holding only the subset
found in the file is returned without an error, so unlike most detectors this
one never returns [resource.ErrPartialResource]. If the enrichment file does not
exist, i.e. the process is not running on a monitored host, an empty resource is
returned without an error.

These attributes describe the host the process runs on. When telemetry is
forwarded on behalf of another host, prefer the attributes already present on
that telemetry over the ones this detector reports, otherwise the telemetry
appears to originate from the forwarding host.

[semantic conventions]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
*/
package dynatrace
