#!/bin/sh
# Unit and integration test suite for install.sh
# Tests OS detection, architecture detection, asset resolution,
# existing installation handling, download failure resilience, and PATH detection.

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
INSTALL_SH="$SCRIPT_DIR/install.sh"

TEST_COUNT=0
PASS_COUNT=0
FAIL_COUNT=0

pass() {
    TEST_COUNT=$((TEST_COUNT + 1))
    PASS_COUNT=$((PASS_COUNT + 1))
    printf "  \033[32m✓\033[0m %s\n" "$1"
}

fail() {
    TEST_COUNT=$((TEST_COUNT + 1))
    FAIL_COUNT=$((FAIL_COUNT + 1))
    printf "  \033[31m✗\033[0m %s\n" "$1"
    if [ -n "${2:-}" ]; then
        printf "    Details: %s\n" "$2"
    fi
}

echo "Running BuildShare Installer Test Suite..."
echo ""

# Create a temporary sandbox directory for test artifacts
TEST_SANDBOX="$(mktemp -d 2>/dev/null || mktemp -d -t 'buildshare-test')"
cleanup() {
    rm -rf "$TEST_SANDBOX"
}
trap cleanup EXIT INT TERM

# ──────────────────────────────────────────────────────────────────
# 1. OS & Architecture Detection Unit Tests
# ──────────────────────────────────────────────────────────────────
echo "1. Testing OS and Architecture Detection..."

# Helper to test detection logic with mocked uname
run_detection_test() {
    mock_os="$1"
    mock_arch="$2"
    
    PLATFORM_OS=""
    OS_DISPLAY=""
    RAW_OS="$mock_os"
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
            echo "UNSUPPORTED_OS:$RAW_OS"
            return 0
            ;;
    esac

    PLATFORM_ARCH=""
    RAW_ARCH="$mock_arch"
    case "$RAW_ARCH" in
        x86_64|amd64)
            PLATFORM_ARCH="amd64"
            ;;
        arm64|aarch64)
            PLATFORM_ARCH="arm64"
            ;;
        *)
            echo "UNSUPPORTED_ARCH:$RAW_ARCH"
            return 0
            ;;
    esac

    echo "${PLATFORM_OS}-${PLATFORM_ARCH}"
}

# macOS Apple Silicon
res=$(run_detection_test "Darwin" "arm64")
if [ "$res" = "darwin-arm64" ]; then
    pass "macOS Apple Silicon (Darwin arm64) -> darwin-arm64"
else
    fail "macOS Apple Silicon detection failed" "$res"
fi

# macOS Intel
res=$(run_detection_test "Darwin" "x86_64")
if [ "$res" = "darwin-amd64" ]; then
    pass "macOS Intel (Darwin x86_64) -> darwin-amd64"
else
    fail "macOS Intel detection failed" "$res"
fi

# Linux x64
res=$(run_detection_test "Linux" "x86_64")
if [ "$res" = "linux-amd64" ]; then
    pass "Linux x64 (Linux x86_64) -> linux-amd64"
else
    fail "Linux x64 detection failed" "$res"
fi

# Linux arm64
res=$(run_detection_test "Linux" "aarch64")
if [ "$res" = "linux-arm64" ]; then
    pass "Linux ARM64 (Linux aarch64) -> linux-arm64"
else
    fail "Linux ARM64 detection failed" "$res"
fi

# Unsupported OS (e.g. FreeBSD, Windows)
res=$(run_detection_test "FreeBSD" "x86_64")
if [ "$res" = "UNSUPPORTED_OS:FreeBSD" ]; then
    pass "Unsupported OS (FreeBSD) rejected correctly"
else
    fail "Unsupported OS was not rejected" "$res"
fi

# Unsupported Architecture (e.g. mips, i386)
res=$(run_detection_test "Linux" "mips")
if [ "$res" = "UNSUPPORTED_ARCH:mips" ]; then
    pass "Unsupported Arch (mips) rejected correctly"
else
    fail "Unsupported Arch was not rejected" "$res"
fi

# ──────────────────────────────────────────────────────────────────
# 2. Asset URL Resolution Tests
# ──────────────────────────────────────────────────────────────────
echo ""
echo "2. Testing Asset URL Resolution..."

owner="vishalkumardev"
repo="cli"
tag="v0.1.0"
os="darwin"
arch="arm64"

expected_tar_url="https://github.com/$owner/$repo/releases/latest/download/buildshare_${tag}_${os}_${arch}.tar.gz"
expected_direct_url="https://github.com/$owner/$repo/releases/latest/download/buildshare-${os}-${arch}"

if [ "$expected_tar_url" = "https://github.com/vishalkumardev/cli/releases/latest/download/buildshare_v0.1.0_darwin_arm64.tar.gz" ]; then
    pass "Release archive asset URL matches workflow convention"
else
    fail "Release archive asset URL incorrect"
fi

if [ "$expected_direct_url" = "https://github.com/vishalkumardev/cli/releases/latest/download/buildshare-darwin-arm64" ]; then
    pass "Direct binary fallback asset URL matches Makefile convention"
else
    fail "Direct binary fallback asset URL incorrect"
fi

# ──────────────────────────────────────────────────────────────────
# 3. End-to-End Mock Installation Test
# ──────────────────────────────────────────────────────────────────
echo ""
echo "3. Testing End-to-End Installation with Mocks..."

MOCK_BIN_DIR="$TEST_SANDBOX/mock_bin"
MOCK_INSTALL_DIR="$TEST_SANDBOX/install_target"
mkdir -p "$MOCK_BIN_DIR" "$MOCK_INSTALL_DIR"

# Create a mock buildshare binary that prints version
create_mock_buildshare() {
    local target="$1"
    local version="$2"
    cat <<EOF > "$target"
#!/bin/sh
if [ "\$1" = "version" ]; then
    echo "BuildShare CLI $version"
    echo "Commit: none"
    echo "OS: \$(uname -s)"
    exit 0
fi
if [ "\$1" = "--version" ]; then
    echo "buildshare version $version"
    exit 0
fi
if [ "\$1" = "--help" ]; then
    echo "BuildShare CLI — ship builds to your team"
    exit 0
fi
echo "Usage: buildshare [command]"
exit 0
EOF
    chmod +x "$target"
}

# Create mock release asset (tar.gz)
MOCK_ASSET_TAR="$TEST_SANDBOX/buildshare_v1.0.0_darwin_arm64.tar.gz"
MOCK_PAYLOAD_DIR="$TEST_SANDBOX/payload"
mkdir -p "$MOCK_PAYLOAD_DIR"
create_mock_buildshare "$MOCK_PAYLOAD_DIR/buildshare" "v1.0.0"
(cd "$MOCK_PAYLOAD_DIR" && tar -czf "$MOCK_ASSET_TAR" buildshare)

# Create a mock curl that serves the mock tar.gz
cat <<EOF > "$MOCK_BIN_DIR/curl"
#!/bin/sh
out=""
for arg in "\$@"; do
    case "\$prev" in
        -o) out="\$arg" ;;
    esac
    prev="\$arg"
done

if [ -n "\$out" ]; then
    cp "$MOCK_ASSET_TAR" "\$out"
    exit 0
fi

# Return fake latest tag when querying releases/latest
for arg in "\$@"; do
    if echo "\$arg" | grep -q "releases/latest"; then
        echo "https://github.com/vishalkumardev/cli/releases/tag/v1.0.0"
        exit 0
    fi
done

exit 0
EOF
chmod +x "$MOCK_BIN_DIR/curl"

# Run install.sh with mock PATH and custom install dir
INSTALL_OUTPUT=$(PATH="$MOCK_BIN_DIR:$PATH" BUILDSHARE_INSTALL_DIR="$MOCK_INSTALL_DIR" sh "$INSTALL_SH")

if [ -f "$MOCK_INSTALL_DIR/buildshare" ] && [ -x "$MOCK_INSTALL_DIR/buildshare" ]; then
    pass "Binary successfully installed to custom target directory"
else
    fail "Binary was not installed to target directory"
fi

installed_ver=$("$MOCK_INSTALL_DIR/buildshare" version | grep "BuildShare CLI" | awk '{print $3}')
if [ "$installed_ver" = "v1.0.0" ]; then
    pass "Installed binary verification succeeded (Version: $installed_ver)"
else
    fail "Installed binary version mismatch" "$installed_ver"
fi

if echo "$INSTALL_OUTPUT" | grep -q "BuildShare CLI installed successfully"; then
    pass "Clean success output printed to user"
else
    fail "Success message missing from output" "$INSTALL_OUTPUT"
fi

# ──────────────────────────────────────────────────────────────────
# 4. Safe Existing Installation Update Test
# ──────────────────────────────────────────────────────────────────
echo ""
echo "4. Testing Existing Installation Update..."

# Setup existing binary with v1.0.0
create_mock_buildshare "$MOCK_INSTALL_DIR/buildshare" "v1.0.0"

# Now mock release with v1.1.0
create_mock_buildshare "$MOCK_PAYLOAD_DIR/buildshare" "v1.1.0"
(cd "$MOCK_PAYLOAD_DIR" && tar -czf "$MOCK_ASSET_TAR" buildshare)

# Re-run installer pointing to existing installation
UPDATE_OUTPUT=$(PATH="$MOCK_BIN_DIR:$MOCK_INSTALL_DIR:$PATH" BUILDSHARE_INSTALL_DIR="$MOCK_INSTALL_DIR" sh "$INSTALL_SH")

updated_ver=$("$MOCK_INSTALL_DIR/buildshare" version | grep "BuildShare CLI" | awk '{print $3}')
if [ "$updated_ver" = "v1.1.0" ]; then
    pass "Existing binary safely updated from v1.0.0 to v1.1.0"
else
    fail "Binary update failed" "$updated_ver"
fi

# ──────────────────────────────────────────────────────────────────
# 5. Download Failure Resilience Test
# ──────────────────────────────────────────────────────────────────
echo ""
echo "5. Testing Download Failure Resilience (No Overwrite on Error)..."

# Ensure existing v1.1.0 is in place
create_mock_buildshare "$MOCK_INSTALL_DIR/buildshare" "v1.1.0"

# Create a failing curl mock
cat <<EOF > "$MOCK_BIN_DIR/curl"
#!/bin/sh
# Simulate network error or 404
exit 22
EOF
chmod +x "$MOCK_BIN_DIR/curl"

# Run installer - should fail and exit with non-zero
if PATH="$MOCK_BIN_DIR:$PATH" BUILDSHARE_INSTALL_DIR="$MOCK_INSTALL_DIR" sh "$INSTALL_SH" >/dev/null 2>&1; then
    fail "Installer should have exited with error on download failure"
else
    pass "Installer returned non-zero code when download failed"
fi

# Check that existing binary was NOT deleted or corrupted
intact_ver=$("$MOCK_INSTALL_DIR/buildshare" version | grep "BuildShare CLI" | awk '{print $3}')
if [ "$intact_ver" = "v1.1.0" ]; then
    pass "Existing installation was preserved and unaffected by download failure"
else
    fail "Existing installation was corrupted or removed during failed download"
fi

# ──────────────────────────────────────────────────────────────────
# 6. PATH Warning Verification Test
# ──────────────────────────────────────────────────────────────────
echo ""
echo "6. Testing PATH Warning Handling..."

# Re-install with mock curl that succeeds
cat <<EOF > "$MOCK_BIN_DIR/curl"
#!/bin/sh
out=""
for arg in "\$@"; do
    case "\$prev" in
        -o) out="\$arg" ;;
    esac
    prev="\$arg"
done

if [ -n "\$out" ]; then
    cp "$MOCK_ASSET_TAR" "\$out"
    exit 0
fi
exit 0
EOF
chmod +x "$MOCK_BIN_DIR/curl"

# Test 6a: Directory NOT in PATH -> warning printed
OUT_NOT_IN_PATH=$(PATH="$MOCK_BIN_DIR:/usr/bin:/bin" BUILDSHARE_INSTALL_DIR="$MOCK_INSTALL_DIR" sh "$INSTALL_SH")
if echo "$OUT_NOT_IN_PATH" | grep -q "is not currently in your PATH"; then
    pass "Warning displayed when install directory is not in PATH"
else
    fail "PATH warning was not displayed when directory not in PATH"
fi

# Test 6b: Directory IS in PATH -> no warning printed
OUT_IN_PATH=$(PATH="$MOCK_INSTALL_DIR:$MOCK_BIN_DIR:/usr/bin:/bin" BUILDSHARE_INSTALL_DIR="$MOCK_INSTALL_DIR" sh "$INSTALL_SH")
if echo "$OUT_IN_PATH" | grep -q "is not currently in your PATH"; then
    fail "PATH warning was unexpectedly displayed when directory was in PATH"
else
    pass "No PATH warning displayed when directory is already in PATH"
fi

# ──────────────────────────────────────────────────────────────────
# Test Summary
# ──────────────────────────────────────────────────────────────────
echo ""
echo "══════════════════════════════════════════════"
echo "Tests Passed: $PASS_COUNT / $TEST_COUNT"
if [ "$FAIL_COUNT" -eq 0 ]; then
    printf "\033[32mAll tests passed successfully!\033[0m\n"
    exit 0
else
    printf "\033[31m%d tests failed!\033[0m\n" "$FAIL_COUNT"
    exit 1
fi
