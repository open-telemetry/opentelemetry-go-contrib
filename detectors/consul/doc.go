// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package consul provides a [resource.Detector] which supports detecting
attributes from a [Consul agent].

The detector reads the agent [configuration endpoint]. According to semantic
conventions for [cloud] and [host] attributes, each of the following attributes
is added if it is available:

  - cloud.region
  - host.id
  - host.name

The cloud.region value is the Consul datacenter the agent belongs to.

When configured with [WithMetaKeyFilter], the detector additionally emits an
attribute for every [node meta] entry whose key satisfies the configured
predicate. Consul is not a cloud provider and has no namespace reserved in the
semantic conventions, so node meta keys are emitted verbatim, matching the
Consul detector of the OpenTelemetry Collector. A node meta key that collides
with a detected attribute, for example "host.name", overrides it. Without
[WithMetaKeyFilter] no node meta entries are emitted.

[Consul agent]: https://developer.hashicorp.com/consul/docs/agent
[configuration endpoint]: https://developer.hashicorp.com/consul/api-docs/agent#read-configuration
[node meta]: https://developer.hashicorp.com/consul/docs/reference/agent/configuration-file/node#node_meta
[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
*/
package consul
