#!/usr/bin/env sh
# Source this file when a non-login shell cannot find the Rust toolchain:
#   . scripts/dev-env.sh

export PATH="${HOME}/.cargo/bin:/opt/homebrew/bin:${PATH}"

if command -v cargo >/dev/null 2>&1; then
  echo "Nightward dev environment ready: $(cargo --version)"
else
  echo "cargo is still unavailable after adding ${HOME}/.cargo/bin to PATH" >&2
  echo "Install Rust with rustup, then reopen the shell or source this file again." >&2
  return 1 2>/dev/null || exit 1
fi
