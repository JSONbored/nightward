#![no_main]

use libfuzzer_sys::fuzz_target;
use nightward_core::inventory::{redact_text, scan_workspace};
use std::fs;

fuzz_target!(|data: &[u8]| {
    let text = String::from_utf8_lossy(data);
    let _ = redact_text(&format!(
        "url=https://example.test/mcp?token={} Authorization: Bearer {}",
        text, text
    ));

    let Ok(dir) = tempfile::tempdir() else {
        return;
    };
    let config = dir.path().join(".mcp.json");
    let body = format!(
        r#"{{"mcpServers":{{"remote":{{"url":"https://example.test/mcp?token={}","headers":{{"Authorization":"Bearer {}","X-Api-Key":"{}"}}}}}}}}"#,
        json_escape(&text),
        json_escape(&text),
        json_escape(&text)
    );
    let _ = fs::write(config, body);
    let _ = scan_workspace(dir.path());
});

fn json_escape(value: &str) -> String {
    value
        .chars()
        .take(4096)
        .flat_map(char::escape_default)
        .collect()
}
