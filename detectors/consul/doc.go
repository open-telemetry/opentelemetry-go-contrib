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

When no agent can be reached, an empty resource and no error are returned. An
agent that answers without a usable configuration is an error.

When configured with [WithMetaKeyFilter], the detector additionally emits a
consul.meta.<key> attribute for every [node meta] entry whose key satisfies the
configured predicate. Consul is not a cloud provider and has no namespace
reserved in the semantic conventions, so node meta is namespaced under
consul.meta, in the same way the Azure VM detector namespaces VM tags under
azure.tag. The prefix also keeps a node meta entry from colliding with a
detected attribute. Without [WithMetaKeyFilter] no node meta entries are
emitted.

[Consul agent]: https://developer.hashicorp.com/consul/docs/agent
[configuration endpoint]: https://developer.hashicorp.com/consul/api-docs/agent#read-configuration
[node meta]: https://developer.hashicorp.com/consul/docs/reference/agent/configuration-file/node#node_meta
[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
*/
package consul
