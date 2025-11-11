# OpenTelemetry-Go Contrib

[![build\_and\_test](https://github.com/open-telemetry/opentelemetry-go-contrib/workflows/build_and_test/badge.svg)](https://github.com/open-telemetry/opentelemetry-go-contrib/actions?query=workflow%3Abuild_and_test+branch%3Amain)
[![codecov.io](https://codecov.io/gh/open-telemetry/opentelemetry-go-contrib/coverage.svg?branch=main)](https://app.codecov.io/gh/open-telemetry/opentelemetry-go-contrib?branch=main)
[![Docs](https://godoc.org/go.opentelemetry.io/contrib?status.svg)](https://pkg.go.dev/go.opentelemetry.io/contrib)
[![Go Report Card](https://goreportcard.com/badge/go.opentelemetry.io/contrib)](https://goreportcard.com/report/go.opentelemetry.io/contrib)
[![Fuzzing Status](https://oss-fuzz-build-logs.storage.googleapis.com/badges/opentelemetry-go-contrib.svg)](https://issues.oss-fuzz.com/issues?q=project:opentelemetry-go-contrib)
[![Slack](https://img.shields.io/badge/slack-@cncf/otel--go-brightgreen.svg?logo=slack)](https://cloud-native.slack.com/archives/C01NPAXACKT)

Collection of 3rd-party packages for [OpenTelemetry-Go](https://github.com/open-telemetry/opentelemetry-go).

---

## Contents

* **[Examples](./examples/)** — Example applications and snippets demonstrating usage.
* **[Instrumentation](./instrumentation/)** — Instrumentation packages for 3rd‑party libraries.
* **[Propagators](./propagators/)** — Context propagation implementations for common formats.
* **[Detectors](./detectors/)** — Resource detectors for cloud environments.
* **[Exporters](./exporters/)** — Exporters for 3rd‑party telemetry backends.
* **[Samplers](./samplers/)** — Additional sampler implementations.
* **[Bridges](./bridges/)** — Adapters for other instrumentation frameworks.
* **[Processors](./processors/)** — Extra processor implementations.

---

## Quickstart

> These steps get a contributor setup quickly to run and test packages locally.

1. **Install Go** (matching supported versions, see Compatibility below). Check with:

   ```bash
   go version
   ```

2. **Clone your fork** (recommended fork workflow for contributing):

   ```bash
   git clone git@github.com:YOUR-USERNAME/opentelemetry-go-contrib.git
   cd opentelemetry-go-contrib
   git remote add upstream https://github.com/open-telemetry/opentelemetry-go-contrib.git
   ```

3. **Sync your main**:

   ```bash
   git checkout main
   git fetch upstream
   git pull --rebase upstream main
   ```

4. **Run tests for a package** (recommended to run only affected packages locally):

   ```bash
   go test ./instrumentation/<package>/...
   # or run all tests (can be slow):
   go test ./...
   ```

5. **Create a branch, make changes, and open a PR** following `CONTRIBUTING.md`.

---

## Running CI & Tests Locally

This repository uses the GitHub Actions `build_and_test` workflow. To reproduce common checks locally:

* Run unit tests:

  ```bash
  go test ./... -v
  ```

* Run package-specific tests when working in a subdirectory:

  ```bash
  cd instrumentation/<pkg>
  go test ./...
  ```

* If a `Makefile` target exists (some packages may provide one), use it:

  ```bash
  make test
  ```

---

## How to Contribute

Please read the repository's **[CONTRIBUTING.md](./CONTRIBUTING.md)** before submitting a PR. A short checklist:

1. Fork the repository and clone your fork.
2. Sync `main` with upstream before starting work.
3. Create a small, focused branch: `git checkout -b feat/<short-description>`.
4. Run tests for the package you change and keep changes well-scoped.
5. Write clear commit messages and a helpful PR description describing what, why, and how to test.
6. Link issues (e.g. `closes #123`) when applicable.
7. Be responsive to reviewer feedback and update your branch as requested.

Maintainers may request a DCO/CLA or additional steps — see `CONTRIBUTING.md` for details.

---

## Compatibility

OpenTelemetry-Go Contrib supports the Go language versions tested in CI. The project follows the Go release support policy: each major release is supported until two newer major releases exist.

| OS      | Go Versions | Architectures |
| ------- | ----------- | ------------- |
| Ubuntu  | 1.24, 1.25  | amd64, 386    |
| macOS   | 1.24, 1.25  | amd64, arm64  |
| Windows | 1.24, 1.25  | amd64, 386    |

---

## Project Status & Governance

This repository contains stable and unstable modules. Check the module's own version and the `versions.yaml` manifest for versioning details.

* Project progress and planning: see the **[project boards](https://github.com/open-telemetry/opentelemetry-go-contrib/projects)** and **[milestones](https://github.com/open-telemetry/opentelemetry-go-contrib/milestones)**.
* For versioning and stability guarantees, see the upstream OpenTelemetry-Go [`VERSIONING.md`](https://github.com/open-telemetry/opentelemetry-go/blob/main/VERSIONING.md).

---

## Finding Issues to Work On

* Use the **Issues** tab and filter by labels like `good first issue`, `help wanted`, or `documentation`.
* Start with documentation fixes or small test/bug fixes to learn the codebase and CI.

---

## Useful Links

* Main repo: [https://github.com/open-telemetry/opentelemetry-go-contrib](https://github.com/open-telemetry/opentelemetry-go-contrib)
* Upstream OpenTelemetry-Go: [https://github.com/open-telemetry/opentelemetry-go](https://github.com/open-telemetry/opentelemetry-go)
* Docs: [https://pkg.go.dev/go.opentelemetry.io/contrib](https://pkg.go.dev/go.opentelemetry.io/contrib)
* CI actions: [https://github.com/open-telemetry/opentelemetry-go-contrib/actions](https://github.com/open-telemetry/opentelemetry-go-contrib/actions)

---

## License

This repository is licensed under the terms specified in the repository — see the `LICENSE` file for details.

---

If you'd like, I can also:

* Produce a PR-ready commit message and branch name for this README change.
* Create a small example `CONTRIBUTING.md` checklist or a `README_CONTRIBUTING.md` that highlights the most important points for first-time contributors.
* Scan the repo issues now and list 3 beginner-friendly issues you can pick.

Tell me which one you want next.
