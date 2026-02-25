#!/bin/sh
set -eu

REPO="markx3/agentboard"
BINARY="agentboard"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

main() {
    detect_platform
    get_version
    download_and_install
    print_success
}

detect_platform() {
    OS=$(uname -s)
    ARCH=$(uname -m)

    case "$OS" in
        Darwin) OS="darwin" ;;
        Linux)  OS="linux" ;;
        *)
            echo "Error: Unsupported operating system: $OS" >&2
            echo "agentboard supports macOS and Linux." >&2
            exit 1
            ;;
    esac

    case "$ARCH" in
        x86_64)         ARCH="amd64" ;;
        arm64|aarch64)  ARCH="arm64" ;;
        *)
            echo "Error: Unsupported architecture: $ARCH" >&2
            echo "agentboard supports amd64 and arm64." >&2
            exit 1
            ;;
    esac

    echo "Detected platform: ${OS}/${ARCH}"
}

get_version() {
    if [ -n "${AGENTBOARD_VERSION:-}" ]; then
        VERSION="$AGENTBOARD_VERSION"
        echo "Installing agentboard $VERSION (pinned)"
        return
    fi

    echo "Fetching latest release..."
    RESPONSE=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>&1) || {
        HTTP_CODE=$?
        case "$RESPONSE" in
            *"rate limit"*|*"403"*)
                echo "Error: GitHub API rate limited. Try again later or install from source:" >&2
                echo "  go install github.com/markx3/agentboard/cmd/agentboard@latest" >&2
                exit 1
                ;;
            *"404"*|*"Not Found"*)
                echo "Error: No releases found for ${REPO}." >&2
                echo "Install from source instead:" >&2
                echo "  go install github.com/markx3/agentboard/cmd/agentboard@latest" >&2
                exit 1
                ;;
            *)
                echo "Error: Failed to fetch latest release (exit code $HTTP_CODE)." >&2
                echo "Install from source instead:" >&2
                echo "  go install github.com/markx3/agentboard/cmd/agentboard@latest" >&2
                exit 1
                ;;
        esac
    }

    VERSION=$(echo "$RESPONSE" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        echo "Error: Could not determine latest version." >&2
        exit 1
    fi

    echo "Latest version: $VERSION"
}

download_and_install() {
    ARCHIVE="${BINARY}-${OS}-${ARCH}.tar.gz"
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"
    CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

    TMPDIR=$(mktemp -d)
    trap 'rm -rf "$TMPDIR"' EXIT

    # Check for existing installation
    if command -v "$BINARY" > /dev/null 2>&1; then
        OLD_VERSION=$("$BINARY" --version 2>/dev/null | awk '{print $NF}') || OLD_VERSION="unknown"
        echo "Existing installation: $OLD_VERSION"
    fi

    echo "Downloading ${ARCHIVE}..."
    curl -fSL --progress-bar "$URL" -o "${TMPDIR}/${ARCHIVE}" || {
        echo "Error: Failed to download ${URL}" >&2
        echo "Check that release $VERSION exists at https://github.com/${REPO}/releases" >&2
        exit 1
    }

    # Verify checksum
    echo "Verifying checksum..."
    if curl -fsSL "$CHECKSUMS_URL" -o "${TMPDIR}/checksums.txt" 2>/dev/null; then
        EXPECTED=$(grep "$ARCHIVE" "${TMPDIR}/checksums.txt" | awk '{print $1}')
        if [ -n "$EXPECTED" ]; then
            if command -v sha256sum > /dev/null 2>&1; then
                ACTUAL=$(sha256sum "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')
            elif command -v shasum > /dev/null 2>&1; then
                ACTUAL=$(shasum -a 256 "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')
            else
                echo "Warning: No sha256sum or shasum available, skipping checksum verification." >&2
                ACTUAL="$EXPECTED"
            fi

            if [ "$EXPECTED" != "$ACTUAL" ]; then
                echo "Error: Checksum verification failed." >&2
                echo "  Expected: $EXPECTED" >&2
                echo "  Got:      $ACTUAL" >&2
                exit 1
            fi
            echo "Checksum verified."
        else
            echo "Warning: Archive not found in checksums.txt, skipping verification." >&2
        fi
    else
        echo "Warning: Could not download checksums.txt, skipping verification." >&2
    fi

    # Extract
    tar -xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"

    # Install
    if [ -w "$INSTALL_DIR" ]; then
        mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    elif command -v sudo > /dev/null 2>&1; then
        echo "Installing to ${INSTALL_DIR} (requires sudo)..."
        sudo mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    else
        echo "Error: ${INSTALL_DIR} is not writable and sudo is not available." >&2
        echo "Run with a custom install directory:" >&2
        echo "  INSTALL_DIR=~/.local/bin curl -fsSL ... | sh" >&2
        exit 1
    fi

    chmod +x "${INSTALL_DIR}/${BINARY}"
}

print_success() {
    INSTALLED_VERSION=$("${INSTALL_DIR}/${BINARY}" --version 2>/dev/null | awk '{print $NF}') || INSTALLED_VERSION="$VERSION"

    echo ""
    echo "agentboard ${INSTALLED_VERSION} installed to ${INSTALL_DIR}/${BINARY}"
    echo ""
    echo "Get started:"
    echo "  agentboard init    # initialize in your project"
    echo "  agentboard         # launch the TUI"
    echo ""
    echo "Prerequisites: tmux (3.0+), gh CLI (2.0+)"
}

main
