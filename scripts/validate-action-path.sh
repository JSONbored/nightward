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

if [[ -n "${GITHUB_WORKSPACE:-}" ]]; then
  workspace_real="$(cd "${GITHUB_WORKSPACE}" && pwd -P)"
  candidate="${GITHUB_WORKSPACE}/${value}"
  current="${GITHUB_WORKSPACE}"
  IFS='/' read -r -a parts <<<"${value}"
  for part in "${parts[@]}"; do
    [[ -z "${part}" || "${part}" == "." ]] && continue
    current="${current}/${part}"
    if [[ -L "${current}" ]]; then
      echo "unsafe ${name} path: ${value}" >&2
      echo "${name} must not contain symlinked path components." >&2
      exit 2
    fi
  done

  parent="$(dirname "${candidate}")"
  nearest="${parent}"
  while [[ ! -e "${nearest}" ]]; do
    next="$(dirname "${nearest}")"
    if [[ "${next}" == "${nearest}" ]]; then
      break
    fi
    nearest="${next}"
  done
  if [[ ! -d "${nearest}" ]]; then
    echo "unsafe ${name} path: ${value}" >&2
    echo "${name} parent must resolve inside GITHUB_WORKSPACE." >&2
    exit 2
  fi
  nearest_real="$(cd "${nearest}" && pwd -P)"
  case "${nearest_real}/" in
    "${workspace_real}/"*) ;;
    *)
      echo "unsafe ${name} path: ${value}" >&2
      echo "${name} parent must resolve inside GITHUB_WORKSPACE." >&2
      exit 2
      ;;
  esac
fi
