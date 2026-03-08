package tui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/mitchgrogg/rita-devtools-tui/internal/api"
	"github.com/mitchgrogg/rita-devtools-tui/internal/types"
)

type delayState int

const (
	delayBrowse delayState = iota
	delayEditGlobal
	delayAddPattern
	delayConfirmDeleteAll
)

type delaysLoadedMsg struct{ delays types.Delays }
type globalDelaySetMsg struct{}
type globalDelayRemovedMsg struct{}
type patternAddedMsg struct{}
type patternDeletedMsg struct{}
type allPatternsDeletedMsg struct{}

type DelaysModel struct {
	client       *api.Client
	state        delayState
	globalDelay  *int
	patterns     []types.PatternDelay
	cursor       int
	globalInput  textinput.Model
	patternInput textinput.Model
	delayInput   textinput.Model
	focusedField int
	loading      bool
	spinner      spinner.Model
	err          error
	width        int
	height       int
	styles       Styles
	keys         KeyMap
}

func NewDelaysModel(client *api.Client, styles Styles, keys KeyMap) DelaysModel {
	gi := textinput.New()
	gi.Placeholder = "delay in ms (e.g. 500)"
	gi.CharLimit = 10
	gi.SetWidth(40)

	pi := textinput.New()
	pi.Placeholder = "URL pattern regex"
	pi.CharLimit = 256
	pi.SetWidth(60)

	di := textinput.New()
	di.Placeholder = "delay in ms"
	di.CharLimit = 10
	di.SetWidth(40)

	s := spinner.New()
	s.Spinner = spinner.Dot

	return DelaysModel{
		client:       client,
		state:        delayBrowse,
		globalInput:  gi,
		patternInput: pi,
		delayInput:   di,
		spinner:      s,
		styles:       styles,
		keys:         keys,
	}
}

func (m DelaysModel) Init() tea.Cmd {
	return tea.Batch(m.loadDelays(), m.spinner.Tick)
}

func (m DelaysModel) Update(msg tea.Msg) (DelaysModel, tea.Cmd) {
	switch msg := msg.(type) {
	case delaysLoadedMsg:
		m.loading = false
		m.globalDelay = msg.delays.GlobalDelayMs
		m.patterns = msg.delays.Patterns
		if m.patterns == nil {
			m.patterns = []types.PatternDelay{}
		}
		m.err = nil
		if m.cursor >= len(m.patterns) && m.cursor > 0 {
			m.cursor = len(m.patterns) - 1
		}
		return m, nil

	case globalDelaySetMsg, globalDelayRemovedMsg:
		m.state = delayBrowse
		m.loading = true
		return m, m.loadDelays()

	case patternAddedMsg:
		m.state = delayBrowse
		m.loading = true
		return m, m.loadDelays()

	case patternDeletedMsg:
		m.loading = true
		return m, m.loadDelays()

	case allPatternsDeletedMsg:
		m.state = delayBrowse
		m.loading = true
		return m, m.loadDelays()

	case apiErrMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		if m.err != nil && m.state == delayBrowse {
			m.err = nil
		}
		return m.handleKey(msg)
	}

	return m.updateInputs(msg)
}

func (m DelaysModel) handleKey(msg tea.KeyPressMsg) (DelaysModel, tea.Cmd) {
	switch m.state {
	case delayBrowse:
		return m.handleBrowseKey(msg)
	case delayEditGlobal:
		return m.handleEditGlobalKey(msg)
	case delayAddPattern:
		return m.handleAddPatternKey(msg)
	case delayConfirmDeleteAll:
		return m.handleConfirmDeleteAllKey(msg)
	}
	return m, nil
}

func (m DelaysModel) handleBrowseKey(msg tea.KeyPressMsg) (DelaysModel, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.EditGlobal):
		m.state = delayEditGlobal
		if m.globalDelay != nil {
			m.globalInput.SetValue(strconv.Itoa(*m.globalDelay))
		} else {
			m.globalInput.SetValue("")
		}
		cmd := m.globalInput.Focus()
		return m, cmd

	case key.Matches(msg, m.keys.Add):
		m.state = delayAddPattern
		m.patternInput.SetValue("")
		m.delayInput.SetValue("")
		m.focusedField = 0
		cmd := m.patternInput.Focus()
		m.delayInput.Blur()
		return m, cmd

	case key.Matches(msg, m.keys.Delete):
		if len(m.patterns) > 0 && m.cursor < len(m.patterns) {
			m.loading = true
			idx := m.cursor
			return m, m.deletePattern(idx)
		}

	case key.Matches(msg, m.keys.DeleteAll):
		if len(m.patterns) > 0 {
			m.state = delayConfirmDeleteAll
		}

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.patterns)-1 {
			m.cursor++
		}

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		return m, m.loadDelays()
	}
	return m, nil
}

func (m DelaysModel) handleEditGlobalKey(msg tea.KeyPressMsg) (DelaysModel, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.state = delayBrowse
		m.globalInput.Blur()
		return m, nil

	case key.Matches(msg, m.keys.Confirm):
		val := strings.TrimSpace(m.globalInput.Value())
		m.globalInput.Blur()
		if val == "" {
			m.loading = true
			return m, m.removeGlobalDelay()
		}
		ms, err := strconv.Atoi(val)
		if err != nil || ms < 0 {
			m.err = fmt.Errorf("invalid delay: must be a non-negative integer")
			return m, nil
		}
		m.loading = true
		return m, m.setGlobalDelay(ms)
	}

	var cmd tea.Cmd
	m.globalInput, cmd = m.globalInput.Update(msg)
	return m, cmd
}

func (m DelaysModel) handleAddPatternKey(msg tea.KeyPressMsg) (DelaysModel, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.state = delayBrowse
		m.patternInput.Blur()
		m.delayInput.Blur()
		return m, nil

	case key.Matches(msg, m.keys.NextField):
		if m.focusedField == 0 {
			m.focusedField = 1
			m.patternInput.Blur()
			cmd := m.delayInput.Focus()
			return m, cmd
		}
		m.focusedField = 0
		m.delayInput.Blur()
		cmd := m.patternInput.Focus()
		return m, cmd

	case key.Matches(msg, m.keys.Confirm):
		pattern := strings.TrimSpace(m.patternInput.Value())
		delayStr := strings.TrimSpace(m.delayInput.Value())
		if pattern == "" {
			m.err = fmt.Errorf("pattern cannot be empty")
			return m, nil
		}
		ms, err := strconv.Atoi(delayStr)
		if err != nil || ms < 0 {
			m.err = fmt.Errorf("invalid delay: must be a non-negative integer")
			return m, nil
		}
		m.patternInput.Blur()
		m.delayInput.Blur()
		m.loading = true
		return m, m.addPattern(types.PatternDelay{Pattern: pattern, DelayMs: ms})
	}

	var cmd tea.Cmd
	if m.focusedField == 0 {
		m.patternInput, cmd = m.patternInput.Update(msg)
	} else {
		m.delayInput, cmd = m.delayInput.Update(msg)
	}
	return m, cmd
}

func (m DelaysModel) handleConfirmDeleteAllKey(msg tea.KeyPressMsg) (DelaysModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.state = delayBrowse
		m.loading = true
		return m, m.deleteAllPatterns()
	default:
		m.state = delayBrowse
		return m, nil
	}
}

func (m DelaysModel) updateInputs(msg tea.Msg) (DelaysModel, tea.Cmd) {
	var cmd tea.Cmd
	switch m.state {
	case delayEditGlobal:
		m.globalInput, cmd = m.globalInput.Update(msg)
	case delayAddPattern:
		if m.focusedField == 0 {
			m.patternInput, cmd = m.patternInput.Update(msg)
		} else {
			m.delayInput, cmd = m.delayInput.Update(msg)
		}
	}
	return m, cmd
}

func (m DelaysModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render("Delays"))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString(m.spinner.View() + " Loading...")
		return b.String()
	}

	// Global delay
	if m.state == delayEditGlobal {
		b.WriteString(m.styles.InputLabel.Render("Global Delay (ms): "))
		b.WriteString(m.globalInput.View())
		b.WriteString("\n")
		b.WriteString(m.styles.HelpStyle.Render("  enter: confirm • esc: cancel • empty: remove"))
		b.WriteString("\n\n")
	} else {
		b.WriteString("Global Delay: ")
		if m.globalDelay != nil {
			b.WriteString(fmt.Sprintf("%dms", *m.globalDelay))
		} else {
			b.WriteString(m.styles.HelpStyle.Render("not set"))
		}
		b.WriteString("\n\n")
	}

	// Confirm delete all
	if m.state == delayConfirmDeleteAll {
		b.WriteString(m.styles.ErrorText.Render("Delete all pattern delays? (y/n)"))
		b.WriteString("\n\n")
	}

	// Add pattern form
	if m.state == delayAddPattern {
		b.WriteString(m.styles.InputLabel.Render("Add Pattern Delay"))
		b.WriteString("\n")
		patternLabel := "  Pattern: "
		delayLabel := "  Delay (ms): "
		if m.focusedField == 0 {
			patternLabel = m.styles.InputLabel.Render("▸ Pattern: ")
		} else {
			delayLabel = m.styles.InputLabel.Render("▸ Delay (ms): ")
		}
		b.WriteString(patternLabel)
		b.WriteString(m.patternInput.View())
		b.WriteString("\n")
		b.WriteString(delayLabel)
		b.WriteString(m.delayInput.View())
		b.WriteString("\n")
		b.WriteString(m.styles.HelpStyle.Render("  tab: next field • enter: submit • esc: cancel"))
		b.WriteString("\n\n")
	}

	// Pattern list
	b.WriteString(m.styles.InputLabel.Render("Pattern Delays"))
	b.WriteString("\n")
	if len(m.patterns) == 0 {
		b.WriteString(m.styles.HelpStyle.Render("  No pattern delays configured"))
		b.WriteString("\n")
	} else {
		for i, p := range m.patterns {
			line := fmt.Sprintf("  %s → %dms", m.truncate(p.Pattern, 50), p.DelayMs)
			if i == m.cursor && m.state == delayBrowse {
				b.WriteString(m.styles.SelectedRow.Render("▸ " + line))
			} else {
				b.WriteString("  " + line)
			}
			b.WriteString("\n")
		}
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(m.styles.ErrorText.Render("Error: " + m.err.Error()))
	}

	return b.String()
}

func (m DelaysModel) truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func (m DelaysModel) HelpKeys() []key.Binding {
	switch m.state {
	case delayEditGlobal:
		return []key.Binding{m.keys.Confirm, m.keys.Cancel}
	case delayAddPattern:
		return []key.Binding{m.keys.NextField, m.keys.Confirm, m.keys.Cancel}
	case delayConfirmDeleteAll:
		return nil
	default:
		bindings := []key.Binding{m.keys.EditGlobal, m.keys.Add, m.keys.Refresh}
		if len(m.patterns) > 0 {
			bindings = append(bindings, m.keys.Delete, m.keys.DeleteAll, m.keys.Up, m.keys.Down)
		}
		return bindings
	}
}

// Tea commands

func (m DelaysModel) loadDelays() tea.Cmd {
	return func() tea.Msg {
		delays, err := m.client.GetDelays()
		if err != nil {
			return apiErrMsg{err}
		}
		return delaysLoadedMsg{*delays}
	}
}

func (m DelaysModel) setGlobalDelay(ms int) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.SetGlobalDelay(ms); err != nil {
			return apiErrMsg{err}
		}
		return globalDelaySetMsg{}
	}
}

func (m DelaysModel) removeGlobalDelay() tea.Cmd {
	return func() tea.Msg {
		if err := m.client.RemoveGlobalDelay(); err != nil {
			return apiErrMsg{err}
		}
		return globalDelayRemovedMsg{}
	}
}

func (m DelaysModel) addPattern(p types.PatternDelay) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.AddPatternDelay(p); err != nil {
			return apiErrMsg{err}
		}
		return patternAddedMsg{}
	}
}

func (m DelaysModel) deletePattern(index int) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.RemovePatternDelay(index); err != nil {
			return apiErrMsg{err}
		}
		return patternDeletedMsg{}
	}
}

func (m DelaysModel) deleteAllPatterns() tea.Cmd {
	return func() tea.Msg {
		if err := m.client.RemoveAllPatternDelays(); err != nil {
			return apiErrMsg{err}
		}
		return allPatternsDeletedMsg{}
	}
}
