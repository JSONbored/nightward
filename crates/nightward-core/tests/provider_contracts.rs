use nightward_core::analysis::SignalCategory;
use nightward_core::providers::{parse_provider_output, run_provider, statuses};
use std::path::{Path, PathBuf};

fn fixture(name: &str) -> String {
    let path = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("../../testdata/providers")
        .join(name);
    std::fs::read_to_string(path).expect("provider fixture")
}

const FIXTURE_SECRETS: [&str; 3] = [
    "example-gitleaks-redacted-value",
    "example-trivy-redacted-value",
    "example-trufflehog-redacted-value",
];

#[test]
fn provider_fixtures_normalize_supported_outputs() {
    let root = Path::new("/tmp/nightward-provider-fixture");
    let cases = [
        (
            "gitleaks",
            "gitleaks.json",
            1,
            SignalCategory::SecretsExposure,
        ),
        (
            "trufflehog",
            "trufflehog.jsonl",
            1,
            SignalCategory::SecretsExposure,
        ),
        ("semgrep", "semgrep.json", 1, SignalCategory::ExecutionRisk),
        ("trivy", "trivy.json", 3, SignalCategory::SupplyChain),
        (
            "osv-scanner",
            "osv-scanner.json",
            1,
            SignalCategory::SupplyChain,
        ),
        ("socket", "socket.json", 1, SignalCategory::SupplyChain),
    ];

    for (provider, file, expected_count, expected_category) in cases {
        let findings =
            parse_provider_output(provider, root, &fixture(file)).expect("parse provider output");
        assert_eq!(findings.len(), expected_count, "{provider}");
        assert!(
            findings
                .iter()
                .any(|finding| finding.category == expected_category),
            "{provider} should expose {expected_category:?}"
        );
        assert!(
            findings
                .iter()
                .all(|finding| FIXTURE_SECRETS.iter().all(|secret| {
                    !finding.evidence.contains(secret) && !finding.message.contains(secret)
                })),
            "{provider} output should be redacted"
        );
    }
}

#[test]
fn provider_statuses_distinguish_skipped_blocked_and_ready() {
    let skipped = statuses(&[], false);
    let gitleaks = skipped
        .iter()
        .find(|status| status.provider.name == "gitleaks")
        .expect("gitleaks status");
    assert_eq!(gitleaks.status, "skipped");
    assert!(!gitleaks.enabled);

    let blocked = statuses(&["socket".to_string()], false);
    let socket = blocked
        .iter()
        .find(|status| status.provider.name == "socket")
        .expect("socket status");
    assert_eq!(socket.status, "blocked");
    assert!(socket.enabled);

    let built_in = skipped
        .iter()
        .find(|status| status.provider.name == "nightward")
        .expect("nightward status");
    assert_eq!(built_in.status, "ready");
    assert!(built_in.enabled);
}

#[cfg(unix)]
#[test]
fn provider_timeout_returns_stable_warning_error() {
    let _guard = EnvRestore::set(&[
        ("PATH", None),
        ("NIGHTWARD_PROVIDER_TIMEOUT_MS", Some("25")),
        ("NIGHTWARD_PROVIDER_STDOUT_CAP", None),
    ]);
    let dir = tempfile::tempdir().expect("temp dir");
    write_executable(dir.path().join("gitleaks"), "#!/bin/sh\n/bin/sleep 1\n");
    std::env::set_var("PATH", dir.path());

    let error = run_provider("gitleaks", dir.path()).expect_err("timeout");

    assert!(error.to_string().contains("provider timed out after"));
}

#[cfg(unix)]
#[test]
fn provider_stdout_cap_fails_closed_before_parsing() {
    let _guard = EnvRestore::set(&[
        ("PATH", None),
        ("NIGHTWARD_PROVIDER_TIMEOUT_MS", Some("1000")),
        ("NIGHTWARD_PROVIDER_STDOUT_CAP", Some("16")),
    ]);
    let dir = tempfile::tempdir().expect("temp dir");
    write_executable(
        dir.path().join("gitleaks"),
        "#!/bin/sh\nprintf '%s' '[aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa]'\n",
    );
    std::env::set_var("PATH", dir.path());

    let error = run_provider("gitleaks", dir.path()).expect_err("output cap");

    assert_eq!(error.to_string(), "provider stdout exceeded 16 byte cap");
}

#[cfg(unix)]
fn write_executable(path: impl AsRef<Path>, body: &str) {
    use std::os::unix::fs::PermissionsExt;

    std::fs::write(path.as_ref(), body).expect("write provider stub");
    let mut permissions = std::fs::metadata(path.as_ref())
        .expect("provider metadata")
        .permissions();
    permissions.set_mode(0o755);
    std::fs::set_permissions(path.as_ref(), permissions).expect("provider executable");
}

#[cfg(unix)]
struct EnvRestore {
    values: Vec<(&'static str, Option<std::ffi::OsString>)>,
    _guard: std::sync::MutexGuard<'static, ()>,
}

#[cfg(unix)]
impl EnvRestore {
    fn set(vars: &[(&'static str, Option<&str>)]) -> Self {
        static LOCK: std::sync::OnceLock<std::sync::Mutex<()>> = std::sync::OnceLock::new();
        let guard = LOCK
            .get_or_init(|| std::sync::Mutex::new(()))
            .lock()
            .unwrap();
        let values = vars
            .iter()
            .map(|(key, _)| (*key, std::env::var_os(key)))
            .collect::<Vec<_>>();
        for (key, value) in vars {
            if let Some(value) = value {
                std::env::set_var(key, value);
            } else {
                std::env::remove_var(key);
            }
        }
        Self {
            values,
            _guard: guard,
        }
    }
}

#[cfg(unix)]
impl Drop for EnvRestore {
    fn drop(&mut self) {
        for (key, value) in &self.values {
            if let Some(value) = value {
                std::env::set_var(key, value);
            } else {
                std::env::remove_var(key);
            }
        }
    }
}
