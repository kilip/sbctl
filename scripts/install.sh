#!/usr/bin/env bash
set -euo pipefail

# sbctl installer
# Usage: curl -fsSL https://raw.githubusercontent.com/kilip/sbctl/main/scripts/install.sh | bash

REPO="kilip/sbctl"
BINARY="sbctl"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RESET='\033[0m'

info()    { echo -e "${CYAN}[sbctl]${RESET} $*"; }
success() { echo -e "${GREEN}[sbctl]${RESET} $*"; }
warn()    { echo -e "${YELLOW}[sbctl]${RESET} $*"; }
error()   { echo -e "${RED}[sbctl]${RESET} $*" >&2; exit 1; }

# Detect OS
detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    *)       error "Unsupported OS: $(uname -s)" ;;
  esac
}

# Detect architecture
detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) error "Unsupported architecture: $(uname -m)" ;;
  esac
}

# Check required tools
check_deps() {
  for cmd in curl tar; do
    if ! command -v "$cmd" &>/dev/null; then
      error "Required tool not found: $cmd"
    fi
  done
}

# Get latest release tag from GitHub
get_latest_version() {
  local version
  version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
  if [[ -z "$version" ]]; then
    error "Could not fetch latest version from GitHub"
  fi
  echo "$version"
}

# Download and install binary
install_binary() {
  local os="$1"
  local arch="$2"
  local version="$3"

  local version_no_v="${version#v}"
  local filename="${BINARY}-${version_no_v}-${os}-${arch}.tar.gz"
  local url="https://github.com/${REPO}/releases/download/${version}/${filename}"
  local tmp_dir
  tmp_dir=$(mktemp -d)

  info "Downloading ${BINARY} ${version} (${os}/${arch})..."
  if ! curl -fsSL "$url" -o "${tmp_dir}/${filename}"; then
    rm -rf "$tmp_dir"
    error "Failed to download: $url"
  fi

  info "Extracting..."
  tar -xzf "${tmp_dir}/${filename}" -C "$tmp_dir"

  mkdir -p "$INSTALL_DIR"
  mv "${tmp_dir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  chmod +x "${INSTALL_DIR}/${BINARY}"

  rm -rf "$tmp_dir"
}

# Check if INSTALL_DIR is in PATH
check_path() {
  if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    warn "${INSTALL_DIR} is not in your PATH."
    warn "Add the following to your shell config (~/.bashrc, ~/.zshrc, etc.):"
    echo ""
    echo -e "  ${YELLOW}export PATH=\"\$HOME/.local/bin:\$PATH\"${RESET}"
    echo ""
  fi
}

main() {
  info "Installing sbctl..."

  check_deps

  local os arch version
  os=$(detect_os)
  arch=$(detect_arch)
  version="${VERSION:-$(get_latest_version)}"

  info "Version : ${version}"
  info "OS      : ${os}"
  info "Arch    : ${arch}"
  info "Target  : ${INSTALL_DIR}/${BINARY}"

  install_binary "$os" "$arch" "$version"

  success "sbctl ${version} installed to ${INSTALL_DIR}/${BINARY}"

  check_path

  info "Running: sbctl setup"
  echo ""
  "${INSTALL_DIR}/${BINARY}" setup
}

main "$@"
