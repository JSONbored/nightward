use serde::Serialize;
use std::path::Path;

#[derive(Debug, Clone, Serialize)]
pub struct BackupPlan {
    pub schema_version: u32,
    pub mode: String,
    pub root: String,
    pub include: Vec<String>,
    pub exclude: Vec<String>,
    pub notes: Vec<String>,
}

pub fn plan(root: impl AsRef<Path>) -> BackupPlan {
    BackupPlan {
        schema_version: 1,
        mode: "plan-only".to_string(),
        root: root.as_ref().display().to_string(),
        include: vec![
            ".codex/config.toml".to_string(),
            ".cursor/mcp.json".to_string(),
            ".claude.json".to_string(),
        ],
        exclude: vec![
            ".codex/auth.json".to_string(),
            ".ollama/id_ed25519".to_string(),
            "**/cache/**".to_string(),
            "**/logs/**".to_string(),
        ],
        notes: vec![
            "Nightward does not create backups or mutate files.".to_string(),
            "Keep auth files and local model keys out of portable sync by default.".to_string(),
        ],
    }
}
