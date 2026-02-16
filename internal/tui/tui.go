package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jaydubyaeey/flux/internal/ansible"
	"github.com/jaydubyaeey/flux/internal/config"
	"github.com/jaydubyaeey/flux/internal/updater"
)

// --- screens ---

type screen int

const (
	screenMain screen = iota
	screenRoles
	screenConfigMenu
	screenConfigShow
	screenConfigEdit
	screenRunning
	screenDone
)

// --- menu items ---

type menuItem struct {
	label string
	desc  string
}

var mainMenu = []menuItem{
	{"Run Setup", "Apply configuration to this machine"},
	{"Dry Run", "Preview changes without applying (--check)"},
	{"Configure", "View or edit your settings"},
	{"Update", "Pull latest changes and rebuild flux"},
	{"Quit", "Exit flux"},
}

var configMenu = []menuItem{
	{"Show Config", "Display current configuration"},
	{"Edit Config", "Modify settings interactively"},
	{"Config Path", "Show config file location"},
	{"Back", "Return to main menu"},
}

// --- model ---

type model struct {
	screen   screen
	cursor   int
	dryRun   bool
	err      error
	message  string
	quitting bool

	// Role selection
	roles    []string
	selected map[int]bool

	// Config
	cfg          *config.Config
	configOutput string

	// Config edit state
	editFields []editField
	editCursor int
	editInput  string
	editDone   bool
}

type editField struct {
	key   string
	label string
	value string
}

func initialModel() model {
	roles := config.AvailableRoles()
	sel := make(map[int]bool)
	for i := range roles {
		sel[i] = true // all selected by default
	}

	cfg, _ := config.Load()

	return model{
		screen:   screenMain,
		roles:    roles,
		selected: sel,
		cfg:      cfg,
	}
}

// --- messages ---

type playbookDoneMsg struct{ err error }
type updateDoneMsg struct{ err error }

// --- bubbletea interface ---

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case playbookDoneMsg:
		m.screen = screenDone
		m.err = msg.err
		if msg.err != nil {
			m.message = fmt.Sprintf("Playbook failed: %v", msg.err)
		} else {
			mode := "applied"
			if m.dryRun {
				mode = "checked (dry run)"
			}
			m.message = fmt.Sprintf("Setup %s successfully!", mode)
		}
		return m, nil
	case updateDoneMsg:
		m.screen = screenDone
		m.err = msg.err
		if msg.err != nil {
			m.message = fmt.Sprintf("Update failed: %v", msg.err)
		} else {
			m.message = "flux updated successfully!"
		}
		return m, nil
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global keys
	switch key {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}

	switch m.screen {
	case screenMain:
		return m.handleMainMenu(key)
	case screenRoles:
		return m.handleRoleSelect(key)
	case screenConfigMenu:
		return m.handleConfigMenu(key)
	case screenConfigShow, screenDone:
		return m.handleAnyKeyBack(key)
	case screenConfigEdit:
		return m.handleConfigEdit(key)
	}

	return m, nil
}

func (m model) handleMainMenu(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(mainMenu)-1 {
			m.cursor++
		}
	case "enter":
		switch m.cursor {
		case 0: // Run
			m.dryRun = false
			m.screen = screenRoles
			m.cursor = 0
		case 1: // Dry Run
			m.dryRun = true
			m.screen = screenRoles
			m.cursor = 0
		case 2: // Configure
			m.screen = screenConfigMenu
			m.cursor = 0
		case 3: // Update
			m.screen = screenRunning
			m.message = "Updating flux..."
			return m, func() tea.Msg {
				err := updater.Update()
				return updateDoneMsg{err: err}
			}
		case 4: // Quit
			m.quitting = true
			return m, tea.Quit
		}
	case "q":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m model) handleRoleSelect(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.roles)-1 {
			m.cursor++
		}
	case " ":
		m.selected[m.cursor] = !m.selected[m.cursor]
	case "a":
		allSelected := true
		for i := range m.roles {
			if !m.selected[i] {
				allSelected = false
				break
			}
		}
		for i := range m.roles {
			m.selected[i] = !allSelected
		}
	case "enter":
		return m.executePlaybook()
	case "esc":
		m.screen = screenMain
		m.cursor = 0
	}
	return m, nil
}

func (m model) handleConfigMenu(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(configMenu)-1 {
			m.cursor++
		}
	case "enter":
		switch m.cursor {
		case 0: // Show
			m.screen = screenConfigShow
			cfg, err := config.Load()
			if err != nil {
				m.configOutput = fmt.Sprintf("No config found: %v\nRun setup first to create one.", err)
			} else {
				out, _ := cfg.Marshal()
				m.configOutput = string(out)
			}
		case 1: // Edit
			m.screen = screenConfigEdit
			m.editCursor = 0
			m.editDone = false
			m.initEditFields()
		case 2: // Path
			m.screen = screenConfigShow
			m.configOutput = config.FilePath()
		case 3: // Back
			m.screen = screenMain
			m.cursor = 0
		}
	case "esc":
		m.screen = screenMain
		m.cursor = 0
	}
	return m, nil
}

func (m model) handleAnyKeyBack(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "enter", "q":
		m.screen = screenMain
		m.cursor = 0
		m.err = nil
		m.message = ""
	}
	return m, nil
}

func (m model) handleConfigEdit(key string) (tea.Model, tea.Cmd) {
	if m.editDone {
		// Save and go back
		switch key {
		case "enter", "esc":
			m.applyEditFields()
			if err := config.Save(m.cfg); err != nil {
				m.message = fmt.Sprintf("Error saving: %v", err)
			}
			m.screen = screenConfigMenu
			m.cursor = 0
		}
		return m, nil
	}

	switch key {
	case "up", "shift+tab":
		if m.editCursor > 0 {
			m.editCursor--
			m.editInput = m.editFields[m.editCursor].value
		}
	case "down", "tab":
		if m.editCursor < len(m.editFields)-1 {
			m.editCursor++
			m.editInput = m.editFields[m.editCursor].value
		}
	case "enter":
		// Save current field value
		m.editFields[m.editCursor].value = m.editInput
		if m.editCursor < len(m.editFields)-1 {
			m.editCursor++
			m.editInput = m.editFields[m.editCursor].value
		} else {
			m.editDone = true
		}
	case "backspace":
		if len(m.editInput) > 0 {
			m.editInput = m.editInput[:len(m.editInput)-1]
		}
	case "esc":
		m.screen = screenConfigMenu
		m.cursor = 0
	default:
		if len(key) == 1 {
			m.editInput += key
		}
	}
	return m, nil
}

func (m *model) initEditFields() {
	cfg := m.cfg
	if cfg == nil {
		cfg = config.DefaultConfig()
		m.cfg = cfg
	}
	m.editFields = []editField{
		{"username", "Username", cfg.Username},
		{"email", "Email", cfg.Email},
		{"git_name", "Git Name", cfg.GitName},
		{"git_email", "Git Email", cfg.GitEmail},
		{"git_https", "GitHub HTTPS (true/false)", boolStr(cfg.GitHTTPS)},
		{"default_shell", "Shell (bash/zsh)", cfg.DefaultShell},
		{"install_podman", "Install Podman (true/false)", boolStr(cfg.InstallPodman)},
		{"podman_wsl_distro", "Podman WSL Distro", cfg.PodmanWSLDistro},
		{"podman_wsl_host", "Podman WSL Host", cfg.PodmanWSLHost},
		{"podman_wsl_port", "Podman WSL Port", cfg.PodmanWSLPort},
		{"install_bun", "Install Bun (true/false)", boolStr(cfg.InstallBun)},
		{"install_go", "Install Go (true/false)", boolStr(cfg.InstallGo)},
		{"go_version", "Go Version", cfg.GoVersion},
		{"install_dotnet", "Install .NET (true/false)", boolStr(cfg.InstallDotnet)},
		{"dotnet_version", ".NET SDK Version", cfg.DotnetVersion},
		{"install_python", "Install Python (true/false)", boolStr(cfg.InstallPython)},
		{"python_version", "Python Version", cfg.PythonVersion},
		{"extra_packages", "Extra Packages (csv)", strings.Join(cfg.ExtraPackages, ", ")},
	}
	m.editInput = m.editFields[0].value
}

func (m *model) applyEditFields() {
	if m.cfg == nil {
		m.cfg = config.DefaultConfig()
	}
	for _, f := range m.editFields {
		switch f.key {
		case "username":
			m.cfg.Username = f.value
		case "email":
			m.cfg.Email = f.value
		case "git_name":
			m.cfg.GitName = f.value
		case "git_email":
			m.cfg.GitEmail = f.value
		case "git_https":
			m.cfg.GitHTTPS = parseBool(f.value)
		case "default_shell":
			m.cfg.DefaultShell = f.value
		case "install_podman":
			m.cfg.InstallPodman = parseBool(f.value)
		case "podman_wsl_distro":
			m.cfg.PodmanWSLDistro = f.value
		case "podman_wsl_host":
			m.cfg.PodmanWSLHost = f.value
		case "podman_wsl_port":
			m.cfg.PodmanWSLPort = f.value
		case "install_bun":
			m.cfg.InstallBun = parseBool(f.value)
		case "install_go":
			m.cfg.InstallGo = parseBool(f.value)
		case "go_version":
			m.cfg.GoVersion = f.value
		case "install_dotnet":
			m.cfg.InstallDotnet = parseBool(f.value)
		case "dotnet_version":
			m.cfg.DotnetVersion = f.value
		case "install_python":
			m.cfg.InstallPython = parseBool(f.value)
		case "python_version":
			m.cfg.PythonVersion = f.value
		case "extra_packages":
			m.cfg.ExtraPackages = nil
			for _, p := range strings.Split(f.value, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					m.cfg.ExtraPackages = append(m.cfg.ExtraPackages, p)
				}
			}
		}
	}
}

func (m model) executePlaybook() (model, tea.Cmd) {
	// Ensure config exists
	if m.cfg == nil {
		cfg, err := config.LoadOrCreate()
		if err != nil {
			m.screen = screenDone
			m.message = fmt.Sprintf("Config error: %v", err)
			m.err = err
			return m, nil
		}
		m.cfg = cfg
	}

	// Build tags from selection
	var tags []string
	for i, r := range m.roles {
		if m.selected[i] {
			tags = append(tags, r)
		}
	}
	if len(tags) == 0 {
		m.message = "No roles selected"
		return m, nil
	}

	m.screen = screenRunning

	tagStr := strings.Join(tags, ",")
	dryRun := m.dryRun
	cfg := m.cfg

	// Run playbook in a goroutine so TUI can show status
	return m, func() tea.Msg {
		if err := ansible.EnsureInstalled(); err != nil {
			return playbookDoneMsg{err: err}
		}
		ansibleDir, err := ansible.FindAnsibleDir()
		if err != nil {
			return playbookDoneMsg{err: err}
		}
		extraVars := cfg.ToExtraVars()
		err = ansible.RunPlaybook(ansibleDir, extraVars, tagStr, dryRun)
		return playbookDoneMsg{err: err}
	}
}

// --- View ---

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	header := titleStyle.Render("⚡ flux")
	b.WriteString(header + "\n")

	switch m.screen {
	case screenMain:
		b.WriteString(subtitleStyle.Render("WSL bootstrap & configuration") + "\n\n")
		for i, item := range mainMenu {
			cursor := "  "
			style := normalStyle
			if i == m.cursor {
				cursor = "▸ "
				style = selectedStyle
			}
			b.WriteString(cursor + style.Render(item.label))
			b.WriteString("  " + subtitleStyle.Render(item.desc) + "\n")
		}
		b.WriteString(helpStyle.Render("↑/↓ navigate • enter select • q quit"))

	case screenRoles:
		mode := "Run"
		if m.dryRun {
			mode = dryRunBadge.Render("DRY RUN")
		}
		b.WriteString(subtitleStyle.Render("Select roles to "+mode) + "\n\n")

		for i, role := range m.roles {
			cursor := "  "
			if i == m.cursor {
				cursor = "▸ "
			}
			check := uncheckStyle.Render("☐")
			if m.selected[i] {
				check = checkStyle.Render("☑")
			}
			style := normalStyle
			if i == m.cursor {
				style = selectedStyle
			}
			b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, style.Render(role)))
		}
		if m.message != "" {
			b.WriteString("\n" + errorStyle.Render(m.message))
			m.message = ""
		}
		b.WriteString(helpStyle.Render("↑/↓ navigate • space toggle • a all/none • enter run • esc back"))

	case screenConfigMenu:
		b.WriteString(subtitleStyle.Render("Configuration") + "\n\n")
		for i, item := range configMenu {
			cursor := "  "
			style := normalStyle
			if i == m.cursor {
				cursor = "▸ "
				style = selectedStyle
			}
			b.WriteString(cursor + style.Render(item.label))
			b.WriteString("  " + subtitleStyle.Render(item.desc) + "\n")
		}
		b.WriteString(helpStyle.Render("↑/↓ navigate • enter select • esc back"))

	case screenConfigShow:
		b.WriteString(subtitleStyle.Render("Configuration") + "\n\n")
		b.WriteString(m.configOutput + "\n")
		b.WriteString(helpStyle.Render("press enter or esc to go back"))

	case screenConfigEdit:
		b.WriteString(subtitleStyle.Render("Edit Configuration") + "\n\n")
		for i, f := range m.editFields {
			cursor := "  "
			if i == m.editCursor && !m.editDone {
				cursor = "▸ "
			}
			label := configKeyStyle.Render(f.label)
			val := f.value
			if i == m.editCursor && !m.editDone {
				val = m.editInput + "▏"
				val = selectedStyle.Render(val)
			} else {
				val = configValStyle.Render(val)
			}
			b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, label, val))
		}
		if m.editDone {
			b.WriteString("\n" + successStyle.Render("✓ Press enter to save"))
		}
		b.WriteString(helpStyle.Render("↑/↓ navigate • enter confirm field • esc cancel"))

	case screenRunning:
		// The playbook runs with stdin/stdout attached, so show minimal TUI
		mode := "Applying"
		if m.dryRun {
			mode = "Checking (dry run)"
		}
		spinner := lipgloss.NewStyle().Foreground(accentColor).Render("⟳")
		b.WriteString(fmt.Sprintf("\n%s %s configuration...\n", spinner, mode))
		b.WriteString(subtitleStyle.Render("Ansible output appears in terminal below"))

	case screenDone:
		if m.err != nil {
			b.WriteString("\n" + errorStyle.Render("✗ "+m.message) + "\n")
		} else {
			b.WriteString("\n" + successStyle.Render("✓ "+m.message) + "\n")
		}
		b.WriteString(helpStyle.Render("press enter or esc to continue"))
	}

	return b.String() + "\n"
}

// --- helpers ---

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func parseBool(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "true" || s == "yes" || s == "y" || s == "1"
}

// --- Public entry points ---

// Run launches the interactive TUI.
func Run() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// RunPlaybookCLI runs the playbook from CLI flags (non-TUI mode).
func RunPlaybookCLI(cfg *config.Config, tags string, dryRun bool) {
	fmt.Printf("Running setup for user: %s\n", cfg.Username)

	if err := ansible.EnsureInstalled(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to install Ansible: %v\n", err)
		os.Exit(1)
	}

	ansibleDir, err := ansible.FindAnsibleDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot find ansible directory: %v\n", err)
		os.Exit(1)
	}

	extraVars := cfg.ToExtraVars()
	if err := ansible.RunPlaybook(ansibleDir, extraVars, tags, dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "\nPlaybook failed: %v\n", err)
		os.Exit(1)
	}

	if dryRun {
		fmt.Println("\n✓ Dry run complete — no changes were applied")
	} else {
		fmt.Println("\n✓ Setup complete!")
	}
}
