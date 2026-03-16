#!/usr/bin/env bash
# scripts/install-prereqs.sh
# Installs all required development prerequisites for NiteOS on Windows (Git Bash / WSL),
# macOS, or Linux. Run this once on a fresh machine.
#
# Required tools:
#   - Go 1.22+
#   - Docker Desktop (Docker + Docker Compose)
#   - Make
#   - golang-migrate CLI
#   - openssl (for key generation)
#
# Usage: bash scripts/install-prereqs.sh

set -euo pipefail

OS="$(uname -s)"
ARCH="$(uname -m)"
GO_VERSION="1.22.5"
MIGRATE_VERSION="4.17.0"

log()  { echo "[prereqs] $*"; }
ok()   { echo "[prereqs] OK: $*"; }
fail() { echo "[prereqs] FAIL: $*" >&2; exit 1; }

# ── Go ─────────────────────────────────────────────────────────────────────────
install_go_windows() {
  log "Installing Go $GO_VERSION via winget..."
  winget install GoLang.Go --silent --accept-source-agreements --accept-package-agreements || true
  log "If winget fails, download from: https://go.dev/dl/go${GO_VERSION}.windows-amd64.msi"
}

install_go_macos() {
  if command -v brew &>/dev/null; then
    log "Installing Go via Homebrew..."
    brew install go
  else
    log "Downloading Go $GO_VERSION for macOS..."
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.darwin-arm64.tar.gz" | sudo tar -C /usr/local -xz
    export PATH="/usr/local/go/bin:$PATH"
    echo 'export PATH="/usr/local/go/bin:$PATH"' >> ~/.profile
  fi
}

install_go_linux() {
  log "Downloading Go $GO_VERSION for Linux..."
  GOARCH="amd64"
  [[ "$ARCH" == "aarch64" ]] && GOARCH="arm64"
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" | sudo tar -C /usr/local -xz
  export PATH="/usr/local/go/bin:$PATH"
  echo 'export PATH="/usr/local/go/bin:$PATH"' | sudo tee /etc/profile.d/go.sh
}

# ── Docker ─────────────────────────────────────────────────────────────────────
install_docker_windows() {
  log "Docker Desktop for Windows must be installed manually."
  log "Download from: https://www.docker.com/products/docker-desktop/"
  log "After installing, start Docker Desktop and wait for the engine to be ready."
}

install_docker_macos() {
  if command -v brew &>/dev/null; then
    log "Installing Docker Desktop via Homebrew..."
    brew install --cask docker
  else
    log "Download Docker Desktop from: https://www.docker.com/products/docker-desktop/"
  fi
}

install_docker_linux() {
  log "Installing Docker Engine on Linux..."
  sudo apt-get update -q
  sudo apt-get install -qy ca-certificates curl gnupg lsb-release
  sudo mkdir -p /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
    | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
  sudo apt-get update -q
  sudo apt-get install -qy docker-ce docker-ce-cli containerd.io docker-compose-plugin
  sudo usermod -aG docker "$USER"
  log "Docker installed. You may need to log out and back in for group membership to take effect."
}

# ── Make ───────────────────────────────────────────────────────────────────────
install_make_windows() {
  log "Installing Make via winget..."
  winget install GnuWin32.Make --silent --accept-source-agreements --accept-package-agreements || true
  log "Alternatively, install via Chocolatey: choco install make"
  log "Or use Git for Windows which includes make in /usr/bin/make"
}

install_make_macos() {
  xcode-select --install 2>/dev/null || true
}

install_make_linux() {
  sudo apt-get install -qy build-essential
}

# ── golang-migrate ─────────────────────────────────────────────────────────────
install_migrate() {
  log "Installing golang-migrate v${MIGRATE_VERSION}..."
  if [[ "$OS" == "Darwin" ]] && command -v brew &>/dev/null; then
    brew install golang-migrate
    return
  fi

  local platform=""
  case "$OS" in
    Linux*)  platform="linux-amd64"  ;;
    Darwin*) platform="darwin-arm64" ;;
    MINGW*|MSYS*|CYGWIN*)
      log "On Windows, download migrate from:"
      log "https://github.com/golang-migrate/migrate/releases/download/v${MIGRATE_VERSION}/migrate.windows-amd64.zip"
      log "Unzip and put migrate.exe in a directory on your PATH (e.g. C:/tools)"
      return
      ;;
  esac

  TMP=$(mktemp -d)
  curl -fsSL "https://github.com/golang-migrate/migrate/releases/download/v${MIGRATE_VERSION}/migrate.${platform}.tar.gz" \
    | tar -xz -C "$TMP"
  sudo mv "$TMP/migrate" /usr/local/bin/migrate
  rm -rf "$TMP"
  ok "golang-migrate installed: $(migrate -version)"
}

# ── Main ───────────────────────────────────────────────────────────────────────
main() {
  log "Detecting OS: $OS"

  # Go
  if ! command -v go &>/dev/null; then
    case "$OS" in
      Darwin*) install_go_macos ;;
      Linux*)  install_go_linux ;;
      MINGW*|MSYS*|CYGWIN*) install_go_windows ;;
      *) fail "Unknown OS: $OS — install Go manually from https://go.dev/dl/" ;;
    esac
  else
    ok "Go already installed: $(go version)"
  fi

  # Docker
  if ! command -v docker &>/dev/null; then
    case "$OS" in
      Darwin*) install_docker_macos ;;
      Linux*)  install_docker_linux ;;
      MINGW*|MSYS*|CYGWIN*) install_docker_windows ;;
      *) fail "Unknown OS — install Docker manually" ;;
    esac
  else
    ok "Docker already installed: $(docker --version)"
  fi

  # Make
  if ! command -v make &>/dev/null; then
    case "$OS" in
      Darwin*) install_make_macos ;;
      Linux*)  install_make_linux ;;
      MINGW*|MSYS*|CYGWIN*) install_make_windows ;;
    esac
  else
    ok "Make already installed: $(make --version | head -1)"
  fi

  # golang-migrate
  if ! command -v migrate &>/dev/null; then
    install_migrate
  else
    ok "golang-migrate already installed: $(migrate -version 2>/dev/null || echo 'installed')"
  fi

  log ""
  log "Installation complete. Run scripts/check-prereqs.sh to verify."
}

main "$@"
