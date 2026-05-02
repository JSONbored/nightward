PREFIX ?= $(HOME)/.local
REPORTS_DIR ?= reports
GOTESTSUM_VERSION ?= v1.13.0
GITLEAKS_VERSION ?= v8.30.1
GOVULNCHECK_VERSION ?= v1.3.0
GOSEC_VERSION ?= v2.26.1
STATICCHECK_VERSION ?= v0.7.0
GORELEASER_VERSION ?= v2.9.0
SYFT_VERSION ?= v1.43.0
COVERAGE_THRESHOLD ?= 83.0
RAYCAST_DIR ?= integrations/raycast
NPM_PACKAGE_DIR ?= packages/npm
SITE_DIR ?= site
GO_PACKAGES ?= $(shell go list ./cmd/... ./internal/... ./tools/...)
COVERAGE_PACKAGES ?= ./internal/...

.PHONY: test test-fast test-security test-ux test-release test-local test-prepush test-release-install test-race vet staticcheck gosec gitleaks govulncheck fuzz-smoke fuzz-test coverage coverage-check go-test-junit test-junit trunk-check trunk-fix trunk-flaky-validate ci-scripts-test raycast-install raycast-test raycast-test-junit raycast-audit raycast-lint raycast-build raycast-store-check raycast-verify npm-package-install npm-package-test npm-package-audit npm-package-pack npm-package-verify docs-reference docs-reference-check docs-freshness docs-qa site-install site-audit site-build site-verify tui-demo demo-assets tool-syft release-snapshot verify build install-local clean-reports

test:
	go test $(GO_PACKAGES)

test-fast: test npm-package-test raycast-test

test-security: vet staticcheck gosec gitleaks govulncheck npm-package-audit raycast-audit site-audit

test-ux: raycast-verify site-verify

test-release: ci-scripts-test npm-package-verify raycast-build site-build release-snapshot

test-local: verify

test-prepush: verify

test-release-install:
	@if [ -z "$${VERSION:-}" ]; then echo "VERSION is required, for example: make test-release-install VERSION=0.1.4" >&2; exit 2; fi
	bash scripts/verify-npm-release.sh "$${VERSION}"

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

fuzz-test: fuzz-smoke

coverage:
	mkdir -p $(REPORTS_DIR)
	go test $(COVERAGE_PACKAGES) -coverprofile=$(REPORTS_DIR)/coverage.out
	go tool cover -func=$(REPORTS_DIR)/coverage.out | tee $(REPORTS_DIR)/coverage.txt

coverage-check:
	mkdir -p $(REPORTS_DIR)
	go test $(COVERAGE_PACKAGES) -coverprofile=$(REPORTS_DIR)/coverage.out
	go tool cover -func=$(REPORTS_DIR)/coverage.out | tee $(REPORTS_DIR)/coverage.txt
	python3 -c 'import pathlib, re, sys; text=pathlib.Path("$(REPORTS_DIR)/coverage.txt").read_text(); match=re.search(r"total:\s+\(statements\)\s+([0-9.]+)%", text); pct=float(match.group(1)) if match else -1; threshold=float("$(COVERAGE_THRESHOLD)"); print(f"coverage {pct:.1f}% / threshold {threshold:.1f}%"); sys.exit(0 if pct >= threshold else 1)'

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

ci-scripts-test:
	bash scripts/test-dco.sh
	bash scripts/test-action-paths.sh
	bash scripts/test-release-scripts.sh

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

raycast-store-check:
	cd $(RAYCAST_DIR) && npm run store-check

raycast-verify: raycast-install raycast-test raycast-audit raycast-lint raycast-build raycast-store-check

npm-package-install:
	cd $(NPM_PACKAGE_DIR) && npm ci --ignore-scripts --no-audit

npm-package-test:
	cd $(NPM_PACKAGE_DIR) && npm test

npm-package-audit:
	cd $(NPM_PACKAGE_DIR) && npm audit --audit-level=moderate

npm-package-pack:
	cd $(NPM_PACKAGE_DIR) && npm run pack:dry-run

npm-package-verify: npm-package-install npm-package-test npm-package-audit npm-package-pack

site-install:
	cd $(SITE_DIR) && npm ci --ignore-scripts --no-audit

site-audit:
	cd $(SITE_DIR) && npm audit --audit-level=moderate

site-build:
	cd $(SITE_DIR) && npm run build

docs-reference:
	node scripts/generate-reference-docs.mjs

docs-reference-check:
	node scripts/generate-reference-docs.mjs --check

docs-freshness:
	node scripts/check-docs-freshness.mjs

docs-qa: docs-reference-check docs-freshness

site-verify: docs-qa site-install site-audit site-build

tui-demo:
	PATH="$(HOME)/go/bin:$(PATH)" vhs docs/demo/nightward-tui.tape

demo-assets:
	node scripts/generate-demo-assets.mjs

tool-syft:
	go install github.com/anchore/syft/cmd/syft@$(SYFT_VERSION)

release-snapshot: tool-syft
	PATH="$$(go env GOPATH)/bin:$$PATH" go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION) release --snapshot --clean --skip=publish,sign

verify: test test-race vet staticcheck gosec gitleaks govulncheck fuzz-smoke coverage-check test-junit trunk-flaky-validate trunk-check ci-scripts-test raycast-audit raycast-lint raycast-build npm-package-verify site-verify

build:
	go build -o bin/nightward ./cmd/nightward
	go build -o bin/nw ./cmd/nw

install-local:
	mkdir -p $(PREFIX)/bin
	go build -o $(PREFIX)/bin/nightward ./cmd/nightward
	go build -o $(PREFIX)/bin/nw ./cmd/nw

clean-reports:
	rm -rf $(REPORTS_DIR)
