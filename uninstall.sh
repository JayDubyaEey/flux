#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="$HOME/.local/share/flux"
BIN="$HOME/.local/bin/flux"
CONFIG_DIR="$HOME/.config/flux"

echo ""
echo "  ⚡ flux — uninstall"
echo ""

# --- Confirm ---
# Read from /dev/tty so prompts work when piped (curl ... | bash)
read -rp "This will remove flux from your system. Continue? [y/N] " confirm < /dev/tty
if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

# --- Remove binary ---
if [ -f "$BIN" ]; then
    echo "→ Removing binary ($BIN)..."
    rm -f "$BIN"
else
    echo "  Binary not found, skipping."
fi

# --- Remove cloned repo ---
if [ -d "$INSTALL_DIR" ]; then
    echo "→ Removing install directory ($INSTALL_DIR)..."
    rm -rf "$INSTALL_DIR"
else
    echo "  Install directory not found, skipping."
fi

# --- Remove config ---
if [ -d "$CONFIG_DIR" ]; then
    read -rp "Remove configuration ($CONFIG_DIR)? [y/N] " remove_config < /dev/tty
    if [[ "$remove_config" =~ ^[Yy]$ ]]; then
        echo "→ Removing config..."
        rm -rf "$CONFIG_DIR"
    else
        echo "  Keeping config."
    fi
fi

# --- Clean PATH entries from shell rc files ---
echo "→ Cleaning PATH entries from shell rc files..."
for rc in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
    if [ -f "$rc" ]; then
        # Remove lines added by install.sh
        sed -i '\|\.local/bin|d' "$rc"
        sed -i '\|/usr/local/go/bin|d' "$rc"
    fi
done

echo ""
echo "✓ flux has been uninstalled."
echo ""
echo "Note: Ansible, Go, and any packages installed by flux roles"
echo "were not removed. Remove them manually if desired:"
echo "  sudo rm -rf /usr/local/go"
echo "  sudo apt remove ansible -y"
echo ""
