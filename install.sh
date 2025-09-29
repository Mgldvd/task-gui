#!/usr/bin/env bash
set -euo pipefail

# install.sh - simple installer for taskg
# Usage:
#   curl -sL https://example.com/install.sh | bash
#   OR
#   ./install.sh [repo-url] [branch]

REPO_URL=${1:-https://github.com/Mgldvd/task-gui.git}
BRANCH=${2:-master}

DEFAULT_REPO_URL="https://github.com/Mgldvd/task-gui.git"

command -v git >/dev/null 2>&1 || { echo "git is required. Aborting." >&2; exit 1; }
command -v go >/dev/null 2>&1 || { echo "Go is required. Aborting." >&2; exit 1; }

# If running inside an existing checkout that looks like the project, prefer local build.
USE_LOCAL=0
if [ "$REPO_URL" = "$DEFAULT_REPO_URL" ]; then
  if [ -f "go.mod" ] && [ -d "cmd/taskg" ]; then
    USE_LOCAL=1
  fi
fi

if [ "$USE_LOCAL" -eq 1 ]; then
  SRC_DIR="$(pwd)"
  echo "Using local source at $SRC_DIR"
  CLONED=0
else
  TMPDIR=$(mktemp -d)
  echo "Using temporary dir: $TMPDIR"
  cleanup() {
    rm -rf "$TMPDIR"
  }
  trap cleanup EXIT

  echo "Cloning $REPO_URL (branch: $BRANCH)"
  git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$TMPDIR/repo"
  SRC_DIR="$TMPDIR/repo"
  CLONED=1
fi

cd "$SRC_DIR"

echo "Building..."
# Prefer make build if available, but fall back to detecting main packages
if command -v make >/dev/null 2>&1 && grep -q "build" Makefile 2>/dev/null; then
  if make build; then
    echo "make build succeeded"
  else
    echo "make build failed, falling back to detecting main packages" >&2
    BUILD_FALLBACK=1
  fi
else
  BUILD_FALLBACK=1
fi

if [ "${BUILD_FALLBACK:-0}" -eq 1 ]; then
  echo "Searching for Go main packages to build..."
  # Prefer using 'go list' to find main packages (more reliable than grep)
  if go list -f '{{if eq .Name "main"}}{{.Dir}}{{end}}' ./... 1>/dev/null 2>&1; then
    mapfile -t uniq_dirs < <(go list -f '{{if eq .Name "main"}}{{.Dir}}{{end}}' ./... | grep -v '^$' || true)
  else
    uniq_dirs=()
  fi

  # Fallback to grep-based search if go list didn't find anything
  if [ ${#uniq_dirs[@]} -eq 0 ]; then
    mapfile -t main_dirs < <(grep -R --line-number -I --exclude-dir=.git --exclude-dir=vendor -e 'package main' . 2>/dev/null || true)
    if [ ${#main_dirs[@]} -gt 0 ]; then
      dirs=()
      for f in "${main_dirs[@]}"; do
        filepath="${f%%:*}"
        dirpath=$(dirname "$filepath")
        dirs+=("$dirpath")
      done
      # Deduplicate while preserving order
      declare -A seen
      for d in "${dirs[@]}"; do
        if [ -z "${seen[$d]:-}" ]; then
          uniq_dirs+=("$d")
          seen[$d]=1
        fi
      done
    fi
  fi

  if [ ${#uniq_dirs[@]} -eq 0 ]; then
    echo "No 'package main' found; attempting 'go build ./...' as a last resort" >&2
    if go build -o taskg ./...; then
      echo "go build ./... succeeded"
      exit 0
    else
      echo "No main package found and 'go build ./...' failed" >&2
      exit 1
    fi
  fi

  # Prefer cmd/taskg if present
  build_target=""
  for d in "${uniq_dirs[@]}"; do
    if [ "$d" = "./cmd/taskg" ] || [ "$d" = "cmd/taskg" ] || [ "$(basename "$d")" = "taskg" ]; then
      build_target="$d"
      break
    fi
  done
  if [ -z "$build_target" ]; then
    build_target="${uniq_dirs[0]}"
  fi

  echo "Building main package at: $build_target"
  if go build -o taskg "$build_target"; then
    echo "go build succeeded"
  else
    echo "go build failed for $build_target" >&2
    echo "Attempting 'go build ./...' as a last resort" >&2
    if go build -o taskg ./...; then
      echo "go build ./... succeeded"
    else
      echo "All build attempts failed" >&2
      exit 1
    fi
  fi
fi

BIN_PATH="$(pwd)/taskg"
if [ ! -f "$BIN_PATH" ]; then
  # try default location produced by make
  if [ -f "./cmd/taskg/taskg" ]; then
    BIN_PATH="./cmd/taskg/taskg"
  fi
fi

if [ ! -f "$BIN_PATH" ]; then
  echo "Build failed: binary not found" >&2
  exit 1
fi

DEST="/usr/local/bin/taskg"
echo "Installing to $DEST"
if [ -w "$(dirname "$DEST")" ]; then
  cp "$BIN_PATH" "$DEST"
else
  echo "Need sudo to copy to $DEST"
  sudo cp "$BIN_PATH" "$DEST"
fi

echo "Installed taskg to $DEST"
echo "Run: taskg --help"
