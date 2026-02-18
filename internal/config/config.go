package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	configDir  = ".config/flux"
	configFile = "config.yaml"
)

// Config holds all user-specific settings passed to Ansible as extra vars.
type Config struct {
	Username      string   `yaml:"username"`
	Email         string   `yaml:"email"`
	GitName       string   `yaml:"git_name"`
	GitEmail      string   `yaml:"git_email"`
	GitHTTPS      bool     `yaml:"git_https"`
	DefaultShell  string   `yaml:"default_shell"`
	InstallPodman bool     `yaml:"install_podman"`
	InstallBun    bool     `yaml:"install_bun"`
	InstallGo     bool     `yaml:"install_go"`
	GoVersion     string   `yaml:"go_version,omitempty"`
	InstallDotnet bool     `yaml:"install_dotnet"`
	DotnetVersion string   `yaml:"dotnet_version,omitempty"`
	InstallPython bool     `yaml:"install_python"`
	PythonVersion string   `yaml:"python_version,omitempty"`
	InstallK9s    bool     `yaml:"install_k9s"`
	ExtraPackages []string `yaml:"extra_packages,omitempty"`
}

// validShells is the set of supported shell values.
var validShells = map[string]bool{"bash": true, "zsh": true}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Username:      whoami(),
		GitHTTPS:      true,
		DefaultShell:  "zsh",
		InstallPodman: true,
		InstallBun:    true,
		InstallGo:     true, GoVersion: "latest", InstallDotnet: true,
		DotnetVersion: "latest",
		InstallPython: true,
		PythonVersion: "latest",
		InstallK9s:    true,
		ExtraPackages: []string{"ripgrep", "fd-find", "jq", "htop"},
	}
}

// FilePath returns the full path to the config file.
func FilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, configDir, configFile)
}

// Load reads the config from disk. Returns error if it doesn't exist.
func Load() (*Config, error) {
	data, err := os.ReadFile(FilePath())
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

// Exists returns true if the config file exists.
func Exists() bool {
	_, err := os.Stat(FilePath())
	return err == nil
}

// Save writes the config to disk, creating directories as needed.
func Save(cfg *Config) error {
	path := FilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadOrCreate loads existing config or runs interactive prompts to create one.
func LoadOrCreate() (*Config, error) {
	cfg, err := Load()
	if err == nil {
		return cfg, nil
	}

	fmt.Println("No config found. Let's set up your preferences.")
	cfg, err = PromptForConfig(nil)
	if err != nil {
		return nil, err
	}
	if err := Save(cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Printf("\nConfig saved to %s\n\n", FilePath())
	return cfg, nil
}

// PromptForConfig runs interactive prompts. If existing is non-nil, its values are used as defaults.
func PromptForConfig(existing *Config) (*Config, error) {
	reader := bufio.NewReader(os.Stdin)
	cfg := DefaultConfig()

	if existing != nil {
		*cfg = *existing
	}

	var err error

	cfg.Username, err = prompt(reader, "Username", cfg.Username, whoami())
	if err != nil {
		return nil, err
	}

	cfg.Email, err = prompt(reader, "Email", cfg.Email, "")
	if err != nil {
		return nil, err
	}

	cfg.GitName, err = prompt(reader, "Git display name", cfg.GitName, cfg.Username)
	if err != nil {
		return nil, err
	}

	cfg.GitEmail, err = prompt(reader, "Git email", cfg.GitEmail, cfg.Email)
	if err != nil {
		return nil, err
	}

	cfg.GitHTTPS, err = promptBool(reader, "Use HTTPS for GitHub (instead of SSH)?", cfg.GitHTTPS)
	if err != nil {
		return nil, err
	}

	for {
		cfg.DefaultShell, err = prompt(reader, "Default shell (bash/zsh)", cfg.DefaultShell, "zsh")
		if err != nil {
			return nil, err
		}
		if validShells[cfg.DefaultShell] {
			break
		}
		fmt.Println("    Invalid shell. Please enter 'bash' or 'zsh'.")
	}

	cfg.InstallPodman, err = promptBool(reader, "Install Podman (remote client)?", cfg.InstallPodman)
	if err != nil {
		return nil, err
	}
	if cfg.InstallPodman {
		cfg.PodmanWSLDistro, err = prompt(reader, "Podman WSL distro name", cfg.PodmanWSLDistro, "podman-machine")
		if err != nil {
			return nil, err
		}
		cfg.PodmanWSLHost, err = prompt(reader, "Podman WSL host", cfg.PodmanWSLHost, "localhost")
		if err != nil {
			return nil, err
		}
		cfg.PodmanWSLPort, err = prompt(reader, "Podman WSL SSH port", cfg.PodmanWSLPort, "22")
		if err != nil {
			return nil, err
		}
	}

	cfg.InstallBun, err = promptBool(reader, "Install Bun?", cfg.InstallBun)
	if err != nil {
		return nil, err
	}

	cfg.InstallGo, err = promptBool(reader, "Install Go?", cfg.InstallGo)
	if err != nil {
		return nil, err
	}

	cfg.InstallDotnet, err = promptBool(reader, "Install .NET SDK?", cfg.InstallDotnet)
	if err != nil {
		return nil, err
	}
	if cfg.InstallDotnet {
		cfg.DotnetVersion, err = prompt(reader, ".NET SDK version (or 'latest')", cfg.DotnetVersion, "latest")
		if err != nil {
			return nil, err
		}
	}

	cfg.InstallPython, err = promptBool(reader, "Install Python?", cfg.InstallPython)
	if err != nil {
		return nil, err
	}
	if cfg.InstallPython {
		cfg.PythonVersion, err = prompt(reader, "Python version (or 'latest')", cfg.PythonVersion, "latest")
		if err != nil {
			return nil, err
		}
	}

	cfg.InstallK9s, err = promptBool(reader, "Install k9s (Kubernetes TUI)?", cfg.InstallK9s)
	if err != nil {
		return nil, err
	}

	pkgs, err := prompt(reader, "Extra apt packages (comma-separated)", strings.Join(cfg.ExtraPackages, ", "), "ripgrep, fd-find, jq, htop")
	if err != nil {
		return nil, err
	}
	cfg.ExtraPackages = nil
	for _, p := range strings.Split(pkgs, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			cfg.ExtraPackages = append(cfg.ExtraPackages, p)
		}
	}

	return cfg, nil
}

// Marshal returns the YAML representation of the config.
func (c *Config) Marshal() ([]byte, error) {
	return yaml.Marshal(c)
}

// ToExtraVars converts the config to a typed map for Ansible --extra-vars.
// Booleans are passed as real booleans and lists as real lists in the JSON.
//
// Version fields set to "latest" are intentionally omitted so that the
// playbook-level defaults take effect.  Ansible extra-vars have the highest
// variable precedence, which would prevent the roles' set_fact tasks from
// resolving "latest" to a real version number.
func (c *Config) ToExtraVars() map[string]interface{} {
	vars := map[string]interface{}{
		"username":          c.Username,
		"email":             c.Email,
		"git_name":          c.GitName,
		"git_email":         c.GitEmail,
		"git_https":         c.GitHTTPS,
		"default_shell":     c.DefaultShell,
		"install_podman":    c.InstallPodman,
		"podman_wsl_distro": c.PodmanWSLDistro,
		"podman_wsl_host":   c.PodmanWSLHost,
		"podman_wsl_port":   c.PodmanWSLPort,
		"install_bun":       c.InstallBun,
		"install_go":        c.InstallGo,
		"install_dotnet":    c.InstallDotnet,
		"install_python":    c.InstallPython,
		"install_k9s":       c.InstallK9s,
		"extra_packages":    c.ExtraPackages,
	}

	// Only pass version extra-vars when a specific version is requested.
	// When "latest", the Ansible roles resolve the version themselves via
	// API calls; omitting the extra-var lets set_fact override the
	// playbook-level default.
	// Note: Go always installs latest; no version pinning supported.
	if !strings.EqualFold(c.DotnetVersion, "latest") {
		vars["dotnet_version"] = c.DotnetVersion
	}
	if !strings.EqualFold(c.PythonVersion, "latest") {
		vars["python_version"] = c.PythonVersion
	}

	if c.ExtraPackages == nil {
		vars["extra_packages"] = []string{}
	}
	return vars
}

// AvailableRoles returns the default role tag names the user can select.
// If an ansible directory is provided, roles are discovered dynamically.
func AvailableRoles() []string {
	return []string{"base", "git-config", "shell", "podman", "golang", "bun", "dotnet", "python", "k9s"}
}

// DiscoverRoles scans the ansible/roles/ directory and returns role names.
// Falls back to the hardcoded list if the directory cannot be read.
func DiscoverRoles(ansibleDir string) []string {
	rolesDir := filepath.Join(ansibleDir, "roles")
	entries, err := os.ReadDir(rolesDir)
	if err != nil {
		return AvailableRoles()
	}
	var roles []string
	for _, e := range entries {
		if e.IsDir() {
			// Verify it has a tasks/main.yml
			tasksFile := filepath.Join(rolesDir, e.Name(), "tasks", "main.yml")
			if _, err := os.Stat(tasksFile); err == nil {
				roles = append(roles, e.Name())
			}
		}
	}
	if len(roles) == 0 {
		return AvailableRoles()
	}
	return roles
}

// --- helpers ---

func prompt(reader *bufio.Reader, label, current, fallback string) (string, error) {
	def := current
	if def == "" {
		def = fallback
	}
	if def != "" {
		fmt.Printf("  %s [%s]: ", label, def)
	} else {
		fmt.Printf("  %s: ", label)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def, nil
	}
	return line, nil
}

func promptBool(reader *bufio.Reader, label string, current bool) (bool, error) {
	def := "y"
	if !current {
		def = "n"
	}
	fmt.Printf("  %s [%s]: ", label, def)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return current, nil
	}
	return line == "y" || line == "yes", nil
}

func whoami() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return ""
}

// BoolStr converts a bool to "true" or "false".
func BoolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
