package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/mitchgrogg/rita-devtools-tui/internal/api"
	"github.com/mitchgrogg/rita-devtools-tui/internal/types"
)

type settingsState int

const (
	settingsBrowse settingsState = iota
	settingsExport
	settingsImport
	settingsConfirmImport
)

type configExportedMsg struct{ path string }
type configImportedMsg struct{}
type configLoadedForImportMsg struct{ path string }

type SettingsModel struct {
	client    *api.Client
	state     settingsState
	pathInput textinput.Model
	cursor    int
	loading   bool
	spinner   spinner.Model
	err       error
	success   string
	importPath string
	width     int
	height    int
	styles    Styles
	keys      KeyMap
}

func NewSettingsModel(client *api.Client, styles Styles, keys KeyMap) SettingsModel {
	pi := textinput.New()
	pi.Placeholder = "/path/to/config.json"
	pi.CharLimit = 512
	pi.SetWidth(60)

	s := spinner.New()
	s.Spinner = spinner.Dot

	return SettingsModel{
		client:    client,
		state:     settingsBrowse,
		pathInput: pi,
		spinner:   s,
		styles:    styles,
		keys:      keys,
	}
}

func (m SettingsModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case configExportedMsg:
		m.loading = false
		m.state = settingsBrowse
		m.err = nil
		m.success = fmt.Sprintf("Config exported to %s", msg.path)
		return m, nil

	case configImportedMsg:
		m.loading = false
		m.state = settingsBrowse
		m.err = nil
		m.success = "Config imported successfully"
		return m, nil

	case configLoadedForImportMsg:
		m.loading = false
		m.importPath = msg.path
		m.state = settingsConfirmImport
		return m, nil

	case apiErrMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		if m.err != nil && m.state == settingsBrowse {
			m.err = nil
		}
		if m.success != "" && m.state == settingsBrowse {
			m.success = ""
		}
		return m.handleKey(msg)
	}

	return m.updateInputs(msg)
}

func (m SettingsModel) handleKey(msg tea.KeyPressMsg) (SettingsModel, tea.Cmd) {
	switch m.state {
	case settingsBrowse:
		return m.handleBrowseKey(msg)
	case settingsExport:
		return m.handleExportKey(msg)
	case settingsImport:
		return m.handleImportKey(msg)
	case settingsConfirmImport:
		return m.handleConfirmImportKey(msg)
	}
	return m, nil
}

func (m SettingsModel) handleBrowseKey(msg tea.KeyPressMsg) (SettingsModel, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.cursor < 1 {
			m.cursor++
		}
	case key.Matches(msg, m.keys.Confirm):
		m.pathInput.SetValue("")
		cmd := m.pathInput.Focus()
		if m.cursor == 0 {
			m.state = settingsExport
		} else {
			m.state = settingsImport
		}
		return m, cmd
	}
	return m, nil
}

func (m SettingsModel) handleExportKey(msg tea.KeyPressMsg) (SettingsModel, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.state = settingsBrowse
		m.pathInput.Blur()
		return m, nil

	case key.Matches(msg, m.keys.Confirm):
		path := strings.TrimSpace(m.pathInput.Value())
		if path == "" {
			m.err = fmt.Errorf("file path cannot be empty")
			return m, nil
		}
		m.pathInput.Blur()
		m.loading = true
		return m, m.exportConfig(path)
	}

	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}

func (m SettingsModel) handleImportKey(msg tea.KeyPressMsg) (SettingsModel, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.state = settingsBrowse
		m.pathInput.Blur()
		return m, nil

	case key.Matches(msg, m.keys.Confirm):
		path := strings.TrimSpace(m.pathInput.Value())
		if path == "" {
			m.err = fmt.Errorf("file path cannot be empty")
			return m, nil
		}
		m.pathInput.Blur()
		// Validate the file exists before confirming
		if _, err := os.Stat(path); err != nil {
			m.err = fmt.Errorf("cannot read file: %w", err)
			return m, nil
		}
		m.importPath = path
		m.state = settingsConfirmImport
		return m, nil
	}

	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}

func (m SettingsModel) handleConfirmImportKey(msg tea.KeyPressMsg) (SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.state = settingsBrowse
		m.loading = true
		return m, m.importConfig(m.importPath)
	default:
		m.state = settingsBrowse
		m.importPath = ""
		return m, nil
	}
}

func (m SettingsModel) updateInputs(msg tea.Msg) (SettingsModel, tea.Cmd) {
	var cmd tea.Cmd
	if m.state == settingsExport || m.state == settingsImport {
		m.pathInput, cmd = m.pathInput.Update(msg)
	}
	return m, cmd
}

var menuItems = []string{"Export config to file", "Import config from file"}

func (m SettingsModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render("Settings"))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString(m.spinner.View() + " Loading...")
		return b.String()
	}

	// Confirm import
	if m.state == settingsConfirmImport {
		b.WriteString(m.styles.ErrorText.Render(
			fmt.Sprintf("Import config from %s? This will replace all current settings. (y/n)", m.importPath)))
		b.WriteString("\n\n")
		return b.String()
	}

	// Export/Import form
	if m.state == settingsExport || m.state == settingsImport {
		label := "Export to"
		if m.state == settingsImport {
			label = "Import from"
		}
		b.WriteString(m.styles.InputLabel.Render(label))
		b.WriteString("\n")
		b.WriteString(m.styles.InputLabel.Render("▸ File path: "))
		b.WriteString(m.pathInput.View())
		b.WriteString("\n")
		b.WriteString(m.styles.HelpStyle.Render("  enter: confirm • esc: cancel"))
		b.WriteString("\n\n")
		if m.err != nil {
			b.WriteString(m.styles.ErrorText.Render("Error: " + m.err.Error()))
		}
		return b.String()
	}

	// Browse menu
	for i, item := range menuItems {
		if i == m.cursor {
			b.WriteString(m.styles.SelectedRow.Render("▸  " + item))
		} else {
			b.WriteString("   " + item)
		}
		b.WriteString("\n")
	}

	if m.success != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.SuccessText.Render(m.success))
	}
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(m.styles.ErrorText.Render("Error: " + m.err.Error()))
	}

	return b.String()
}

func (m SettingsModel) HelpKeys() []key.Binding {
	switch m.state {
	case settingsExport, settingsImport:
		return []key.Binding{m.keys.Confirm, m.keys.Cancel}
	case settingsConfirmImport:
		return nil
	default:
		return []key.Binding{m.keys.Confirm, m.keys.Up, m.keys.Down}
	}
}

// Tea commands

func (m SettingsModel) exportConfig(path string) tea.Cmd {
	return func() tea.Msg {
		cfg, err := m.client.GetConfig()
		if err != nil {
			return apiErrMsg{err}
		}
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return apiErrMsg{fmt.Errorf("marshal config: %w", err)}
		}
		if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
			return apiErrMsg{fmt.Errorf("write file: %w", err)}
		}
		return configExportedMsg{path}
	}
}

func (m SettingsModel) importConfig(path string) tea.Cmd {
	return func() tea.Msg {
		data, err := os.ReadFile(path)
		if err != nil {
			return apiErrMsg{fmt.Errorf("read file: %w", err)}
		}
		var cfg types.Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return apiErrMsg{fmt.Errorf("invalid config JSON: %w", err)}
		}
		if err := m.client.PutConfig(cfg); err != nil {
			return apiErrMsg{err}
		}
		return configImportedMsg{}
	}
}
