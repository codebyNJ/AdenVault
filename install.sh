#!/usr/bin/env bash
# adenVault macOS / Linux installer
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/codebyNJ/AdenVault/main/install.sh | bash
#
# Options (set as env vars before piping):
#   ADENV_INSTALL_DIR   default: /usr/local/bin  (falls back to ~/.local/bin if not writable)
#   ADENV_VERSION       default: latest release tag

set -euo pipefail

REPO="codebyNJ/AdenVault"
BINARY="adenV"
INSTALL_DIR="${ADENV_INSTALL_DIR:-}"
VERSION="${ADENV_VERSION:-}"

# ── colours ──────────────────────────────────────────────────────────────────
reset="\033[0m"
bold="\033[1m"
pink="\033[38;5;213m"
cyan="\033[38;5;87m"
green="\033[38;5;82m"
red="\033[38;5;196m"
muted="\033[38;5;240m"

info()    { printf "  ${muted}%s${reset}\n" "$*"; }
success() { printf "  ${green}✓${reset}  %s\n" "$*"; }
error()   { printf "  ${red}✗${reset}  %s\n" "$*" >&2; exit 1; }
header()  { printf "${pink}${bold}%s${reset}\n" "$*"; }

# ── banner ───────────────────────────────────────────────────────────────────
printf "\n"
header " █████╗ ██████╗ ███████╗███╗   ██╗██╗   ██╗"
header "██╔══██╗██╔══██╗██╔════╝████╗  ██║██║   ██║"
header "███████║██║  ██║█████╗  ██╔██╗ ██║██║   ██║"
header "██╔══██║██║  ██║██╔══╝  ██║╚██╗██║╚██╗ ██╔╝"
header "██║  ██║██████╔╝███████╗██║ ╚████║ ╚████╔╝ "
header "╚═╝  ╚═╝╚═════╝ ╚══════╝╚═╝  ╚═══╝  ╚═══╝  "
printf "\n"
printf "  ${pink}${bold}adenVault${reset}  ${muted}· a vault that lives in your home dir, not the cloud${reset}\n"
printf "\n"

# ── prerequisites ─────────────────────────────────────────────────────────────
for cmd in curl tar uname; do
  command -v "$cmd" &>/dev/null || error "required command not found: $cmd"
done

# ── detect OS + arch ─────────────────────────────────────────────────────────
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  darwin) OS="darwin" ;;
  linux)  OS="linux"  ;;
  *)      error "unsupported OS: $OS" ;;
esac

case "$ARCH" in
  x86_64|amd64)         ARCH="amd64" ;;
  arm64|aarch64)        ARCH="arm64" ;;
  *)                    error "unsupported architecture: $ARCH" ;;
esac

info "platform: $OS/$ARCH"

# ── resolve version ───────────────────────────────────────────────────────────
if [[ -z "$VERSION" ]]; then
  info "fetching latest release..."
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
  [[ -n "$VERSION" ]] || error "could not determine latest release — set ADENV_VERSION manually"
fi

info "version: $VERSION"

# ── resolve download URL ──────────────────────────────────────────────────────
# Expected asset name: adenV-<os>-<arch>  (no extension on unix)
ASSET_NAME="${BINARY}-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET_NAME}"

info "downloading ${ASSET_NAME}..."
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL --progress-bar "$DOWNLOAD_URL" -o "${TMP_DIR}/${BINARY}" \
  || error "download failed — check https://github.com/${REPO}/releases for asset names"

chmod +x "${TMP_DIR}/${BINARY}"

# ── choose install directory ──────────────────────────────────────────────────
if [[ -z "$INSTALL_DIR" ]]; then
  if [[ -w /usr/local/bin ]]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
  fi
fi

# ── install ───────────────────────────────────────────────────────────────────
info "installing to ${INSTALL_DIR}/${BINARY}..."

if [[ -w "$INSTALL_DIR" ]]; then
  mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  sudo mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

# ── PATH reminder if using ~/.local/bin ───────────────────────────────────────
if [[ "$INSTALL_DIR" == "$HOME/.local/bin" ]]; then
  if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    printf "\n"
    printf "  ${muted}~/.local/bin is not on your PATH. Add this to your shell profile:${reset}\n"
    printf "  ${cyan}export PATH=\"\$HOME/.local/bin:\$PATH\"${reset}\n"
  fi
fi

# ── verify ────────────────────────────────────────────────────────────────────
printf "\n"
"${INSTALL_DIR}/${BINARY}" --version

printf "\n"
success "adenV is ready."
info    "run: adenV init"
printf "\n"
