#!/usr/bin/env bash
set -euo pipefail

name="${1:?path name required}"
value="${2:-}"

if [[ -z "${value}" ]]; then
  exit 0
fi
if [[ "${name}" == "output" && "${value}" == "-" ]]; then
  exit 0
fi

case "${value}" in
  /* | *\\* | *$'\n'* | *$'\r'* | .. | ../* | */.. | */../*)
    echo "unsafe ${name} path: ${value}" >&2
    echo "${name} must be a relative path inside GITHUB_WORKSPACE without parent traversal." >&2
    exit 2
    ;;
esac
