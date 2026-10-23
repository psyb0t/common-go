MIN_TEST_COVERAGE ?= 10
PKG ?= ./...
export PKG

ifeq ($(COMMON_GO_IN_DEV),1)

all: lint build test ## Lint, build and run all tests

dep: ## Get project dependencies
	@echo "Getting project dependencies..."
	@bash scripts/deps.sh lock

pkg-lock: dep ## Synchronize existing dependency pins and vendor

pkg-add pkg-add-tool pkg-update pkg-upgrade pkg-remove: ## Manage dependencies, PKG=module[@version]
	@bash scripts/deps.sh $(patsubst pkg-%,%,$(subst add-tool,tool,$@))

generate: ## Run all code generation
	@echo "Running code generation..."
	@go generate ./...

format: ## Format database helpers and development scripts
	@gofmt -w db/mysql db/postgresql scripts
	@shfmt -w scripts/*.sh

lint: ## Lint all Golang files
	@echo "Linting all Go files..."
	@out=$$(go fix -diff ./... 2>&1); \
	if [ -n "$$out" ]; then \
		echo "$$out"; \
		echo "go fix found issues. Run 'make lint-fix' to apply."; \
		exit 1; \
	fi
	@go tool golangci-lint run --timeout=30m0s ./...
	@go tool actionlint
	@hadolint Dockerfile.dev
	@shellcheck --source-path=SCRIPTDIR -x scripts/*.sh
	@shfmt -d scripts/*.sh

build: ## Compile every library package
	@go build ./...

sec: ## Scan source, history and dependencies and write sec.sarif
	@bash scripts/sec.sh

audit: sec ## Alias for security checks

lint-fix: ## Lint all Golang files and fix
	@echo "Linting and fixing all Go files..."
	@go fix $(PKG)
	@go tool golangci-lint run --fix --timeout=30m0s $(PKG)

test: ## Run all tests
	@echo "Running all tests..."
	@go test -race -count=1 -timeout=10m $(PKG)

test-coverage: ## Run tests with coverage check. Fails if coverage is below the threshold.
	@echo "Running tests with coverage check..."
	@trap 'rm -f coverage.txt' EXIT; \
	go test -race -count=1 -timeout=10m -coverprofile=coverage.txt $(PKG); \
	if [ $$? -ne 0 ]; then \
		echo "Test failed. Exiting."; \
		exit 1; \
	fi; \
	pct=$$(go tool cover -func=coverage.txt | awk '$$1 == "total:" {gsub(/%/, "", $$3); print $$3}'); \
	if [ -z "$$pct" ]; then \
		echo "FAIL: No test coverage information available."; \
		exit 1; \
	fi; \
	printf '%s\n' "$$pct" > coverage-percent.txt; \
	awk -v actual="$$pct" -v minimum="$(MIN_TEST_COVERAGE)" \
		'BEGIN { printf "Coverage: %s%% (minimum %s%%)\n", actual, minimum; exit(actual < minimum) }'

else

DEV_IMAGE ?= common-go-dev
DEV_RUN = docker run --rm --init $(DEV_TERMINAL_FLAGS) --cap-drop ALL --security-opt no-new-privileges:true \
	--user "$$(id -u):$$(id -g)" -e HOME=/tmp -e PKG \
	-e COMMON_GO_IN_DEV=1 -e GOCACHE=/opt/go-build \
	-v common-go-build-cache:/opt/go-build \
	-v common-go-trivy-cache:/opt/trivy-cache \
	-v "$(CURDIR):$(CURDIR)" -w "$(CURDIR)" $(DEV_IMAGE)
DEV_RUN_INFRA = docker run --rm --init --cap-drop ALL --security-opt no-new-privileges:true \
	--network=host --user "$$(id -u):$$(id -g)" \
	--group-add "$$(stat -c %g /var/run/docker.sock)" \
	-e HOME=/tmp -e COMMON_GO_IN_DEV=1 -e GOCACHE=/opt/go-build \
	-v common-go-build-cache:/opt/go-build \
	-e TESTCONTAINERS_HOST_OVERRIDE=127.0.0.1 \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v "$(CURDIR):$(CURDIR)" -w "$(CURDIR)" $(DEV_IMAGE)

all: lint build test ## Lint, build and run all tests in Docker

dep pkg-lock pkg-add pkg-add-tool pkg-update pkg-upgrade pkg-remove generate format lint lint-fix build sec audit: dev-image
	$(DEV_RUN) make $@

test test-coverage: dev-image
	$(DEV_RUN_INFRA) make $@ PKG='$(PKG)' MIN_TEST_COVERAGE=$(MIN_TEST_COVERAGE)

dev-image: ## Build the development toolchain
	docker build --build-arg DEV_UID="$$(id -u)" --build-arg DEV_GID="$$(id -g)" \
		-f Dockerfile.dev -t $(DEV_IMAGE) .

shell: DEV_TERMINAL_FLAGS = --interactive --tty
shell: dev-image ## Open the development shell without the Docker socket
	$(DEV_RUN) bash

endif

.PHONY: all dep pkg-lock pkg-add pkg-add-tool pkg-update pkg-upgrade pkg-remove generate format lint lint-fix build sec audit test test-coverage dev-image shell help

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_ -]+:.*## / {n=split($$1, names, " "); for (i=1; i<=n; i++) if (!seen[names[i]]++) printf "%-24s %s\n", names[i], $$2}' $(MAKEFILE_LIST)
