PREFIX ?= $(HOME)/.local
REPORTS_DIR ?= reports
GOTESTSUM_VERSION ?= v1.13.0
GOSEC_VERSION ?= v2.26.1
STATICCHECK_VERSION ?= v0.7.0
RAYCAST_DIR ?= integrations/raycast
GO_PACKAGES ?= $(shell go list ./cmd/... ./internal/...)

.PHONY: test test-race vet staticcheck gosec go-test-junit test-junit trunk-check trunk-fix trunk-flaky-validate raycast-install raycast-test raycast-test-junit raycast-audit raycast-lint raycast-build raycast-verify verify build install-local clean-reports

test:
	go test $(GO_PACKAGES)

test-race:
	go test -race $(GO_PACKAGES)

vet:
	go vet $(GO_PACKAGES)

staticcheck:
	go run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) $(GO_PACKAGES)

gosec:
	go run github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION) -exclude-generated -exclude-dir=$(RAYCAST_DIR)/node_modules -exclude-dir=$(RAYCAST_DIR)/dist ./...

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

raycast-audit:
	cd $(RAYCAST_DIR) && npm audit --audit-level=moderate

raycast-lint:
	cd $(RAYCAST_DIR) && npm run lint

raycast-build:
	cd $(RAYCAST_DIR) && npm run build

raycast-verify: raycast-install raycast-test raycast-audit raycast-lint raycast-build

verify: test test-race vet staticcheck gosec test-junit trunk-flaky-validate trunk-check raycast-audit raycast-lint raycast-build

build:
	go build -o bin/nightward ./cmd/nightward
	go build -o bin/nw ./cmd/nw

install-local:
	mkdir -p $(PREFIX)/bin
	go build -o $(PREFIX)/bin/nightward ./cmd/nightward
	go build -o $(PREFIX)/bin/nw ./cmd/nw

clean-reports:
	rm -rf $(REPORTS_DIR)
