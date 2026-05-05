mod cli;

#[cfg(unix)]
mod tui;

#[cfg(not(unix))]
mod tui {
    use anyhow::{bail, Result};
    use nightward_core::reportdiff::DiffReport;
    use nightward_core::Report;

    pub fn run(_: &Report) -> Result<()> {
        bail!("Nightward TUI is not supported on this platform; use `nightward scan --json` or `nightward report html`.")
    }

    pub fn run_compare(_: &DiffReport) -> Result<()> {
        bail!("Nightward TUI compare is not supported on this platform; use `nightward report html --from <old.json> --to <new.json>`.")
    }
}

pub fn run() -> anyhow::Result<()> {
    cli::run()
}
