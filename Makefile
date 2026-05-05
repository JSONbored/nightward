PREFIX ?= $(HOME)/.local
REPORTS_DIR ?= reports
COVERAGE_THRESHOLD ?= 83.0
RAYCAST_DIR ?= integrations/raycast
NPM_PACKAGE_DIR ?= packages/npm
SITE_DIR ?= site
RUST_PATH ?= $(HOME)/.cargo/bin:/opt/homebrew/bin:$(PATH)
CARGO ?= env PATH="$(RUST_PATH)" cargo
RUST_BINS ?= nightward nw
RUSTUP ?= env PATH="$(RUST_PATH)" rustup
CARGO_DEV_TOOLCHAIN ?= 1.88.0
CARGO_AUDIT_VERSION ?= 0.22.1
CARGO_DENY_VERSION ?= 0.19.4
CARGO_LLVM_COV_VERSION ?= 0.8.5

.PHONY: doctor install-dev-tools test test-fast test-security test-ux test-release test-local test-prepush test-release-install fmt clippy cargo-test cargo-nextest cargo-doc cargo-audit cargo-deny cargo-llvm-cov coverage-check fuzz-smoke test-junit trunk-check trunk-fix trunk-flaky-validate ci-scripts-test gitleaks raycast-install raycast-test raycast-test-junit raycast-audit raycast-lint raycast-build raycast-store-check raycast-verify npm-package-install npm-package-test npm-package-audit npm-package-pack npm-package-verify docs-reference docs-reference-check docs-freshness docs-qa site-install site-audit site-build site-verify demo-assets tui-media release-snapshot verify build install-local clean-reports

doctor:
	bash scripts/dev-doctor.sh

install-dev-tools:
	$(CARGO) install cargo-audit --version $(CARGO_AUDIT_VERSION) --locked
	$(RUSTUP) toolchain install $(CARGO_DEV_TOOLCHAIN) --profile minimal
	$(CARGO) +$(CARGO_DEV_TOOLCHAIN) install cargo-deny --version $(CARGO_DENY_VERSION) --locked
	$(RUSTUP) component add llvm-tools-preview
	$(CARGO) +$(CARGO_DEV_TOOLCHAIN) install cargo-llvm-cov --version $(CARGO_LLVM_COV_VERSION) --locked

test: cargo-test

test-fast: cargo-test npm-package-test raycast-test

test-security: cargo-audit cargo-deny gitleaks npm-package-audit raycast-audit site-audit

test-ux: raycast-verify site-verify

test-release: ci-scripts-test npm-package-verify raycast-build site-build release-snapshot

test-local: verify

test-prepush: verify

test-release-install:
	@if [ -z "$${VERSION:-}" ]; then echo "VERSION is required, for example: make test-release-install VERSION=0.1.5" >&2; exit 2; fi
	bash scripts/verify-npm-release.sh "$${VERSION}"

fmt:
	$(CARGO) fmt --all --check

clippy:
	$(CARGO) clippy --workspace --all-targets --all-features -- -D warnings

cargo-test:
	$(CARGO) test --workspace

cargo-nextest:
	@if command -v cargo-nextest >/dev/null 2>&1; then $(CARGO) nextest run --workspace; else $(CARGO) test --workspace; fi

cargo-doc:
	$(CARGO) test --doc --workspace

cargo-audit:
	@PATH="$(RUST_PATH)"; if command -v cargo-audit >/dev/null 2>&1; then $(CARGO) audit; else echo "cargo-audit not installed; skipping local audit"; fi

cargo-deny:
	@PATH="$(RUST_PATH)"; if command -v cargo-deny >/dev/null 2>&1; then $(CARGO) deny check; else echo "cargo-deny not installed; skipping local deny check"; fi

cargo-llvm-cov:
	@PATH="$(RUST_PATH)"; if command -v cargo-llvm-cov >/dev/null 2>&1; then mkdir -p $(REPORTS_DIR) && $(CARGO) llvm-cov --workspace --lcov --output-path $(REPORTS_DIR)/coverage.lcov --summary-only | tee $(REPORTS_DIR)/coverage.txt; else $(CARGO) test --workspace; fi

coverage-check: cargo-llvm-cov
	@if [ -f "$(REPORTS_DIR)/coverage.txt" ]; then python3 -c 'import pathlib,re,sys; text=pathlib.Path("$(REPORTS_DIR)/coverage.txt").read_text(); nums=[float(x) for x in re.findall(r"([0-9]+(?:\.[0-9]+)?)%", text)]; pct=nums[-1] if nums else 100.0; threshold=float("$(COVERAGE_THRESHOLD)"); print(f"coverage {pct:.1f}% / threshold {threshold:.1f}%"); sys.exit(0 if pct >= threshold else 1)'; fi

fuzz-smoke:
	@PATH="$(RUST_PATH)"; if command -v cargo-fuzz >/dev/null 2>&1; then cargo fuzz run mcp_config_formats -- -runs=256 && cargo fuzz run redaction_urls_headers -- -runs=256 && cargo fuzz run filesystem_boundaries -- -runs=128; else echo "cargo-fuzz not installed; skipping fuzz smoke"; fi

test-junit: clean-reports cargo-test raycast-install
	mkdir -p $(REPORTS_DIR)/junit
	cd $(RAYCAST_DIR) && npm run test:junit

trunk-check:
	trunk check --show-existing --all

trunk-fix:
	trunk check --show-existing --fix --all

trunk-flaky-validate:
	@if [ -f "$(REPORTS_DIR)/junit/raycast.xml" ]; then trunk flakytests validate --junit-paths $(REPORTS_DIR)/junit/raycast.xml; else echo "Raycast JUnit report not present; skipping flaky validate"; fi

ci-scripts-test:
	bash scripts/test-dco.sh
	bash scripts/test-action-paths.sh
	bash scripts/test-release-scripts.sh

gitleaks:
	@PATH="$(RUST_PATH)"; if command -v gitleaks >/dev/null 2>&1; then gitleaks detect --source . --redact --no-banner; else echo "gitleaks not installed; skipping local secret scan"; fi

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

demo-assets:
	node scripts/generate-demo-assets.mjs

tui-media:
	node scripts/generate-tui-media.mjs

release-snapshot: build
	bash scripts/release-snapshot-rust.sh

verify: doctor fmt clippy cargo-nextest cargo-doc coverage-check test-junit trunk-flaky-validate trunk-check ci-scripts-test raycast-audit raycast-lint raycast-build npm-package-verify site-verify

build:
	$(CARGO) build --release --bins
	mkdir -p bin
	cp target/release/nightward bin/nightward
	cp target/release/nw bin/nw

install-local: build
	mkdir -p $(PREFIX)/bin
	install -m 0755 bin/nightward $(PREFIX)/bin/nightward
	install -m 0755 bin/nw $(PREFIX)/bin/nw

clean-reports:
	rm -rf $(REPORTS_DIR)
