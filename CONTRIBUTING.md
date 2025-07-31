# Contributing to opentelemetry-go-contrib

Welcome to the OpenTelemetry Go contrib repository!
Thank you for your interest in contributing to this project.
Before you start please be sure to read through these contributing requirements and recommendations.

## Become a Contributor

All contributions to this project MUST be licensed under this project's [license](LICENSE).
You will need to sign the [CNCF CLA](https://identity.linuxfoundation.org/projects/cncf) before your contributions will be accepted.

## Code Owners

To ensure code that lives in this repository is not abandoned, all modules added are required to have a Code Owner.
A Code Owner is responsible for a module within this repository.
This status is identified in the [CODEOWNERS file](./CODEOWNERS).
That responsibility includes maintaining the component, triaging and responding to issues, and reviewing pull requests.

### Requirements

To become a Code Owner, you will need to meet the following requirements.

1. You will need to be a [member of the OpenTelemetry organization] and maintain that membership.
2. You need to have good working knowledge of the code you are sponsoring and any project that that code instruments or is based on.

If you are not an existing member, this is not an immediate disqualification.
You will need to engage with the OpenTelemetry community so you can achieve this membership in the process of becoming a Code Owner.

It is best to have resolved at least an issue related to the module, contributed directly to the module, and/or reviewed module PRs.
How much interaction with the module is required before becoming a Code Owner is up to the existing Code Owners.

Code Ownership is ultimately up to the judgement of the existing Code Owners and Maintainers of this repository.
Meeting the above requirements is not a guarantee to be granted Code Ownership.

[member of the OpenTelemetry organization]: https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#member

### Responsibilities

As a Code Owner you will be responsible for the following:

- You will be responsible for keeping up with the instrumented library. Any "upstream" changes that impact this module need to be proactively handle by you.
- You will be expected to review any Pull Requests or Issues created that relate to this module.
- You will be responsible for the stability and versioning compliance of the module.
- You will be responsible for deciding any additional Code Owners of the module.

### How to become a Code Owner

To become a Code Owner, open [an Issue](https://github.com/open-telemetry/opentelemetry-go-contrib/issues/new?assignees=&labels=&projects=&template=owner.md&title=).

### Removing Code Ownership

Code Owners are expected to remove their ownership if they cannot fulfill their responsibilities anymore.

It is at the discretion of the repository Maintainers and fellow Code Owners to decide if a Code Owner should be considered for removal.
If a Code Owner is determined to be unable to perform their duty, a repository Maintainer will remove their ownership.

Inactivity greater than 5 months, during which time there are active Issues or Pull Requests to address, is deemed an automatic disqualification from being a Code Owner.
A repository Maintainer may remove an Code Owner inactive for this length.

## Filing Issues

Sensitive security-related issues should be reported to <cncf-opentelemetry-tc@lists.cncf.io>. See the [security policy](https://github.com/open-telemetry/opentelemetry-go-contrib/security/policy) for details.

When reporting bugs, please be sure to include the following.

- What version of Go and opentelemetry-go-contrib are you using?
- What operating system and processor architecture are you using?
- What did you do?
- What did you expect to see?
- What did you see instead?

For instrumentation requests, please see the [instrumentation documentation](./instrumentation/README.md#new-instrumentation).

## Contributing Code

The project welcomes code patches, but to make sure things are well coordinated you should discuss any significant change before starting the work.
It's recommended that you signal your intention to contribute in the issue tracker, either by [filing a new issue](https://github.com/open-telemetry/opentelemetry-go-contrib/issues/new) or by claiming an [existing one](https://github.com/open-telemetry/opentelemetry-go-contrib/issues).

Please follow [Contributing to opentelemetry-go](https://github.com/open-telemetry/opentelemetry-go/blob/main/CONTRIBUTING.md).

## New Component

**Do not submit pull requests for new components without reading the following.**

This project is dedicated to promoting the development of quality components, such as instrumentation libraries, bridges, detectors, propagators, samplers, processors, using OpenTelemetry.
To achieve this goal, we recognize that the components needs to be written using the best practices of OpenTelemetry.
Additionally, the produced component needs to be maintained and evolved.

The size of the OpenTelemetry Go developer community is not large enough to support an ever growing amount of components.
Therefore, to reach our goal, we have the following recommendations for where components should live.

1. Native to the instrumented package
2. A dedicated public repository
3. Here in the opentelemetry-go-contrib repository

If possible, OpenTelemetry instrumentation should be included in the instrumented package.
This will ensure the instrumentation reaches all package users, and is continuously maintained by developers that understand the package.

If component cannot be directly included in the package it is related to, it should be hosted in a dedicated public repository owned by its maintainer(s).
This will appropriately assign maintenance responsibilities for the instrumentation and ensure these maintainers have the needed privilege to maintain the code.

The last place component should be hosted is here in this repository as a separate Go module.
Maintaining components here hampers the development of OpenTelemetry for Go and therefore should be avoided.
When instrumentation cannot be included in a target package and there is good reason to not host it in a separate and dedicated repository a [new component or instrumentation request](https://github.com/open-telemetry/opentelemetry-go-contrib/issues/new/choose) should be filed.
The request needs to be accepted before any pull requests for the component can be considered for merging.

Regardless of where component is hosted, it needs to be discoverable.
The [OpenTelemetry registry](https://opentelemetry.io/registry/)
exists to ensure that component is discoverable.
You can find out how to add component to the registry [here](https://github.com/open-telemetry/opentelemetry.io#adding-a-project-to-the-opentelemetry-registry).

## Approvers and Maintainers

### Maintainers

- [Damien Mathieu](https://github.com/dmathieu), Elastic
- [David Ashpole](https://github.com/dashpole), Google
- [Robert Pająk](https://github.com/pellared), Splunk
- [Sam Xie](https://github.com/XSAM), Cisco/AppDynamics
- [Tyler Yahn](https://github.com/MrAlias), Splunk

For more information about the maintainer role, see the [community repository](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#maintainer).

### Approvers

- [Flc](https://github.com/flc1125), Independent

For more information about the approver role, see the [community repository](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#approver).

### Triagers

- [Alex Kats](https://github.com/akats7), Capital One
- [Cheng-Zhen Yang](https://github.com/scorpionknifes), Independent

For more information about the triager role, see the [community repository](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#triager).

### Emeritus

- [Aaron Clawson](https://github.com/MadVikingGod)
- [Anthony Mirabella](https://github.com/Aneurysm9)
- [Chester Cheung](https://github.com/hanyuancheung)
- [Evan Torrie](https://github.com/evantorrie)
- [Gustavo Silva Paiva](https://github.com/paivagustavo)
- [Josh MacDonald](https://github.com/jmacd)

For more information about the emeritus role, see the [community repository](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#emeritus-maintainerapprovertriager).

### Become an Approver or a Maintainer

See the [community membership document in OpenTelemetry community
repo](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md).
