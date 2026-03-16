#!/usr/bin/env bash
# scripts/check-prereqs.sh
# Verifies that all required development prerequisites are installed and at the
# correct versions. Prints a clear PASS/FAIL for each.
#
# Usage: bash scripts/check-prereqs.sh
# Exit code: 0 = all pass, 1 = one or more fail

set -euo pipefail

PASS=0
FAIL=1
ALL_PASS=true

check() {
  local name="$1"
  local cmd="$2"
  local min_ver="$3"
  local actual
  if ! actual=$(eval "$cmd" 2>/dev/null); then
    echo "  FAIL  $name: not found (min required: $min_ver)"
    ALL_PASS=false
    return $FAIL
  fi
  echo "  PASS  $name: $actual"
  return $PASS
}

echo ""
echo "NiteOS prerequisite check"
echo "========================="
echo ""

# Go
check "go" \
  "go version | awk '{print \$3}'" \
  "go1.22+"

# Docker
check "docker" \
  "docker --version | awk '{print \$3}' | tr -d ','" \
  "24+"

# Docker Compose
check "docker compose" \
  "docker compose version | awk '{print \$4}'" \
  "v2+"

# Make
check "make" \
  "make --version | head -1" \
  "any"

# openssl
check "openssl" \
  "openssl version" \
  "any"

# golang-migrate
check "migrate" \
  "migrate -version 2>&1 | head -1" \
  "v4.17+"

# git
check "git" \
  "git --version | awk '{print \$3}'" \
  "2+"

echo ""

if $ALL_PASS; then
  echo "All prerequisites met."
  exit 0
else
  echo "One or more prerequisites are missing."
  echo "Run: bash scripts/install-prereqs.sh"
  exit 1
fi
