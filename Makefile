# Copyright The OpenTelemetry Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

TOOLS_MOD_DIR := ./tools

ALL_DOCS := $(shell find . -name '*.md' -type f | sort)
ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)
OTEL_GO_MOD_DIRS := $(filter-out $(TOOLS_MOD_DIR), $(ALL_GO_MOD_DIRS))
ALL_COVERAGE_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | egrep -v '^./example|^$(TOOLS_MOD_DIR)' | sort)

# URLs to check if all contrib entries exist in the registry.
REGISTRY_BASE_URL = https://raw.githubusercontent.com/open-telemetry/opentelemetry.io/main/content/en/registry
CONTRIB_REPO_URL = https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main

GO = go
GOTEST_MIN = $(GO) test -v -timeout 30s
GOTEST = $(GOTEST_MIN) -race
GOTEST_WITH_COVERAGE = $(GOTEST) -coverprofile=coverage.out -covermode=atomic

.DEFAULT_GOAL := precommit

.PHONY: precommit ci
precommit: dependabot-check license-check misspell go-mod-tidy golangci-lint-fix test
ci: dependabot-check license-check lint vanity-import-check build test check-clean-work-tree

# Tools

.PHONY: tools

TOOLS = $(CURDIR)/.tools

$(TOOLS):
	@mkdir -p $@
$(TOOLS)/%: | $(TOOLS)
	cd $(TOOLS_MOD_DIR) && \
	$(GO) build -o $@ $(PACKAGE)

GOLANGCI_LINT = $(TOOLS)/golangci-lint
$(GOLANGCI_LINT): PACKAGE=github.com/golangci/golangci-lint/cmd/golangci-lint

MISSPELL = $(TOOLS)/misspell
$(MISSPELL): PACKAGE=github.com/client9/misspell/cmd/misspell

GOCOVMERGE = $(TOOLS)/gocovmerge
$(GOCOVMERGE): PACKAGE=github.com/wadey/gocovmerge

ESC = $(TOOLS)/esc
$(ESC): PACKAGE=github.com/mjibson/esc

STRINGER = $(TOOLS)/stringer
$(STRINGER): PACKAGE=golang.org/x/tools/cmd/stringer

MULTIMOD = $(TOOLS)/multimod
$(MULTIMOD): PACKAGE=go.opentelemetry.io/build-tools/multimod

tools: $(GOLANGCI_LINT) $(MISSPELL) $(GOCOVMERGE) $(STRINGER) $(ESC) $(MULTIMOD)

# Build

.PHONY: generate build

generate: $(OTEL_GO_MOD_DIRS:%=generate/%)
generate/%: DIR=$*
generate/%: | $(STRINGER) $(ESC)
	@echo "$(GO) generate $(DIR)/..." \
		&& cd $(DIR) \
		&& PATH="$(TOOLS):$${PATH}" $(GO) generate ./...

build: generate $(OTEL_GO_MOD_DIRS:%=build/%) $(OTEL_GO_MOD_DIRS:%=build-tests/%)
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
golangci-lint/%: | $(GOLANGCI_LINT)
	@echo 'golangci-lint $(if $(ARGS),$(ARGS) ,)$(DIR)' \
		&& cd $(DIR) \
		&& $(GOLANGCI_LINT) run --allow-serial-runners $(ARGS)

.PHONY: go-mod-tidy
go-mod-tidy: $(ALL_GO_MOD_DIRS:%=go-mod-tidy/%)
go-mod-tidy/%: DIR=$*
go-mod-tidy/%:
	@echo "$(GO) mod tidy in $(DIR)" \
		&& cd $(DIR) \
		&& $(GO) mod tidy

.PHONY: misspell
misspell: | $(MISSPELL)
	@$(MISSPELL) -w $(ALL_DOCS)

.PHONY: lint
lint: go-mod-tidy golangci-lint misspell

.PHONY: license-check
license-check:
	@licRes=$$(for f in $$(find . -type f \( -iname '*.go' -o -iname '*.sh' \) ! -path './vendor/*' ! -path './exporters/otlp/internal/opentelemetry-proto/*') ; do \
	           awk '/Copyright The OpenTelemetry Authors|generated|GENERATED/ && NR<=3 { found=1; next } END { if (!found) print FILENAME }' $$f; \
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

.PHONY: dependabot-check
dependabot-check:
	@result=$$( \
		for f in $$( find . -type f -name go.mod -exec dirname {} \; | sed 's/^.\/\?/\//' ); \
			do grep -q "$$f" .github/dependabot.yml \
			|| echo "$$f"; \
		done; \
	); \
	if [ -n "$$result" ]; then \
		echo "missing go.mod dependabot check:"; echo "$$result"; \
		exit 1; \
	fi

# Tests

.PHONY: test-with-coverage
test-with-coverage: | $(GOCOVMERGE)
	set -e; \
	printf "" > coverage.txt; \
	for dir in $(ALL_COVERAGE_MOD_DIRS); do \
	  CMD="$(GOTEST_WITH_COVERAGE)"; \
	  echo "$$dir" | \
	    grep -q 'test$$' && \
	    CMD="$$CMD -coverpkg=go.opentelemetry.io/contrib/$$( dirname "$$dir" | sed -e "s/^\.\///g" )/..."; \
	  echo "$$CMD $$dir/..."; \
	  (cd "$$dir" && \
	    $$CMD ./... && \
	    $(GO) tool cover -html=coverage.out -o coverage.html); \
	done; \
	$(TOOLS_DIR)/gocovmerge $$(find . -name coverage.out) > coverage.txt

.PHONY: test-gocql
test-gocql:
	@if ./tools/should_build.sh gocql; then \
	  set -e; \
	  docker run --name cass-integ --rm -p 9042:9042 -d cassandra:3; \
	  CMD=cassandra IMG_NAME=cass-integ ./tools/wait.sh; \
	  (cd instrumentation/github.com/gocql/gocql/otelgocql/test/ && \
	    $(GOTEST_WITH_COVERAGE) -coverpkg=go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/...  ./... && \
	    $(GO) tool cover -html=coverage.out -o coverage.html); \
	  cp ./instrumentation/github.com/gocql/gocql/otelgocql/test/coverage.out ./; \
	  docker stop cass-integ; \
	fi

.PHONY: test-mongo-driver
test-mongo-driver:
	@if ./tools/should_build.sh mongo-driver; then \
	  set -e; \
	  docker run --name mongo-integ --rm -p 27017:27017 -d mongo; \
	  CMD=mongo IMG_NAME=mongo-integ ./tools/wait.sh; \
	  (cd instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo/test && \
	    $(GOTEST_WITH_COVERAGE) -coverpkg=go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo/...  ./... && \
	    $(GO) tool cover -html=coverage.out -o coverage.html); \
	  cp ./instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo/test/coverage.out ./; \
	  docker stop mongo-integ; \
	fi

.PHONY: test-gomemcache
test-gomemcache:
	@if ./tools/should_build.sh gomemcache; then \
	  set -e; \
	  docker run --name gomemcache-integ --rm -p 11211:11211 -d memcached; \
	  CMD=gomemcache IMG_NAME=gomemcache-integ  ./tools/wait.sh; \
	  (cd instrumentation/github.com/bradfitz/gomemcache/memcache/otelmemcache/test && \
	    $(GOTEST_WITH_COVERAGE) -coverpkg=go.opentelemetry.io/contrib/instrumentation/github.com/bradfitz/gomemcache/memcache/otelmemcache/...  ./... && \
	    $(GO) tool cover -html=coverage.out -o coverage.html); \
	  docker stop gomemcache-integ ; \
	  cp ./instrumentation/github.com/bradfitz/gomemcache/memcache/otelmemcache/test/coverage.out ./; \
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

.PHONY: test
test:
	set -e; for dir in $(OTEL_GO_MOD_DIRS); do \
	  echo "$(GO) test ./... + race in $${dir}"; \
	  (cd "$${dir}" && \
	    $(GOTEST) ./...); \
	done

.PHONY: test-short
test-short:
	set -e; for dir in $(OTEL_GO_MOD_DIRS); do \
	  echo "$(GO) test ./... + race in $${dir}"; \
	  (cd "$${dir}" && \
	    $(GOTEST_MIN) -short ./...); \
	done

# Releasing

COREPATH ?= "../opentelemetry-go"
.PHONY: sync-core
sync-core: | $(MULTIMOD)
	@[ ! -d $COREPATH ] || ( echo ">> Path to core repository must be set in COREPATH and must exist"; exit 1 )
	$(MULTIMOD) verify && $(MULTIMOD) sync -a -o ${COREPATH}


.PHONY: prerelease
prerelease: | $(MULTIMOD)
	@[ "${MODSET}" ] || ( echo ">> env var MODSET is not set"; exit 1 )
	$(MULTIMOD) verify && $(MULTIMOD) prerelease -m ${MODSET}

COMMIT ?= "HEAD"
.PHONY: add-tags
add-tags: | $(MULTIMOD)
	@[ "${MODSET}" ] || ( echo ">> env var MODSET is not set"; exit 1 )
	$(MULTIMOD) verify && $(MULTIMOD) tag -m ${MODSET} -c ${COMMIT}
