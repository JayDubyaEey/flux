package ansible

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// EnsureInstalled checks if ansible-playbook is available and installs it if not.
func EnsureInstalled() error {
	if _, err := exec.LookPath("ansible-playbook"); err == nil {
		return nil
	}

	fmt.Println("Installing Ansible...")

	cmds := [][]string{
		{"sudo", "apt-get", "update", "-qq"},
		{"sudo", "apt-get", "install", "-y", "-qq", "software-properties-common"},
		{"sudo", "apt-add-repository", "--yes", "--update", "ppa:ansible/ansible"},
		{"sudo", "apt-get", "install", "-y", "-qq", "ansible"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("command %q failed: %w", strings.Join(args, " "), err)
		}
	}

	return nil
}

// FindAnsibleDir locates the ansible/ directory by checking:
// 1. Next to the running binary
// 2. Current working directory
// 3. Walking up parent directories
func FindAnsibleDir() (string, error) {
	candidates := []string{}

	// Relative to executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "ansible"),
			filepath.Join(exeDir, "..", "ansible"),
			filepath.Join(exeDir, "..", "..", "ansible"),
		)
	}

	// Relative to CWD
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "ansible"))
		// Walk up from CWD
		dir := cwd
		for i := 0; i < 10; i++ {
			candidates = append(candidates, filepath.Join(dir, "ansible"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	for _, c := range candidates {
		if isAnsibleDir(c) {
			return filepath.Clean(c), nil
		}
	}

	return "", fmt.Errorf("cannot find ansible/ directory containing playbook.yml")
}

// RunPlaybook executes ansible-playbook with the given options.
func RunPlaybook(ansibleDir string, extraVars map[string]string, tags string, dryRun bool) error {
	playbook := filepath.Join(ansibleDir, "playbook.yml")
	inventory := filepath.Join(ansibleDir, "inventory.ini")

	if _, err := os.Stat(playbook); err != nil {
		return fmt.Errorf("playbook not found: %s", playbook)
	}

	args := []string{
		playbook,
		"-i", inventory,
		"--connection=local",
	}

	if len(extraVars) > 0 {
		varsJSON, err := json.Marshal(extraVars)
		if err != nil {
			return fmt.Errorf("failed to marshal extra vars: %w", err)
		}
		args = append(args, "--extra-vars", string(varsJSON))
	}

	if tags != "" {
		args = append(args, "--tags", tags)
	}

	if dryRun {
		args = append(args, "--check", "--diff")
	}

	// Ask for become password if not root
	if os.Getuid() != 0 {
		args = append(args, "--ask-become-pass")
	}

	mode := "APPLY"
	if dryRun {
		mode = "DRY RUN (check mode)"
	}
	fmt.Printf("[%s] ansible-playbook %s\n\n", mode, strings.Join(args, " "))

	cmd := exec.Command("ansible-playbook", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Dir = ansibleDir

	return cmd.Run()
}

func isAnsibleDir(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "playbook.yml"))
	return err == nil && !info.IsDir()
}
