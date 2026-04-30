PREFIX ?= $(HOME)/.local
REPORTS_DIR ?= reports
GOTESTSUM_VERSION ?= v1.13.0
GITLEAKS_VERSION ?= v8.30.1
GOVULNCHECK_VERSION ?= v1.3.0
GOSEC_VERSION ?= v2.26.1
STATICCHECK_VERSION ?= v0.7.0
RAYCAST_DIR ?= integrations/raycast
GO_PACKAGES ?= $(shell go list ./cmd/... ./internal/... ./tools/...)

.PHONY: test test-race vet staticcheck gosec gitleaks govulncheck fuzz-smoke go-test-junit test-junit trunk-check trunk-fix trunk-flaky-validate raycast-install raycast-test raycast-test-junit raycast-audit raycast-lint raycast-build raycast-verify verify build install-local clean-reports

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

gitleaks:
	go run github.com/zricethezav/gitleaks/v8@$(GITLEAKS_VERSION) detect --source . --redact --no-banner

govulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

fuzz-smoke:
	go test ./internal/inventory -run=^$$ -fuzz=FuzzMCPConfigParsing -fuzztime=10s

go-test-junit:
	mkdir -p $(REPORTS_DIR)
	go run gotest.tools/gotestsum@$(GOTESTSUM_VERSION) --format testname --junitfile $(REPORTS_DIR)/go-tests.raw.xml -- $(GO_PACKAGES)
	go run ./tools/normalize-go-junit $(REPORTS_DIR)/go-tests.raw.xml $(REPORTS_DIR)/go-tests.xml

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

verify: test test-race vet staticcheck gosec gitleaks govulncheck fuzz-smoke test-junit trunk-flaky-validate trunk-check raycast-audit raycast-lint raycast-build

build:
	go build -o bin/nightward ./cmd/nightward
	go build -o bin/nw ./cmd/nw

install-local:
	mkdir -p $(PREFIX)/bin
	go build -o $(PREFIX)/bin/nightward ./cmd/nightward
	go build -o $(PREFIX)/bin/nw ./cmd/nw

clean-reports:
	rm -rf $(REPORTS_DIR)
