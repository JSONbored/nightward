#![no_main]

use libfuzzer_sys::fuzz_target;
use nightward_core::inventory::scan_workspace;
use std::fs::{self, File};

const HUGE_CONFIG_BYTES: u64 = 2 * 1024 * 1024 + 1;

fuzz_target!(|data: &[u8]| {
    let Ok(dir) = tempfile::tempdir() else {
        return;
    };
    match data.first().copied().unwrap_or(0) % 4 {
        0 => {
            let target = dir.path().join("target.json");
            let _ = fs::write(
                &target,
                br#"{"mcpServers":{"demo":{"command":"npx","args":["pkg"]}}}"#,
            );
            #[cfg(unix)]
            {
                let _ = std::os::unix::fs::symlink(&target, dir.path().join(".mcp.json"));
            }
            #[cfg(not(unix))]
            {
                let _ = fs::write(dir.path().join(".mcp.json"), &target.display().to_string());
            }
        }
        1 => {
            if let Ok(file) = File::create(dir.path().join(".mcp.json")) {
                let _ = file.set_len(HUGE_CONFIG_BYTES);
            }
        }
        2 => {
            let _ = fs::write(dir.path().join(".mcp.json"), data);
        }
        _ => {
            let _ = fs::create_dir_all(dir.path().join(".codex"));
            let _ = fs::write(dir.path().join(".codex/config.toml"), data);
        }
    }
    let _ = scan_workspace(dir.path());
});
