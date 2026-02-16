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
	Username        string   `yaml:"username"`
	Email           string   `yaml:"email"`
	GitName         string   `yaml:"git_name"`
	GitEmail        string   `yaml:"git_email"`
	GitHTTPS        bool     `yaml:"git_https"`
	DefaultShell    string   `yaml:"default_shell"`
	InstallPodman   bool     `yaml:"install_podman"`
	PodmanWSLDistro string   `yaml:"podman_wsl_distro,omitempty"`
	PodmanWSLHost   string   `yaml:"podman_wsl_host,omitempty"`
	PodmanWSLPort   string   `yaml:"podman_wsl_port,omitempty"`
	InstallBun      bool     `yaml:"install_bun"`
	InstallGo       bool     `yaml:"install_go"`
	GoVersion       string   `yaml:"go_version,omitempty"`
	InstallDotnet   bool     `yaml:"install_dotnet"`
	DotnetVersion   string   `yaml:"dotnet_version,omitempty"`
	InstallPython   bool     `yaml:"install_python"`
	PythonVersion   string   `yaml:"python_version,omitempty"`
	ExtraPackages   []string `yaml:"extra_packages,omitempty"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Username:        whoami(),
		GitHTTPS:        true,
		DefaultShell:    "zsh",
		InstallPodman:   true,
		PodmanWSLDistro: "podman-machine",
		PodmanWSLHost:   "localhost",
		PodmanWSLPort:   "22",
		InstallBun:      true,
		InstallGo:       true,
		GoVersion:       "1.26",
		InstallDotnet:   true,
		DotnetVersion:   "10.0",
		InstallPython:   true,
		PythonVersion:   "3.13",
		ExtraPackages:   []string{"ripgrep", "fd-find", "jq", "htop"},
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

	cfg.DefaultShell, err = prompt(reader, "Default shell (bash/zsh)", cfg.DefaultShell, "zsh")
	if err != nil {
		return nil, err
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
	if cfg.InstallGo {
		cfg.GoVersion, err = prompt(reader, "Go version", cfg.GoVersion, "1.23")
		if err != nil {
			return nil, err
		}
	}

	cfg.InstallDotnet, err = promptBool(reader, "Install .NET SDK?", cfg.InstallDotnet)
	if err != nil {
		return nil, err
	}
	if cfg.InstallDotnet {
		cfg.DotnetVersion, err = prompt(reader, ".NET SDK version", cfg.DotnetVersion, "8.0")
		if err != nil {
			return nil, err
		}
	}

	cfg.InstallPython, err = promptBool(reader, "Install Python?", cfg.InstallPython)
	if err != nil {
		return nil, err
	}
	if cfg.InstallPython {
		cfg.PythonVersion, err = prompt(reader, "Python version", cfg.PythonVersion, "3.13")
		if err != nil {
			return nil, err
		}
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

// ToExtraVars converts the config to a flat map for Ansible --extra-vars.
func (c *Config) ToExtraVars() map[string]string {
	vars := map[string]string{
		"username":          c.Username,
		"email":             c.Email,
		"git_name":          c.GitName,
		"git_email":         c.GitEmail,
		"git_https":         boolStr(c.GitHTTPS),
		"default_shell":     c.DefaultShell,
		"install_podman":    boolStr(c.InstallPodman),
		"podman_wsl_distro": c.PodmanWSLDistro,
		"podman_wsl_host":   c.PodmanWSLHost,
		"podman_wsl_port":   c.PodmanWSLPort,
		"install_bun":       boolStr(c.InstallBun),
		"install_go":        boolStr(c.InstallGo),
		"go_version":        c.GoVersion,
		"install_dotnet":    boolStr(c.InstallDotnet),
		"dotnet_version":    c.DotnetVersion,
		"install_python":    boolStr(c.InstallPython),
		"python_version":    c.PythonVersion,
	}
	if len(c.ExtraPackages) > 0 {
		vars["extra_packages"] = strings.Join(c.ExtraPackages, ",")
	}
	return vars
}

// AvailableRoles returns all role tag names the user can select.
func AvailableRoles() []string {
	return []string{"base", "git-config", "shell", "dev-tools"}
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

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
