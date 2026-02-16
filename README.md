# ⚡ flux

A Go CLI with an interactive TUI that wraps Ansible playbooks to bootstrap and configure a fresh WSL instance. Run it once from GitHub and get your entire dev environment configured exactly how you like it.

## Quick Start

On a fresh WSL instance:

```bash
# One-liner install
curl -fsSL https://raw.githubusercontent.com/jaydubyaeey/flux/main/install.sh | bash

# Then launch the TUI
flux
```

Or manually:

```bash
git clone https://github.com/jaydubyaeey/flux.git
cd flux
go build -o flux ./cmd/flux
./flux
```

## What It Does

1. **First run** — prompts for username, email, git config, tool preferences, and saves them to `~/.config/flux/config.yaml`
2. **Installs Ansible** if not already present
3. **Runs Ansible playbooks** with your config values injected as extra vars
4. **Subsequent runs** — reads existing config and re-runs playbooks (idempotent)
5. **Dry run mode** — preview what Ansible would change without applying anything

## Interactive TUI

Launch `flux` with no arguments for the interactive TUI:

```
⚡ flux
WSL bootstrap & configuration

▸ Run Setup     Apply configuration to this machine
  Dry Run       Preview changes without applying (--check)
  Configure     View or edit your settings
  Update        Pull latest changes and rebuild flux
  Quit          Exit flux
```

Navigate with arrow keys, select roles to run, toggle dry-run mode — all without memorising flags.

## CLI Commands

| Command | Description |
|---------|-------------|
| `flux` | Launch interactive TUI |
| `flux run` | Run the full setup (prompts for config on first run) |
| `flux run --dry-run` | Preview changes without applying |
| `flux run --tags dev-tools,shell` | Run only specific tagged roles |
| `flux run --dry-run --tags base` | Dry-run a specific role |
| `flux config show` | Print current config |
| `flux config edit` | Re-run the interactive config prompts |
| `flux config path` | Print the config file path |
| `flux update` | Pull latest changes and rebuild flux |
| `flux version` | Print version |

## Project Structure

```
flux/
├── .github/
│   └── workflows/
│       └── pages.yml                # GitHub Action to deploy GitHub Pages
├── cmd/
│   └── flux/
│       └── main.go                  # CLI entrypoint & arg parsing
├── internal/
│   ├── config/
│   │   └── config.go                # Config loading, saving, prompting
│   ├── ansible/
│   │   └── runner.go                # Ansible install check & playbook runner
│   ├── tui/
│   │   ├── tui.go                   # Bubbletea TUI (menus, role select, config edit)
│   │   └── styles.go                # Lipgloss styles & colours
│   └── updater/
│       └── updater.go               # Self-update (git pull + rebuild)
├── ansible/
│   ├── playbook.yml                 # Main playbook
│   ├── inventory.ini                # Local inventory (localhost)
│   └── roles/
│       ├── base/
│       │   └── tasks/main.yml       # Core packages (curl, git, build-essential, etc.)
│       ├── shell/
│       │   ├── tasks/main.yml       # Zsh, oh-my-zsh, starship, dotfiles
│       │   └── templates/.zshrc.j2
│       ├── dev-tools/
│       │   └── tasks/main.yml       # Podman, Go, Bun, .NET, Python, k9s
│       └── git-config/
│           ├── tasks/main.yml
│           └── templates/.gitconfig.j2
├── docs/
│   └── index.html                   # GitHub Pages landing page with install command
├── .gitignore
├── install.sh                       # One-liner bootstrap script
├── go.mod
├── go.sum
└── README.md
```

## Configuration

Config lives at `~/.config/flux/config.yaml`:

```yaml
username: johndoe
email: john@example.com
git_name: John Doe
git_email: john@example.com
git_https: true
default_shell: zsh
install_podman: true
podman_wsl_distro: podman-machine
podman_wsl_host: localhost
podman_wsl_port: "22"
install_bun: true
install_go: true
go_version: "1.26"
install_dotnet: true
dotnet_version: "10.0"
install_python: true
python_version: "3.13"
extra_packages:
  - ripgrep
  - fd-find
  - jq
  - htop
```

You can edit this file directly or use `flux config edit` / the TUI.

## Dry Run

Dry run passes `--check --diff` to Ansible, which shows what **would** change without modifying your system. Useful for:

- Testing your playbooks before applying
- Verifying idempotency
- Reviewing changes after editing config

```bash
# CLI
flux run --dry-run

# Or from the TUI — select "Dry Run" from the main menu
```

## Ansible Roles

| Role | Tag | What it does |
|------|-----|-------------|
| **base** | `base` | Updates apt, installs essential packages (build-essential, curl, git, etc.) |
| **git-config** | `git-config` | Deploys ~/.gitconfig from template with your name/email, optional HTTPS-for-GitHub rewrite |
| **shell** | `shell` | Installs zsh, oh-my-zsh, plugins, starship prompt, deploys .zshrc |
| **dev-tools** | `dev-tools` | Installs Podman (remote client + compose), Go, Bun, .NET SDK, Python, k9s — each gated by config flags |

## Customising

### Adding a new role

1. Create `ansible/roles/<name>/tasks/main.yml`
2. Add the role to `ansible/playbook.yml` with a tag
3. Add the tag to `AvailableRoles()` in `internal/config/config.go`
4. If it needs config values, add fields to the `Config` struct and prompts

### Adding config fields

Edit `internal/config/config.go` — add the field to the `Config` struct, update `ToExtraVars()`, and add a prompt in `PromptForConfig()`. The value will be available in Ansible as an extra var automatically.

## Requirements

- WSL2 (Ubuntu recommended)
- Go 1.23+ (the install script handles this)
- Internet connection (first run)

## Self-Update

Flux can update itself by pulling the latest source and rebuilding:

```bash
flux update
```

Or select **Update** from the TUI main menu.
