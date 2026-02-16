package ansible

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
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
// 1. Standard installation directory
// 2. Next to the running binary
// 3. Current working directory
// 4. Walking up parent directories
func FindAnsibleDir() (string, error) {
	candidates := []string{}

	// Standard installation directory
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".local", "share", "flux", "ansible"))
	}

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
		// Walk up from CWD (includes CWD itself on first iteration)
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
func RunPlaybook(ansibleDir string, extraVars map[string]interface{}, tags string, dryRun bool) error {
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
	cmd.Env = append(os.Environ(), "LC_ALL=C.UTF-8", "LANG=C.UTF-8")

	return cmd.Run()
}

func isAnsibleDir(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "playbook.yml"))
	return err == nil && !info.IsDir()
}

// OutputFunc is called for each line of output from a streaming command.
type OutputFunc func(line string)

// EnsureInstalledStreaming is like EnsureInstalled but sends output through onOutput.
func EnsureInstalledStreaming(onOutput OutputFunc) error {
	if _, err := exec.LookPath("ansible-playbook"); err == nil {
		onOutput("✓ ansible-playbook already installed")
		return nil
	}

	onOutput("Installing Ansible...")

	cmds := [][]string{
		{"sudo", "apt-get", "update", "-qq"},
		{"sudo", "apt-get", "install", "-y", "-qq", "software-properties-common"},
		{"sudo", "apt-add-repository", "--yes", "--update", "ppa:ansible/ansible"},
		{"sudo", "apt-get", "install", "-y", "-qq", "ansible"},
	}

	for _, args := range cmds {
		onOutput(fmt.Sprintf("→ %s", strings.Join(args, " ")))
		if err := runCmdStreaming(args, "", onOutput); err != nil {
			return fmt.Errorf("command %q failed: %w", strings.Join(args, " "), err)
		}
	}

	return nil
}

// RunPlaybookStreaming executes ansible-playbook, sending output line-by-line
// through onOutput. If becomePass is non-empty it is piped to ansible's stdin
// in place of --ask-become-pass.
func RunPlaybookStreaming(ansibleDir string, extraVars map[string]interface{}, tags string, dryRun bool, becomePass string, onOutput OutputFunc) error {
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

	// If we have a password, write it to a temp file for --become-password-file
	if os.Getuid() != 0 {
		if becomePass != "" {
			tmpFile, err := os.CreateTemp("", "flux-become-*")
			if err != nil {
				return fmt.Errorf("failed to create temp password file: %w", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(becomePass); err != nil {
				tmpFile.Close()
				return fmt.Errorf("failed to write temp password file: %w", err)
			}
			tmpFile.Close()

			// Restrict permissions to owner-only
			if err := os.Chmod(tmpFile.Name(), 0600); err != nil {
				return fmt.Errorf("failed to chmod temp password file: %w", err)
			}

			args = append(args, "--become-password-file", tmpFile.Name())
		} else {
			args = append(args, "--ask-become-pass")
		}
	}

	mode := "APPLY"
	if dryRun {
		mode = "DRY RUN (check mode)"
	}
	onOutput(fmt.Sprintf("[%s] ansible-playbook %s", mode, strings.Join(args, " ")))
	onOutput("")

	return runCmdStreaming([]string{"ansible-playbook"}, ansibleDir, onOutput, args[0:]...)
}

// runCmdStreaming runs a command, piping merged stdout+stderr line-by-line to onOutput.
// cmdAndArgs is the set of arguments; if extraArgs is provided they are used as the
// full arg list instead of cmdAndArgs[1:].
func runCmdStreaming(cmdAndArgs []string, dir string, onOutput OutputFunc, extraArgs ...string) error {
	name := cmdAndArgs[0]
	var args []string
	if len(extraArgs) > 0 {
		args = extraArgs
	} else {
		args = cmdAndArgs[1:]
	}

	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), "LC_ALL=C.UTF-8", "LANG=C.UTF-8", "ANSIBLE_FORCE_COLOR=0", "ANSIBLE_NOCOLOR=1")

	// Merge stdout and stderr into a single pipe
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		return err
	}

	// Read lines in a goroutine so we don't block
	done := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(pr)
		// Increase buffer for long ansible lines
		scanner.Buffer(make([]byte, 0, 64*1024), 512*1024)
		for scanner.Scan() {
			onOutput(scanner.Text())
		}
		done <- scanner.Err()
	}()

	err := cmd.Wait()
	pw.Close()
	<-done // wait for reader goroutine to finish
	pr.Close()

	return err
}
