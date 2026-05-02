fn main() {
    if let Err(error) = nightward_cli::run() {
        eprintln!("nightward: {error}");
        std::process::exit(1);
    }
}
