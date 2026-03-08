package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/mitchgrogg/rita-devtools-tui/internal/api"
)

type apiErrMsg struct{ err error }

type App struct {
	client      *api.Client
	activeTab   int
	tabs        []string
	delays      DelaysModel
	alterations AlterationsModel
	settings    SettingsModel
	width       int
	height      int
	styles      Styles
	keys        KeyMap
}

func NewApp(client *api.Client) *App {
	styles := NewStyles()
	keys := NewKeyMap()
	return &App{
		client:      client,
		tabs:        []string{"Delays", "Alterations", "Settings"},
		delays:      NewDelaysModel(client, styles, keys),
		alterations: NewAlterationsModel(client, styles, keys),
		settings:    NewSettingsModel(client, styles, keys),
		styles:      styles,
		keys:        keys,
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.delays.Init(),
		a.alterations.Init(),
		a.settings.Init(),
	)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.delays.width = msg.Width
		a.delays.height = msg.Height - 4
		a.alterations.width = msg.Width
		a.alterations.height = msg.Height - 4
		a.settings.width = msg.Width
		a.settings.height = msg.Height - 4
		return a, nil

	case tea.KeyPressMsg:
		// Global quit
		if key.Matches(msg, a.keys.Quit) {
			if a.isInForm() {
				if msg.String() == "q" {
					return a.updateActiveTab(msg)
				}
				return a, tea.Quit
			}
			return a, tea.Quit
		}

		// Tab switching (only when not in a form)
		if !a.isInForm() {
			switch {
			case key.Matches(msg, a.keys.Tab):
				a.activeTab = (a.activeTab + 1) % len(a.tabs)
				return a, nil
			case key.Matches(msg, a.keys.ShiftTab):
				a.activeTab = (a.activeTab - 1 + len(a.tabs)) % len(a.tabs)
				return a, nil
			case msg.String() == "1":
				a.activeTab = 0
				return a, nil
			case msg.String() == "2":
				a.activeTab = 1
				return a, nil
			case msg.String() == "3":
				a.activeTab = 2
				return a, nil
			}
		}

		return a.updateActiveTab(msg)
	}

	// Route non-key messages to both sub-models so spinners/async work
	return a.updateAll(msg)
}

func (a *App) isInForm() bool {
	switch a.activeTab {
	case 0:
		return a.delays.state != delayBrowse
	case 1:
		return a.alterations.state != alterationBrowse
	case 2:
		return a.settings.state != settingsBrowse
	}
	return false
}

func (a *App) updateActiveTab(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch a.activeTab {
	case 0:
		a.delays, cmd = a.delays.Update(msg)
	case 1:
		a.alterations, cmd = a.alterations.Update(msg)
	case 2:
		a.settings, cmd = a.settings.Update(msg)
	}
	return a, cmd
}

func (a *App) updateAll(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	a.delays, cmd = a.delays.Update(msg)
	cmds = append(cmds, cmd)
	a.alterations, cmd = a.alterations.Update(msg)
	cmds = append(cmds, cmd)
	a.settings, cmd = a.settings.Update(msg)
	cmds = append(cmds, cmd)
	return a, tea.Batch(cmds...)
}

func (a *App) View() tea.View {
	var b strings.Builder

	// Tab bar
	var tabs []string
	for i, t := range a.tabs {
		if i == a.activeTab {
			tabs = append(tabs, a.styles.ActiveTab.Render(t))
		} else {
			tabs = append(tabs, a.styles.InactiveTab.Render(t))
		}
	}
	b.WriteString(a.styles.TabBar.Render(strings.Join(tabs, " ")))
	b.WriteString("\n")

	// Active view
	switch a.activeTab {
	case 0:
		b.WriteString(a.delays.View())
	case 1:
		b.WriteString(a.alterations.View())
	case 2:
		b.WriteString(a.settings.View())
	}

	// Help bar
	b.WriteString("\n\n")
	var helpKeys []key.Binding
	helpKeys = append(helpKeys, a.keys.Tab)
	switch a.activeTab {
	case 0:
		helpKeys = append(helpKeys, a.delays.HelpKeys()...)
	case 1:
		helpKeys = append(helpKeys, a.alterations.HelpKeys()...)
	case 2:
		helpKeys = append(helpKeys, a.settings.HelpKeys()...)
	}
	helpKeys = append(helpKeys, a.keys.Quit)

	var helpParts []string
	for _, k := range helpKeys {
		help := k.Help()
		helpParts = append(helpParts, help.Key+": "+help.Desc)
	}
	b.WriteString(a.styles.HelpStyle.Render(strings.Join(helpParts, " • ")))

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}
