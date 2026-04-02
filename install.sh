#!/usr/bin/env bash
# SubSage Install Script
# Usage: curl -fsSL https://raw.githubusercontent.com/wangjc683/subsage/main/install.sh | bash
set -euo pipefail

REPO="wangjc683/subsage"
INSTALL_DIR="${SUBSAGE_DIR:-$HOME/subsage}"
PORT="${SAGE_PORT:-8321}"

# --- Colors ---
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${CYAN}[info]${NC}  $*"; }
ok()    { echo -e "${GREEN}[ok]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[warn]${NC}  $*"; }
fail()  { echo -e "${RED}[error]${NC} $*"; exit 1; }

# --- Detect OS & Arch ---
detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"

  case "$os" in
    linux)  os="linux" ;;
    darwin) os="darwin" ;;
    *)      fail "Unsupported OS: $os. SubSage supports Linux and macOS." ;;
  esac

  case "$arch" in
    x86_64|amd64)   arch="amd64" ;;
    aarch64|arm64)   arch="arm64" ;;
    *)               fail "Unsupported architecture: $arch. SubSage supports amd64 and arm64." ;;
  esac

  echo "${os}-${arch}"
}

# --- Get latest version from GitHub ---
get_latest_version() {
  local version
  version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
  [ -z "$version" ] && fail "Could not determine latest version. Check your network connection."
  echo "$version"
}

# --- Main ---
main() {
  echo ""
  echo -e "${GREEN}🌱 SubSage Installer${NC}"
  echo "────────────────────────────────"
  echo ""

  # Detect platform
  local platform
  platform=$(detect_platform)
  info "Detected platform: ${platform}"

  # Get latest version
  info "Fetching latest version..."
  local version
  version=$(get_latest_version)
  ok "Latest version: ${version}"

  # Download
  local binary="subsage-${platform}"
  local url="https://github.com/${REPO}/releases/download/${version}/${binary}"

  info "Downloading ${binary}..."
  mkdir -p "$INSTALL_DIR"

  if command -v curl &>/dev/null; then
    curl -fSL --progress-bar "$url" -o "${INSTALL_DIR}/subsage" || fail "Download failed. Check if the release has binaries attached."
  elif command -v wget &>/dev/null; then
    wget -q --show-progress "$url" -O "${INSTALL_DIR}/subsage" || fail "Download failed."
  else
    fail "Neither curl nor wget found. Please install one of them."
  fi

  chmod +x "${INSTALL_DIR}/subsage"
  ok "Downloaded to ${INSTALL_DIR}/subsage"

  # Create data directory
  mkdir -p "${INSTALL_DIR}/data"

  # --- Platform-specific setup ---
  local os="${platform%%-*}"

  if [ "$os" = "linux" ] && command -v systemctl &>/dev/null; then
    setup_systemd
  elif [ "$os" = "darwin" ]; then
    setup_launchd
  else
    info "Skipping service registration (no systemd detected)"
  fi

  # --- Done ---
  echo ""
  echo "────────────────────────────────"
  ok "SubSage ${version} installed successfully!"
  echo ""
  echo -e "  ${CYAN}Install directory:${NC}  ${INSTALL_DIR}"
  echo -e "  ${CYAN}Database:${NC}           ${INSTALL_DIR}/data/sage.db"
  echo -e "  ${CYAN}Access URL:${NC}         http://localhost:${PORT}"
  echo ""

  if [ "$os" = "linux" ] && command -v systemctl &>/dev/null; then
    echo -e "  ${CYAN}Start:${NC}   sudo systemctl start subsage"
    echo -e "  ${CYAN}Stop:${NC}    sudo systemctl stop subsage"
    echo -e "  ${CYAN}Logs:${NC}    journalctl -u subsage -f"
  elif [ "$os" = "darwin" ]; then
    echo -e "  ${CYAN}Start:${NC}   launchctl load ~/Library/LaunchAgents/com.subsage.plist"
    echo -e "  ${CYAN}Stop:${NC}    launchctl unload ~/Library/LaunchAgents/com.subsage.plist"
  else
    echo -e "  ${CYAN}Run:${NC}     SAGE_DB_PATH=${INSTALL_DIR}/data/sage.db ${INSTALL_DIR}/subsage"
  fi

  echo ""
  echo -e "  First visit will prompt you to create an admin account."
  echo ""

  # Auto-start on Linux
  if [ "$os" = "linux" ] && command -v systemctl &>/dev/null; then
    info "Starting SubSage..."
    sudo systemctl start subsage 2>/dev/null && ok "SubSage is running!" || warn "Could not auto-start. Run: sudo systemctl start subsage"
  fi
}

# --- systemd service (Linux) ---
setup_systemd() {
  info "Setting up systemd service..."
  sudo tee /etc/systemd/system/subsage.service >/dev/null <<EOF
[Unit]
Description=SubSage Subscription Tracker
After=network.target

[Service]
Type=simple
User=$(whoami)
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/subsage
Environment=SAGE_DB_PATH=${INSTALL_DIR}/data/sage.db
Environment=SAGE_PORT=${PORT}
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
  sudo systemctl daemon-reload
  sudo systemctl enable subsage >/dev/null 2>&1
  ok "systemd service registered (auto-start on boot)"
}

# --- launchd plist (macOS) ---
setup_launchd() {
  info "Setting up launchd service..."
  local plist_dir="$HOME/Library/LaunchAgents"
  mkdir -p "$plist_dir"
  cat > "${plist_dir}/com.subsage.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.subsage</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/subsage</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>SAGE_DB_PATH</key>
        <string>${INSTALL_DIR}/data/sage.db</string>
        <key>SAGE_PORT</key>
        <string>${PORT}</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>${INSTALL_DIR}/subsage.log</string>
    <key>StandardErrorPath</key>
    <string>${INSTALL_DIR}/subsage.log</string>
</dict>
</plist>
EOF
  ok "launchd plist created (auto-start on login)"
}

main "$@"
