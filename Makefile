TOOLS_MOD_DIR := ./tools

# All source code and documents. Used in spell check.
ALL_DOCS := $(shell find . -name '*.md' -type f | sort)
# All directories with go.mod files related to opentelemetry library. Used for building, testing and linting.
ALL_GO_MOD_DIRS := $(filter-out $(TOOLS_MOD_DIR), $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort))
ALL_COVERAGE_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | egrep -v '^./example|^$(TOOLS_MOD_DIR)' | sort)


# Mac OS Catalina 10.5.x doesn't support 386. Hence skip 386 test
SKIP_386_TEST = false
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	SW_VERS := $(shell sw_vers -productVersion)
	ifeq ($(shell echo $(SW_VERS) | egrep '^(10.1[5-9]|1[1-9]|[2-9])'), $(SW_VERS))
		SKIP_386_TEST = true
	endif
endif


GOTEST_MIN = go test -v -timeout 30s
GOTEST = $(GOTEST_MIN) -race
GOTEST_WITH_COVERAGE = $(GOTEST) -coverprofile=coverage.txt -covermode=atomic

.DEFAULT_GOAL := precommit

.PHONY: precommit

TOOLS_DIR := $(abspath ./.tools)

$(TOOLS_DIR)/golangci-lint: $(TOOLS_MOD_DIR)/go.mod $(TOOLS_MOD_DIR)/go.sum $(TOOLS_MOD_DIR)/tools.go
	cd $(TOOLS_MOD_DIR) && \
	go build -o $(TOOLS_DIR)/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

$(TOOLS_DIR)/misspell: $(TOOLS_MOD_DIR)/go.mod $(TOOLS_MOD_DIR)/go.sum $(TOOLS_MOD_DIR)/tools.go
	cd $(TOOLS_MOD_DIR) && \
	go build -o $(TOOLS_DIR)/misspell github.com/client9/misspell/cmd/misspell

$(TOOLS_DIR)/stringer: $(TOOLS_MOD_DIR)/go.mod $(TOOLS_MOD_DIR)/go.sum $(TOOLS_MOD_DIR)/tools.go
	cd $(TOOLS_MOD_DIR) && \
	go build -o $(TOOLS_DIR)/stringer golang.org/x/tools/cmd/stringer

precommit: generate build lint test

.PHONY: test-with-coverage
test-with-coverage:
	set -e; for dir in $(ALL_COVERAGE_MOD_DIRS); do \
	  echo "go test ./... + coverage in $${dir}"; \
	  (cd "$${dir}" && \
	    $(GOTEST_WITH_COVERAGE) ./... && \
	    go tool cover -html=coverage.txt -o coverage.html); \
	done

.PHONY: ci
ci: precommit check-clean-work-tree test-with-coverage test-386

.PHONY: cassandra
cassandra:
	set -e; \
	docker run --name cass-integ --rm -p 9042:9042 -d cassandra:3; \
	for i in 1 2 3 4; do \
    if docker exec cass-integ nodetool status | grep "^UN"; then \
      break; \
    fi; \
    echo "cassandra not yet read..."; \
    sleep 10; \
  done; \
	(cd instrumentation/github.com/gocql/gocql && \
	INTEGRATION=gocql go test); \
	docker stop cass-integ; \

.PHONY: check-clean-work-tree
check-clean-work-tree:
	@if ! git diff --quiet; then \
	  echo; \
	  echo 'Working tree is not clean, did you forget to run "make precommit"?'; \
	  echo; \
	  git status; \
	  exit 1; \
	fi

.PHONY: build
build:
	# TODO: Fix this on windows.
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  echo "compiling all packages in $${dir}"; \
	  (cd "$${dir}" && \
	    go build ./... && \
	    go test -run xxxxxMatchNothingxxxxx ./... >/dev/null); \
	done

.PHONY: test
test:
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  echo "go test ./... + race in $${dir}"; \
	  (cd "$${dir}" && \
	    $(GOTEST) ./...); \
	done

.PHONY: test-386
test-386:
	if [ $(SKIP_386_TEST) = true ] ; then \
		echo "skipping the test for GOARCH 386 as it is not supported on the current OS"; \
  else \
    set -e; for dir in $(ALL_GO_MOD_DIRS); do \
        echo "go test ./... GOARCH 386 in $${dir}"; \
        (cd "$${dir}" && \
          GOARCH=386 $(GOTEST_MIN) ./...); \
	  done; \
	fi

.PHONY: lint
lint: $(TOOLS_DIR)/golangci-lint $(TOOLS_DIR)/misspell
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  echo "golangci-lint in $${dir}"; \
	  (cd "$${dir}" && \
	    $(TOOLS_DIR)/golangci-lint run --fix && \
	    $(TOOLS_DIR)/golangci-lint run); \
	done
	$(TOOLS_DIR)/misspell -w $(ALL_DOCS)
	set -e; for dir in $(ALL_GO_MOD_DIRS) $(TOOLS_MOD_DIR); do \
	  echo "go mod tidy in $${dir}"; \
	  (cd "$${dir}" && \
	    go mod tidy); \
	done

generate: $(TOOLS_DIR)/stringer
	PATH="$(TOOLS_DIR):$${PATH}" go generate ./...
