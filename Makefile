PREFIX ?= $(HOME)/.local
REPORTS_DIR ?= reports
GOTESTSUM_VERSION ?= v1.13.0
RAYCAST_DIR ?= integrations/raycast
GO_PACKAGES ?= $(shell go list ./cmd/... ./internal/...)

.PHONY: test test-race go-test-junit test-junit trunk-check trunk-fix trunk-flaky-validate raycast-install raycast-test raycast-test-junit raycast-lint raycast-build raycast-verify verify build install-local clean-reports

test:
	go test $(GO_PACKAGES)

test-race:
	go test -race $(GO_PACKAGES)

go-test-junit:
	mkdir -p $(REPORTS_DIR)
	go run gotest.tools/gotestsum@$(GOTESTSUM_VERSION) --format testname --junitfile $(REPORTS_DIR)/go-tests.xml -- $(GO_PACKAGES)

test-junit: clean-reports go-test-junit raycast-install
	cd $(RAYCAST_DIR) && npm run test:junit

trunk-check:
	trunk check --show-existing --all

trunk-fix:
	trunk check --show-existing --fix --all

trunk-flaky-validate:
	trunk flakytests validate --junit-paths $(REPORTS_DIR)/go-tests.xml,$(REPORTS_DIR)/junit/raycast.xml

raycast-install:
	cd $(RAYCAST_DIR) && npm ci --ignore-scripts --no-audit

raycast-test:
	cd $(RAYCAST_DIR) && npm test

raycast-test-junit:
	cd $(RAYCAST_DIR) && npm run test:junit

raycast-lint:
	cd $(RAYCAST_DIR) && npm run lint

raycast-build:
	cd $(RAYCAST_DIR) && npm run build

raycast-verify: raycast-install raycast-test raycast-lint raycast-build

verify: test test-race test-junit trunk-flaky-validate trunk-check raycast-lint raycast-build

build:
	go build -o bin/nightward ./cmd/nightward
	go build -o bin/nw ./cmd/nw

install-local:
	mkdir -p $(PREFIX)/bin
	go build -o $(PREFIX)/bin/nightward ./cmd/nightward
	go build -o $(PREFIX)/bin/nw ./cmd/nw

clean-reports:
	rm -rf $(REPORTS_DIR)
