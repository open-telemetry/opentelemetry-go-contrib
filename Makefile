# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

TOOLS_MOD_DIR := ./tools

ALL_DOCS := $(shell find . -name '*.md' -type f | sort)
ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)
OTEL_GO_MOD_DIRS := $(filter-out $(TOOLS_MOD_DIR), $(ALL_GO_MOD_DIRS))
ALL_COVERAGE_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | grep -E -v '^./example|^$(TOOLS_MOD_DIR)' | sort)

# URLs to check if all contrib entries exist in the registry.
REGISTRY_BASE_URL = https://raw.githubusercontent.com/open-telemetry/opentelemetry.io/main/content/en/registry
CONTRIB_REPO_URL = https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main

GO = go
TIMEOUT = 60

# User to run as in docker images.
DOCKER_USER=$(shell id -u):$(shell id -g)
DEPENDENCIES_DOCKERFILE=./dependencies.Dockerfile

.DEFAULT_GOAL := precommit

.PHONY: precommit ci
precommit: generate toolchain-check license-check misspell go-mod-tidy golangci-lint-fix test-default
ci: generate toolchain-check license-check lint vanity-import-check build test-default check-clean-work-tree test-coverage

# Tools

.PHONY: tools

TOOLS = $(CURDIR)/.tools

$(TOOLS):
	@mkdir -p $@
$(TOOLS)/%: $(TOOLS_MOD_DIR)/go.mod | $(TOOLS)
	cd $(TOOLS_MOD_DIR) && \
	$(GO) build -o $@ $(PACKAGE)

GOLANGCI_LINT = $(TOOLS)/golangci-lint
$(GOLANGCI_LINT): PACKAGE=github.com/golangci/golangci-lint/cmd/golangci-lint

MISSPELL = $(TOOLS)/misspell
$(MISSPELL): PACKAGE=github.com/client9/misspell/cmd/misspell

GOCOVMERGE = $(TOOLS)/gocovmerge
$(GOCOVMERGE): PACKAGE=github.com/wadey/gocovmerge

STRINGER = $(TOOLS)/stringer
$(STRINGER): PACKAGE=golang.org/x/tools/cmd/stringer

PORTO = $(TOOLS)/porto
$(TOOLS)/porto: PACKAGE=github.com/jcchavezs/porto/cmd/porto

MULTIMOD = $(TOOLS)/multimod
$(MULTIMOD): PACKAGE=go.opentelemetry.io/build-tools/multimod

CROSSLINK = $(TOOLS)/crosslink
$(CROSSLINK): PACKAGE=go.opentelemetry.io/build-tools/crosslink

GOJQ = $(TOOLS)/gojq
$(TOOLS)/gojq: PACKAGE=github.com/itchyny/gojq/cmd/gojq

GOTMPL = $(TOOLS)/gotmpl
$(GOTMPL): PACKAGE=go.opentelemetry.io/build-tools/gotmpl

GORELEASE = $(TOOLS)/gorelease
$(GORELEASE): PACKAGE=golang.org/x/exp/cmd/gorelease

GOJSONSCHEMA = $(TOOLS)/go-jsonschema
$(GOJSONSCHEMA): PACKAGE=github.com/atombender/go-jsonschema

GOVULNCHECK = $(TOOLS)/govulncheck
$(GOVULNCHECK): PACKAGE=golang.org/x/vuln/cmd/govulncheck

tools: $(GOLANGCI_LINT) $(MISSPELL) $(GOCOVMERGE) $(STRINGER) $(PORTO) $(GOJQ) $(MULTIMOD) $(CROSSLINK) $(GOTMPL) $(GORELEASE) $(GOJSONSCHEMA) $(GOVULNCHECK)

# Virtualized python tools via docker

# The directory where the virtual environment is created.
VENVDIR := venv

# The directory where the python tools are installed.
PYTOOLS := $(VENVDIR)/bin

# The pip executable in the virtual environment.
PIP := $(PYTOOLS)/pip

# The directory in the docker image where the current directory is mounted.
WORKDIR := /workdir

# The python image to use for the virtual environment.
PYTHONIMAGE := $(shell awk '$$4=="python" {print $$2}' $(DEPENDENCIES_DOCKERFILE))

# Run the python image with the current directory mounted.
DOCKERPY := docker run --rm -u $(DOCKER_USER) -v "$(CURDIR):$(WORKDIR)" -w $(WORKDIR) $(PYTHONIMAGE)

# Create a virtual environment for Python tools.
$(PYTOOLS):
# The `--upgrade` flag is needed to ensure that the virtual environment is
# created with the latest pip version.
	@$(DOCKERPY) bash -c "python3 -m venv $(VENVDIR) && $(PIP) install --upgrade --cache-dir=$(WORKDIR)/.cache/pip pip"

# Install python packages into the virtual environment.
$(PYTOOLS)/%: $(PYTOOLS)
	@$(DOCKERPY) $(PIP) install --cache-dir=$(WORKDIR)/.cache/pip -r requirements.txt

CODESPELL = $(PYTOOLS)/codespell
$(CODESPELL): PACKAGE=codespell

# Generate

.PHONY: generate
generate: go-generate genjsonschema vanity-import-fix

.PHONY: go-generate
go-generate: $(OTEL_GO_MOD_DIRS:%=go-generate/%)
go-generate/%: DIR=$*
go-generate/%: $(STRINGER) $(GOTMPL)
	@echo "$(GO) generate $(DIR)/..." \
		&& cd $(DIR) \
		&& PATH="$(TOOLS):$${PATH}" $(GO) generate ./...

.PHONY: vanity-import-fix
vanity-import-fix: $(PORTO)
	@$(PORTO) --include-internal -w .

# Generate go.work file for local development.
.PHONY: go-work
go-work: $(CROSSLINK)
	$(CROSSLINK) work --root=$(shell pwd)

# Build

.PHONY: build

build: $(OTEL_GO_MOD_DIRS:%=build/%) $(OTEL_GO_MOD_DIRS:%=build-tests/%)
build/%: DIR=$*
build/%:
	@echo "$(GO) build $(DIR)/..." \
		&& cd $(DIR) \
		&& $(GO) build ./...

build-tests/%: DIR=$*
build-tests/%:
	@echo "$(GO) build tests $(DIR)/..." \
		&& cd $(DIR) \
		&& $(GO) list ./... \
		| grep -v third_party \
		| xargs $(GO) test -vet=off -run xxxxxMatchNothingxxxxx >/dev/null

# Linting

.PHONY: golangci-lint golangci-lint-fix
golangci-lint-fix: ARGS=--fix
golangci-lint-fix: golangci-lint
golangci-lint: $(OTEL_GO_MOD_DIRS:%=golangci-lint/%)
golangci-lint/%: DIR=$*
golangci-lint/%: $(GOLANGCI_LINT)
	@echo 'golangci-lint $(if $(ARGS),$(ARGS) ,)$(DIR)' \
		&& cd $(DIR) \
		&& $(GOLANGCI_LINT) run --allow-serial-runners $(ARGS)

.PHONY: crosslink
crosslink: $(CROSSLINK)
	@echo "Updating intra-repository dependencies in all go modules" \
		&& $(CROSSLINK) --root=$(shell pwd) --prune

.PHONY: go-mod-tidy
go-mod-tidy: $(ALL_GO_MOD_DIRS:%=go-mod-tidy/%)
go-mod-tidy/%: DIR=$*
go-mod-tidy/%:
	@echo "$(GO) mod tidy in $(DIR)" \
		&& cd $(DIR) \
		&& $(GO) mod tidy -compat=1.21

.PHONY: misspell
misspell: $(MISSPELL)
	@$(MISSPELL) -w $(ALL_DOCS)

.PHONY: govulncheck
govulncheck: $(ALL_GO_MOD_DIRS:%=govulncheck/%)
govulncheck/%: DIR=$*
govulncheck/%: $(GOVULNCHECK)
	@echo "govulncheck in $(DIR)" \
		&& cd $(DIR) \
		&& $(GOVULNCHECK) ./...

.PHONY: vanity-import-check
vanity-import-check: | $(PORTO)
	@$(PORTO) --include-internal -l . || ( echo "(run: make vanity-import-fix)"; exit 1 )

.PHONY: lint
lint: go-mod-tidy golangci-lint misspell govulncheck

.PHONY: toolchain-check
toolchain-check:
	@toolchainRes=$$(for f in $(ALL_GO_MOD_DIRS); do \
	           awk '/^toolchain/ { found=1; next } END { if (found) print FILENAME }' $$f/go.mod; \
	done); \
	if [ -n "$${toolchainRes}" ]; then \
			echo "toolchain checking failed:"; echo "$${toolchainRes}"; \
			exit 1; \
	fi

.PHONY: license-check
license-check:
	@licRes=$$(for f in $$(find . -type f \( -iname '*.go' -o -iname '*.sh' \) ! -path './vendor/*' ! -path './exporters/otlp/internal/opentelemetry-proto/*') ; do \
	           awk '/Copyright The OpenTelemetry Authors|generated|GENERATED/ && NR<=4 { found=1; next } END { if (!found) print FILENAME }' $$f; \
	   done); \
	   if [ -n "$${licRes}" ]; then \
	           echo "license header checking failed:"; echo "$${licRes}"; \
	           exit 1; \
	   fi

.PHONY: registry-links-check
registry-links-check:
	@checkRes=$$( \
		for f in $$( find ./instrumentation ./exporters ./detectors ! -path './instrumentation/net/*' -type f -name 'go.mod' -exec dirname {} \; | egrep -v '/example|/utils' | sort ) \
			./instrumentation/net/http; do \
			TYPE="instrumentation"; \
			if $$(echo "$$f" | grep -q "exporters"); then \
				TYPE="exporter"; \
			fi; \
			if $$(echo "$$f" | grep -q "detectors"); then \
				TYPE="detector"; \
			fi; \
			NAME=$$(echo "$$f" | sed -e 's/.*\///' -e 's/.*otel//'); \
			LINK=$(CONTRIB_REPO_URL)/$$(echo "$$f" | sed -e 's/..//' -e 's/\/otel.*$$//'); \
			if ! $$(curl -s $(REGISTRY_BASE_URL)/$${TYPE}-go-$${NAME}.md | grep -q "$${LINK}"); then \
				echo "$$f"; \
			fi \
		done; \
	); \
	if [ -n "$$checkRes" ]; then \
		echo "WARNING: registry link check failed for the following packages:"; echo "$${checkRes}"; \
	fi

.PHONY: check-clean-work-tree
check-clean-work-tree:
	@if ! git diff --quiet; then \
	  echo; \
	  echo 'Working tree is not clean, did you forget to run "make precommit"?'; \
	  echo; \
	  git status; \
	  exit 1; \
	fi

# Tests

TEST_TARGETS := test-default test-bench test-short test-verbose test-race
.PHONY: $(TEST_TARGETS) test
test-default test-race: ARGS=-race
test-bench:   ARGS=-run=xxxxxMatchNothingxxxxx -test.benchtime=1ms -bench=.
test-short:   ARGS=-short
test-verbose: ARGS=-v
$(TEST_TARGETS): test
test: $(OTEL_GO_MOD_DIRS:%=test/%)
test/%: DIR=$*
test/%:
	@echo "$(GO) test -timeout $(TIMEOUT)s $(ARGS) $(DIR)/..." \
		&& cd $(DIR) \
		&& $(GO) test -timeout $(TIMEOUT)s $(ARGS) ./...

COVERAGE_MODE    = atomic
COVERAGE_PROFILE = coverage.out
.PHONY: test-coverage
test-coverage: $(ALL_COVERAGE_MOD_DIRS:%=test-coverage/%) | $(GOCOVMERGE)
	@printf "" > coverage.txt \
		&& $(GOCOVMERGE) $$(find . -name $(COVERAGE_PROFILE)) > coverage.txt

test-coverage/%: DIR=$*
test-coverage/%:
	@set -e; \
		CMD="$(GO) test -race -covermode=$(COVERAGE_MODE) -coverprofile=$(COVERAGE_PROFILE)"; \
		echo "$(DIR)" | grep -q 'test$$' \
		&& CMD="$$CMD -coverpkg=go.opentelemetry.io/contrib/$$( dirname "$(DIR)" | sed -e "s/^\.\///g" )/..."; \
		echo "$$CMD $(DIR)/..."; \
		cd "$(DIR)" \
		&& $$CMD ./... \
		&& $(GO) tool cover -html=coverage.out -o coverage.html;

# Releasing

.PHONY: gorelease
gorelease: $(OTEL_GO_MOD_DIRS:%=gorelease/%)
gorelease/%: DIR=$*
gorelease/%: $(GORELEASE)
	@echo "gorelease in $(DIR):" \
		&& cd $(DIR) \
		&& $(GORELEASE) \
		|| echo ""

COREPATH ?= "../opentelemetry-go"
.PHONY: sync-core
sync-core: $(MULTIMOD)
	@[ ! -d $COREPATH ] || ( echo ">> Path to core repository must be set in COREPATH and must exist"; exit 1 )
	$(MULTIMOD) verify && $(MULTIMOD) sync -a -o ${COREPATH}

.PHONY: prerelease
prerelease: $(MULTIMOD)
	@[ "${MODSET}" ] || ( echo ">> env var MODSET is not set"; exit 1 )
	$(MULTIMOD) verify && $(MULTIMOD) prerelease -m ${MODSET}

COMMIT ?= "HEAD"
.PHONY: add-tags
add-tags: $(MULTIMOD)
	@[ "${MODSET}" ] || ( echo ">> env var MODSET is not set"; exit 1 )
	$(MULTIMOD) verify && $(MULTIMOD) tag -m ${MODSET} -c ${COMMIT}

.PHONY: update-all-otel-deps
update-all-otel-deps:
	@[ "${GITSHA}" ] || ( echo ">> env var GITSHA is not set"; exit 1 )
	@echo "Updating OpenTelemetry dependencies to ${GITSHA}"
	@set -e; \
		for dir in $(OTEL_GO_MOD_DIRS); do \
			echo "Updating OpenTelemetry dependencies in $${dir}"; \
			(cd $${dir} \
			&& grep -o 'go.opentelemetry.io/otel\S*' go.mod | xargs -I {} -n1 $(GO) get {}@${GITSHA}); \
		done

# The source directory for opentelemetry-configuration schema.
OPENTELEMETRY_CONFIGURATION_JSONSCHEMA_SRC_DIR=tmp/opentelemetry-configuration

# The SHA matching the current version of the opentelemetry-configuration schema to use
OPENTELEMETRY_CONFIGURATION_JSONSCHEMA_VERSION=v0.3.0

# Cleanup temporary directory
genjsonschema-cleanup:
	rm -Rf ${OPENTELEMETRY_CONFIGURATION_JSONSCHEMA_SRC_DIR}

GENERATED_CONFIG=./config/${OPENTELEMETRY_CONFIGURATION_JSONSCHEMA_VERSION}/generated_config.go

# Generate structs for configuration from opentelemetry-configuration schema
genjsonschema: genjsonschema-cleanup $(GOJSONSCHEMA)
	mkdir -p ${OPENTELEMETRY_CONFIGURATION_JSONSCHEMA_SRC_DIR}
	mkdir -p ./config/${OPENTELEMETRY_CONFIGURATION_JSONSCHEMA_VERSION}
	curl -sSL https://api.github.com/repos/open-telemetry/opentelemetry-configuration/tarball/${OPENTELEMETRY_CONFIGURATION_JSONSCHEMA_VERSION} | tar xz --strip 1 -C ${OPENTELEMETRY_CONFIGURATION_JSONSCHEMA_SRC_DIR}
	$(GOJSONSCHEMA) \
		--capitalization ID \
		--capitalization OTLP \
		--struct-name-from-title \
		--package config \
		--only-models \
		--output ${GENERATED_CONFIG} \
		${OPENTELEMETRY_CONFIGURATION_JSONSCHEMA_SRC_DIR}/schema/opentelemetry_configuration.json
	@echo Modify jsonschema generated files.
	sed -f ./config/jsonschema_patch.sed ${GENERATED_CONFIG} > ${GENERATED_CONFIG}.tmp
	mv ${GENERATED_CONFIG}.tmp ${GENERATED_CONFIG}
	$(MAKE) genjsonschema-cleanup

.PHONY: codespell
codespell: $(CODESPELL)
	@$(DOCKERPY) $(CODESPELL)
