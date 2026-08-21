// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package digitalocean provides a [resource.Detector] which supports detecting
attributes specific to DigitalOcean Droplets.

According to semantic conventions for [cloud] and [host] attributes,
each of the following attributes is added if it is available:

  - cloud.provider
  - cloud.region
  - host.id
  - host.name

The cloud.provider value is reported as "digitalocean". The semantic
conventions do not define a DigitalOcean value yet, so it is not a generated
constant.

Neither cloud.platform nor host.type is reported. No cloud.platform value is
standardized for DigitalOcean, and the metadata service does not report the size
of the Droplet.

The metadata client used to reach the metadata service does not accept a
[context.Context]. The request therefore runs in its own goroutine, and the
context passed to Detect only cancels the wait for it: the goroutine can outlive
the call, bounded by the timeout of the HTTP client.

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
*/
package digitalocean
