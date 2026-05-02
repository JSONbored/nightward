use serde::Serialize;
use std::path::Path;

#[derive(Debug, Clone, Serialize)]
pub struct SnapshotPlan {
    pub schema_version: u32,
    pub mode: String,
    pub root: String,
    pub destination: String,
    pub writes: bool,
    pub notes: Vec<String>,
}

pub fn plan(root: impl AsRef<Path>, destination: impl AsRef<Path>) -> SnapshotPlan {
    SnapshotPlan {
        schema_version: 1,
        mode: "plan-only".to_string(),
        root: root.as_ref().display().to_string(),
        destination: destination.as_ref().display().to_string(),
        writes: false,
        notes: vec![
            "Snapshot support is plan-only in v1.".to_string(),
            "Use the plan to review what would be copied before any future write-capable workflow."
                .to_string(),
        ],
    }
}
