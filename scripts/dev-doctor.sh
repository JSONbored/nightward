#!/usr/bin/env bash
set -euo pipefail

export PATH="${HOME}/.cargo/bin:/opt/homebrew/bin:${PATH}"

required=(cargo rustc node npm git make)
optional=(trunk gitleaks vhs ffmpeg cargo-audit cargo-deny cargo-llvm-cov)
missing_required=()

echo "Nightward developer toolchain"
echo

for tool in "${required[@]}"; do
  if command -v "$tool" >/dev/null 2>&1; then
    printf "  ok       %-15s %s\n" "$tool" "$(command -v "$tool")"
  else
    printf "  missing  %-15s required\n" "$tool"
    missing_required+=("$tool")
  fi
done

echo
for tool in "${optional[@]}"; do
  if command -v "$tool" >/dev/null 2>&1; then
    printf "  ok       %-15s %s\n" "$tool" "$(command -v "$tool")"
  else
    printf "  optional %-15s not installed\n" "$tool"
  fi
done

echo
echo "Optional Cargo tools can be installed with: make install-dev-tools"

if ((${#missing_required[@]} > 0)); then
  echo
  echo "Missing required tools: ${missing_required[*]}" >&2
  echo "Install Rust with rustup, install Node/npm, then run: . scripts/dev-env.sh" >&2
  exit 1
fi

echo
echo "Required developer tools are available."
