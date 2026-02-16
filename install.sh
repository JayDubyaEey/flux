#!/usr/bin/env bash
set -euo pipefail

REPO="https://github.com/jaydubyaeey/flux.git"
INSTALL_DIR="$HOME/.local/share/flux"
BIN_DIR="$HOME/.local/bin"
BIN="$BIN_DIR/flux"
GO_INSTALL_DIR="/usr/local/go"
GO_BIN="$GO_INSTALL_DIR/bin/go"
GO_FALLBACK_VERSION="1.23.4"

echo ""
echo "  ⚡ flux — WSL bootstrap"
echo ""

# Ensure ~/.local/bin exists
mkdir -p "$BIN_DIR"

# Install git and curl if missing
if ! command -v git &>/dev/null || ! command -v curl &>/dev/null; then
    echo "→ Installing prerequisites..."
    sudo apt-get update -qq && sudo apt-get install -y -qq git curl
fi

# --- Go installation ---
# Resolve the latest stable Go version from go.dev (same source as the ansible role).
resolve_go_version() {
    local raw
    raw=$(curl -fsSL "https://go.dev/VERSION?m=text" 2>/dev/null || true)
    if [ -n "$raw" ]; then
        # The first line looks like "go1.23.4"; strip the "go" prefix.
        echo "$raw" | head -n1 | sed 's/^go//'
    else
        echo "$GO_FALLBACK_VERSION"
    fi
}

install_go() {
    local ver="$1"
    echo "→ Installing Go ${ver}..."
    curl -fsSL "https://go.dev/dl/go${ver}.linux-amd64.tar.gz" -o /tmp/go.tar.gz
    # Remove any previous /usr/local/go to avoid stale files
    sudo rm -rf "$GO_INSTALL_DIR"
    sudo tar -C /usr/local -xzf /tmp/go.tar.gz
    rm -f /tmp/go.tar.gz
}

# Always put /usr/local/go/bin on PATH for this session
export PATH="$GO_INSTALL_DIR/bin:$PATH"

TARGET_GO_VERSION=$(resolve_go_version)

needs_go_install=false
if [ -x "$GO_BIN" ]; then
    INSTALLED_GO_VERSION=$("$GO_BIN" version 2>/dev/null | grep -oP 'go\K[0-9]+\.[0-9]+(\.[0-9]+)?' || true)
    if [ "$INSTALLED_GO_VERSION" = "$TARGET_GO_VERSION" ]; then
        echo "→ Go ${TARGET_GO_VERSION} is already installed at ${GO_INSTALL_DIR}, skipping."
    else
        echo "→ Go ${INSTALLED_GO_VERSION:-unknown} found, upgrading to ${TARGET_GO_VERSION}..."
        needs_go_install=true
    fi
else
    needs_go_install=true
fi

if [ "$needs_go_install" = true ]; then
    install_go "$TARGET_GO_VERSION"
fi

# Verify Go is usable
if ! "$GO_BIN" version &>/dev/null; then
    echo "✗ Go installation failed — $GO_BIN is not executable." >&2
    exit 1
fi

# --- Ensure /usr/local/go/bin is on PATH persistently ---
go_path_line='export PATH="/usr/local/go/bin:$PATH"'
for rc in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
    if [ -f "$rc" ] && ! grep -qF '/usr/local/go/bin' "$rc"; then
        echo "$go_path_line" >> "$rc"
    fi
done
# Always ensure bashrc has it (fresh WSL may not have .bashrc yet)
if [ ! -f "$HOME/.bashrc" ] || ! grep -qF '/usr/local/go/bin' "$HOME/.bashrc"; then
    echo "$go_path_line" >> "$HOME/.bashrc"
fi

# --- Clone or update the repo ---
if [ -d "$INSTALL_DIR/.git" ]; then
    echo "→ Updating existing installation..."
    cd "$INSTALL_DIR"
    git pull --ff-only
else
    echo "→ Cloning flux..."
    rm -rf "$INSTALL_DIR"
    git clone "$REPO" "$INSTALL_DIR"
    cd "$INSTALL_DIR"
fi

# Build
echo "→ Building..."
"$GO_BIN" build -o "$BIN" ./cmd/flux

# --- Ensure ~/.local/bin is on PATH persistently ---
if ! echo "$PATH" | grep -q "$BIN_DIR"; then
    export PATH="$BIN_DIR:$PATH"
fi
for rc in "$HOME/.bashrc" "$HOME/.zshrc"; do
    if [ -f "$rc" ] && ! grep -qF "$BIN_DIR" "$rc"; then
        echo "export PATH=\"$BIN_DIR:\$PATH\"" >> "$rc"
    fi
done
if ! grep -qF "$BIN_DIR" "$HOME/.bashrc" 2>/dev/null; then
    echo "export PATH=\"$BIN_DIR:\$PATH\"" >> "$HOME/.bashrc"
fi

echo ""
echo "✓ flux installed to $BIN"
echo ""
echo "To use flux, reload your shell or run this command:"
echo "  source ~/.$(basename $SHELL)rc"
echo ""
echo "Then launch the interactive TUI:"
echo "  flux"
echo ""
echo "Or run directly:"
echo "  flux run             # apply setup"
echo "  flux run --dry-run   # preview without changes"
echo ""

# Attempt to reload shell config for current session
if [ -n "${BASH_VERSION:-}" ]; then
    . "$HOME/.bashrc" 2>/dev/null || true
elif [ -n "${ZSH_VERSION:-}" ]; then
    . "$HOME/.zshrc" 2>/dev/null || true
fi
