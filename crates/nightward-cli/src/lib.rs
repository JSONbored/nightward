mod cli;
mod tui;

pub fn run() -> anyhow::Result<()> {
    cli::run()
}
