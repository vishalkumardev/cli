#!/bin/sh
# BuildShare CLI Installer for macOS and Linux
# https://buildshare.in
#
# Usage:
#   curl -fsSL https://buildshare.in/install.sh | sh
#
# Environment variables:
#   BUILDSHARE_INSTALL_DIR  Custom directory to install the binary into
#   BUILDSHARE_VERSION      Specific version/tag to install (default: latest)
#   GITHUB_OWNER            GitHub repository owner (default: vishalkumardev)
#   GITHUB_REPO             GitHub repository name (default: cli)

set -eu

# Configuration
GITHUB_OWNER="${GITHUB_OWNER:-vishalkumardev}"
GITHUB_REPO="${GITHUB_REPO:-cli}"
BINARY_NAME="buildshare"

# Color support
if [ -t 1 ]; then
    COLOR_RESET="\033[0m"
    COLOR_BOLD="\033[1m"
    COLOR_GREEN="\033[32m"
    COLOR_YELLOW="\033[33m"
    COLOR_RED="\033[31m"
    COLOR_CYAN="\033[36m"
else
    COLOR_RESET=""
    COLOR_BOLD=""
    COLOR_GREEN=""
    COLOR_YELLOW=""
    COLOR_RED=""
    COLOR_CYAN=""
fi

echo "${COLOR_BOLD}BuildShare CLI Installer${COLOR_RESET}"
echo ""

# 1. Detect operating system
RAW_OS="$(uname -s)"
case "$RAW_OS" in
    Darwin)
        PLATFORM_OS="darwin"
        OS_DISPLAY="macOS"
        ;;
    Linux)
        PLATFORM_OS="linux"
        OS_DISPLAY="Linux"
        ;;
    *)
        echo "${COLOR_RED}Error: Unsupported operating system: ${RAW_OS}${COLOR_RESET}" >&2
        echo "BuildShare CLI currently supports macOS and Linux via install.sh." >&2
        echo "For Windows, install via PowerShell:" >&2
        echo "  irm https://buildshare.in/install.ps1 | iex" >&2
        exit 1
        ;;
esac

# 2. Detect CPU architecture
RAW_ARCH="$(uname -m)"
case "$RAW_ARCH" in
    x86_64|amd64)
        PLATFORM_ARCH="amd64"
        ;;
    arm64|aarch64)
        PLATFORM_ARCH="arm64"
        ;;
    *)
        echo "${COLOR_RED}Error: Unsupported architecture: ${RAW_ARCH}${COLOR_RESET}" >&2
        echo "BuildShare CLI supports x86_64 (amd64) and Apple Silicon / ARM64 (arm64)." >&2
        exit 1
        ;;
esac

echo "Detected platform: ${OS_DISPLAY}"
echo "Detected architecture: ${PLATFORM_ARCH}"
echo ""

# Check download utilities
if command -v curl >/dev/null 2>&1; then
    DOWNLOAD_CMD="curl"
elif command -v wget >/dev/null 2>&1; then
    DOWNLOAD_CMD="wget"
else
    echo "${COLOR_RED}Error: curl or wget is required to download BuildShare CLI.${COLOR_RESET}" >&2
    exit 1
fi

download_file() {
    local url="$1"
    local output="$2"
    if [ "$DOWNLOAD_CMD" = "curl" ]; then
        curl -fsSL --retry 2 "$url" -o "$output" 2>/dev/null
    else
        wget -q -O "$output" "$url" 2>/dev/null
    fi
}

resolve_latest_tag() {
    if [ -n "${BUILDSHARE_VERSION:-}" ]; then
        echo "$BUILDSHARE_VERSION"
        return 0
    fi

    local tag=""
    # Method 1: Check GitHub redirect header
    if [ "$DOWNLOAD_CMD" = "curl" ]; then
        local effective_url
        effective_url=$(curl -sIL -o /dev/null -w '%{url_effective}' "https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest" 2>/dev/null || true)
        tag=$(echo "$effective_url" | sed -E 's#.*/tag/([^/?#]+).*#\1#')
    fi

    if [ -n "$tag" ] && [ "$tag" != "latest" ] && [ "$tag" != "releases" ]; then
        echo "$tag"
        return 0
    fi

    # Method 2: GitHub API
    if [ "$DOWNLOAD_CMD" = "curl" ]; then
        tag=$(curl -fsSL "https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | head -n 1 | cut -d '"' -f 4 || true)
    else
        tag=$(wget -qO- "https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | head -n 1 | cut -d '"' -f 4 || true)
    fi

    if [ -n "$tag" ]; then
        echo "$tag"
        return 0
    fi

    echo ""
}

# 3. Setup temporary workspace
TMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'buildshare-installer')
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

# 4. Check for existing installation
EXISTING_PATH="$(command -v "$BINARY_NAME" 2>/dev/null || true)"
if [ -n "$EXISTING_PATH" ]; then
    echo "Found existing installation at: ${EXISTING_PATH}"
fi

# 5. Resolve version and download asset
TAG="$(resolve_latest_tag)"

echo "Downloading BuildShare CLI..."

# Candidate download URLs to handle release asset naming conventions
ASSET_DOWNLOADED=false
DOWNLOADED_FILE="$TMP_DIR/downloaded_asset"

CANDIDATE_URLS=""
if [ -n "$TAG" ]; then
    CANDIDATE_URLS="${CANDIDATE_URLS} https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest/download/buildshare_${TAG}_${PLATFORM_OS}_${PLATFORM_ARCH}.tar.gz"
    CANDIDATE_URLS="${CANDIDATE_URLS} https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/${TAG}/buildshare_${TAG}_${PLATFORM_OS}_${PLATFORM_ARCH}.tar.gz"
fi
CANDIDATE_URLS="${CANDIDATE_URLS} https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest/download/buildshare_${PLATFORM_OS}_${PLATFORM_ARCH}.tar.gz"
CANDIDATE_URLS="${CANDIDATE_URLS} https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest/download/buildshare-${PLATFORM_OS}-${PLATFORM_ARCH}.tar.gz"
CANDIDATE_URLS="${CANDIDATE_URLS} https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest/download/buildshare-${PLATFORM_OS}-${PLATFORM_ARCH}"

for url in $CANDIDATE_URLS; do
    if download_file "$url" "$DOWNLOADED_FILE"; then
        if [ -s "$DOWNLOADED_FILE" ]; then
            ASSET_DOWNLOADED=true
            break
        fi
    fi
done

if [ "$ASSET_DOWNLOADED" = "false" ]; then
    echo "${COLOR_RED}Error: Failed to download BuildShare CLI binary for ${PLATFORM_OS}-${PLATFORM_ARCH}.${COLOR_RESET}" >&2
    echo "Please check that a release exists at: https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases" >&2
    exit 1
fi

# 6. Extract / Prepare binary
EXTRACT_DIR="$TMP_DIR/extracted"
mkdir -p "$EXTRACT_DIR"

NEW_BINARY=""
if tar -tzf "$DOWNLOADED_FILE" >/dev/null 2>&1; then
    tar -xzf "$DOWNLOADED_FILE" -C "$EXTRACT_DIR"
    if [ -f "$EXTRACT_DIR/$BINARY_NAME" ]; then
        NEW_BINARY="$EXTRACT_DIR/$BINARY_NAME"
    else
        # Find any executable or binary named buildshare inside
        FOUND=$(find "$EXTRACT_DIR" -type f -name "$BINARY_NAME" 2>/dev/null | head -n 1)
        if [ -n "$FOUND" ]; then
            NEW_BINARY="$FOUND"
        fi
    fi
else
    # Direct binary
    NEW_BINARY="$DOWNLOADED_FILE"
fi

if [ -z "$NEW_BINARY" ] || [ ! -f "$NEW_BINARY" ]; then
    echo "${COLOR_RED}Error: Could not locate '${BINARY_NAME}' executable in downloaded asset.${COLOR_RESET}" >&2
    exit 1
fi

chmod +x "$NEW_BINARY"

# 7. Checksum verification structure (verifies sha256 if checksum file is available)
verify_checksum() {
    local binary_file="$1"
    local checksum_url="https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest/download/checksums.txt"
    local checksum_file="$TMP_DIR/checksums.txt"

    if download_file "$checksum_url" "$checksum_file" && [ -s "$checksum_file" ]; then
        if command -v shasum >/dev/null 2>&1; then
            local expected_hash
            expected_hash=$(grep "$(basename "$DOWNLOADED_FILE")" "$checksum_file" 2>/dev/null | awk '{print $1}')
            if [ -n "$expected_hash" ]; then
                local actual_hash
                actual_hash=$(shasum -a 256 "$DOWNLOADED_FILE" | awk '{print $1}')
                if [ "$expected_hash" != "$actual_hash" ]; then
                    echo "${COLOR_RED}Error: Checksum verification failed!${COLOR_RESET}" >&2
                    exit 1
                fi
            fi
        fi
    fi
}
verify_checksum "$NEW_BINARY"

# 8. Determine installation directory
USE_SUDO=false

if [ -n "${BUILDSHARE_INSTALL_DIR:-}" ]; then
    INSTALL_DIR="$BUILDSHARE_INSTALL_DIR"
elif [ -n "$EXISTING_PATH" ] && [ -w "$(dirname "$EXISTING_PATH")" ]; then
    INSTALL_DIR="$(dirname "$EXISTING_PATH")"
elif [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
elif [ -d "/usr/local/bin" ] && command -v sudo >/dev/null 2>&1 && (sudo -n true 2>/dev/null || [ -t 0 ]); then
    INSTALL_DIR="/usr/local/bin"
    USE_SUDO=true
elif [ ! -d "/usr/local/bin" ] && [ -w "/usr/local" ]; then
    mkdir -p "/usr/local/bin"
    INSTALL_DIR="/usr/local/bin"
elif command -v sudo >/dev/null 2>&1 && (sudo -n true 2>/dev/null || [ -t 0 ]); then
    INSTALL_DIR="/usr/local/bin"
    USE_SUDO=true
else
    INSTALL_DIR="$HOME/.local/bin"
fi

echo "Installing BuildShare CLI..."

# Ensure target directory exists
if [ ! -d "$INSTALL_DIR" ]; then
    if [ "$USE_SUDO" = "true" ]; then
        sudo mkdir -p "$INSTALL_DIR"
    else
        mkdir -p "$INSTALL_DIR" 2>/dev/null || {
            if command -v sudo >/dev/null 2>&1 && (sudo -n true 2>/dev/null || [ -t 0 ]); then
                USE_SUDO=true
                sudo mkdir -p "$INSTALL_DIR"
            else
                INSTALL_DIR="$HOME/.local/bin"
                mkdir -p "$INSTALL_DIR"
            fi
        }
    fi
fi

TARGET_FILE="$INSTALL_DIR/$BINARY_NAME"
TARGET_TMP="$INSTALL_DIR/.${BINARY_NAME}.tmp.$$"

# Safe atomic replacement
if [ "$USE_SUDO" = "true" ]; then
    if [ -t 0 ] || sudo -n true 2>/dev/null; then
        sudo cp "$NEW_BINARY" "$TARGET_TMP"
        sudo chmod 755 "$TARGET_TMP"
        sudo mv -f "$TARGET_TMP" "$TARGET_FILE"
    else
        # Cannot elevate sudo non-interactively, fallback to user-local bin
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
        TARGET_FILE="$INSTALL_DIR/$BINARY_NAME"
        TARGET_TMP="$INSTALL_DIR/.${BINARY_NAME}.tmp.$$"
        cp "$NEW_BINARY" "$TARGET_TMP"
        chmod 755 "$TARGET_TMP"
        mv -f "$TARGET_TMP" "$TARGET_FILE"
    fi
else
    if [ -w "$INSTALL_DIR" ]; then
        cp "$NEW_BINARY" "$TARGET_TMP"
        chmod 755 "$TARGET_TMP"
        mv -f "$TARGET_TMP" "$TARGET_FILE"
    else
        # Fallback to user-local directory
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
        TARGET_FILE="$INSTALL_DIR/$BINARY_NAME"
        TARGET_TMP="$INSTALL_DIR/.${BINARY_NAME}.tmp.$$"
        cp "$NEW_BINARY" "$TARGET_TMP"
        chmod 755 "$TARGET_TMP"
        mv -f "$TARGET_TMP" "$TARGET_FILE"
    fi
fi

# 9. Verify installation
if [ ! -x "$TARGET_FILE" ]; then
    echo "${COLOR_RED}Error: Installed binary at ${TARGET_FILE} is not executable.${COLOR_RESET}" >&2
    exit 1
fi

INSTALLED_VERSION=""
if VER_CMD=$("$TARGET_FILE" version 2>/dev/null); then
    PARSED_VER=$(echo "$VER_CMD" | grep -i "BuildShare CLI" | awk '{print $3}' || true)
    if [ -n "$PARSED_VER" ]; then
        INSTALLED_VERSION="$PARSED_VER"
    fi
fi

if [ -z "$INSTALLED_VERSION" ]; then
    if VER_FLAG=$("$TARGET_FILE" --version 2>/dev/null); then
        INSTALLED_VERSION=$(echo "$VER_FLAG" | awk '{print $NF}' || true)
    fi
fi

if [ -z "$INSTALLED_VERSION" ] && [ -n "$TAG" ]; then
    INSTALLED_VERSION="$TAG"
fi

if [ -z "$INSTALLED_VERSION" ]; then
    INSTALLED_VERSION="installed"
fi

echo ""
echo "${COLOR_GREEN}✓ BuildShare CLI installed successfully${COLOR_RESET}"
echo ""
echo "Version: ${INSTALLED_VERSION}"
echo ""
echo "Run:"
echo ""
echo "  buildshare --help"
echo ""

# 10. PATH check and guidance
case ":$PATH:" in
    *":$INSTALL_DIR:"*)
        # Already in PATH
        ;;
    *)
        echo "${COLOR_YELLOW}⚠️  Note: '${INSTALL_DIR}' is not currently in your PATH.${COLOR_RESET}"
        echo "To run '${BINARY_NAME}' from anywhere, add it to your PATH:"
        echo ""
        USER_SHELL="$(basename "${SHELL:-sh}")"
        case "$USER_SHELL" in
            zsh)
                echo "  ${COLOR_CYAN}echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.zshrc${COLOR_RESET}"
                echo "  ${COLOR_CYAN}source ~/.zshrc${COLOR_RESET}"
                ;;
            bash)
                if [ "$PLATFORM_OS" = "darwin" ]; then
                    echo "  ${COLOR_CYAN}echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.bash_profile${COLOR_RESET}"
                    echo "  ${COLOR_CYAN}source ~/.bash_profile${COLOR_RESET}"
                else
                    echo "  ${COLOR_CYAN}echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.bashrc${COLOR_RESET}"
                    echo "  ${COLOR_CYAN}source ~/.bashrc${COLOR_RESET}"
                fi
                ;;
            fish)
                echo "  ${COLOR_CYAN}fish_add_path ${INSTALL_DIR}${COLOR_RESET}"
                ;;
            *)
                echo "  ${COLOR_CYAN}export PATH=\"${INSTALL_DIR}:\$PATH\"${COLOR_RESET}"
                ;;
        esac
        echo ""
        ;;
esac
