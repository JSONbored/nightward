#![no_main]

use libfuzzer_sys::fuzz_target;
use nightward_core::inventory::{scan_home, scan_workspace};
use std::fs;

fuzz_target!(|data: &[u8]| {
    let Ok(dir) = tempfile::tempdir() else {
        return;
    };
    let (path, body, home_scan) = match data.first().copied().unwrap_or(0) % 3 {
        0 => (dir.path().join(".mcp.json"), data, false),
        1 => (dir.path().join(".codex/config.toml"), data, false),
        _ => (dir.path().join(".config/goose/config.yaml"), data, true),
    };
    if let Some(parent) = path.parent() {
        let _ = fs::create_dir_all(parent);
    }
    let _ = fs::write(&path, body);
    if home_scan {
        let _ = scan_home(dir.path());
    } else {
        let _ = scan_workspace(dir.path());
    }
});
