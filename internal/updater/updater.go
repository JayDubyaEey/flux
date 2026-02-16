package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	defaultInstallDir = ".local/share/flux"
	defaultBinPath    = ".local/bin/flux"
)

// InstallDir returns the path where flux was cloned.
func InstallDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, defaultInstallDir)
}

// BinPath returns the path to the flux binary.
func BinPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, defaultBinPath)
}

// Update pulls the latest changes from git and rebuilds the binary.
func Update() error {
	dir := InstallDir()

	// Check the install directory exists
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return fmt.Errorf("flux install directory not found at %s — was it installed via install.sh?", dir)
	}

	// Git fetch and check for updates
	fmt.Println("→ Checking for updates...")
	fetch := exec.Command("git", "fetch", "--quiet")
	fetch.Dir = dir
	fetch.Stdout = os.Stdout
	fetch.Stderr = os.Stderr
	if err := fetch.Run(); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}

	// Check if we're behind
	status := exec.Command("git", "status", "-uno")
	status.Dir = dir
	out, err := status.Output()
	if err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}

	if strings.Contains(string(out), "Your branch is up to date") {
		fmt.Println("✓ Already up to date")
		return nil
	}

	// Pull
	fmt.Println("→ Pulling latest changes...")
	pull := exec.Command("git", "pull", "--ff-only")
	pull.Dir = dir
	pull.Stdout = os.Stdout
	pull.Stderr = os.Stderr
	if err := pull.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	// Rebuild
	fmt.Println("→ Rebuilding...")
	binPath := BinPath()

	// Ensure Go is on PATH (may have been installed to /usr/local/go/bin)
	goPath, err := exec.LookPath("go")
	if err != nil {
		goPath = "/usr/local/go/bin/go"
		if _, statErr := os.Stat(goPath); statErr != nil {
			return fmt.Errorf("go not found on PATH or in /usr/local/go/bin — is Go installed?")
		}
	}

	build := exec.Command(goPath, "build", "-o", binPath, "./cmd/flux")
	build.Dir = dir
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Printf("✓ Updated successfully (%s)\n", binPath)
	return nil
}
