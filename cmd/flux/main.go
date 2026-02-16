package main

import (
	"fmt"
	"os"

	"github.com/jaydubyaeey/flux/internal/config"
	"github.com/jaydubyaeey/flux/internal/tui"
	"github.com/jaydubyaeey/flux/internal/updater"
)

const version = "0.1.0"

const usage = `flux - Bootstrap and configure your WSL instance

Usage:
  flux                            Launch interactive TUI
  flux run [--dry-run] [--tags t] Run setup playbooks
  flux config show                Show current configuration
  flux config edit                Re-run interactive config prompts
  flux config path                Print config file path
  flux update                     Pull latest changes and rebuild
  flux version                    Print version
  flux help                       Show this help message

Flags:
  --dry-run     Run Ansible in check mode (no changes applied)
  --tags <t>    Comma-separated list of role tags to run
`

func main() {
	if len(os.Args) < 2 {
		// No args — launch TUI
		tui.Run()
		return
	}

	switch os.Args[1] {
	case "run":
		cmdRun()
	case "config":
		if len(os.Args) < 3 {
			fmt.Println("Usage: flux config [show|edit|path]")
			os.Exit(1)
		}
		cmdConfig(os.Args[2])
	case "update":
		if err := updater.Update(); err != nil {
			fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
			os.Exit(1)
		}
	case "version", "--version", "-v":
		fmt.Printf("flux %s\n", version)
	case "help", "--help", "-h":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		fmt.Print(usage)
		os.Exit(1)
	}
}

func cmdRun() {
	cfg, err := config.LoadOrCreate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error with config: %v\n", err)
		os.Exit(1)
	}

	var tags string
	var dryRun bool
	for i, arg := range os.Args {
		if arg == "--tags" && i+1 < len(os.Args) {
			tags = os.Args[i+1]
		}
		if arg == "--dry-run" {
			dryRun = true
		}
	}

	tui.RunPlaybookCLI(cfg, tags, dryRun)
}

func cmdConfig(sub string) {
	switch sub {
	case "show":
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "No config found. Run 'flux' to create one.\n")
			os.Exit(1)
		}
		out, _ := cfg.Marshal()
		fmt.Println(string(out))

	case "edit":
		cfg, loadErr := config.Load()
		if loadErr != nil && config.Exists() {
			// Config file exists but is corrupt
			fmt.Fprintf(os.Stderr, "Warning: config file is corrupt: %v\n", loadErr)
			fmt.Fprintf(os.Stderr, "Starting with defaults. Your old config will be overwritten on save.\n\n")
			cfg = nil
		}
		cfg, err := config.PromptForConfig(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Config updated.")

	case "path":
		fmt.Println(config.FilePath())

	default:
		fmt.Fprintf(os.Stderr, "Unknown config command: %s\n", sub)
		fmt.Println("Usage: flux config [show|edit|path]")
		os.Exit(1)
	}
}
