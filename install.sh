#!/usr/bin/env bash
set -euo pipefail

REPO="https://github.com/jaydubyaeey/flux.git"
INSTALL_DIR="$HOME/.local/share/flux"
BIN_DIR="$HOME/.local/bin"
BIN="$BIN_DIR/flux"

echo ""
echo "  ⚡ flux — WSL bootstrap"
echo ""

# Ensure ~/.local/bin exists
mkdir -p "$BIN_DIR"

# Install git if missing
if ! command -v git &>/dev/null; then
    echo "→ Installing git..."
    sudo apt-get update -qq && sudo apt-get install -y -qq git
fi

# Install Go if missing
if ! command -v go &>/dev/null; then
    echo "→ Installing Go..."
    GO_VERSION="1.23.4"
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go.tar.gz
    sudo tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz
    export PATH="$PATH:/usr/local/go/bin"
fi

# Clone or update the repo
if [ -d "$INSTALL_DIR" ]; then
    echo "→ Updating existing installation..."
    cd "$INSTALL_DIR"
    git pull --ff-only
else
    echo "→ Cloning flux..."
    git clone "$REPO" "$INSTALL_DIR"
    cd "$INSTALL_DIR"
fi

# Build
echo "→ Building..."
go build -o "$BIN" ./cmd/flux

# Ensure ~/.local/bin is on PATH
if ! echo "$PATH" | grep -q "$BIN_DIR"; then
    export PATH="$BIN_DIR:$PATH"
    # Add to both bashrc and zshrc if they exist
    for rc in "$HOME/.bashrc" "$HOME/.zshrc"; do
        if [ -f "$rc" ] && ! grep -q "$BIN_DIR" "$rc"; then
            echo "export PATH=\"$BIN_DIR:\$PATH\"" >> "$rc"
        fi
    done
    # Always ensure bashrc has it (fresh WSL)
    if ! grep -q "$BIN_DIR" "$HOME/.bashrc"; then
        echo "export PATH=\"$BIN_DIR:\$PATH\"" >> "$HOME/.bashrc"
    fi
fi

echo ""
echo "✓ flux installed to $BIN"
echo ""
echo "Launch the interactive TUI:"
echo "  flux"
echo ""
echo "Or run directly:"
echo "  flux run             # apply setup"
echo "  flux run --dry-run   # preview without changes"
echo ""
