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

type alterationState int

const (
	alterationBrowse alterationState = iota
	alterationAdd
	alterationConfirmDeleteAll
)

type alterationsLoadedMsg struct{ alterations []types.Alteration }
type alterationAddedMsg struct{}
type alterationDeletedMsg struct{}
type allAlterationsDeletedMsg struct{}

type AlterationsModel struct {
	client       *api.Client
	state        alterationState
	alterations  []types.Alteration
	cursor       int
	patternInput textinput.Model
	statusInput  textinput.Model
	bodyInput    textinput.Model
	focusedField int
	loading      bool
	spinner      spinner.Model
	err          error
	width        int
	height       int
	styles       Styles
	keys         KeyMap
}

func NewAlterationsModel(client *api.Client, styles Styles, keys KeyMap) AlterationsModel {
	pi := textinput.New()
	pi.Placeholder = "URL pattern regex"
	pi.CharLimit = 256
	pi.SetWidth(60)

	si := textinput.New()
	si.Placeholder = "status code (e.g. 503)"
	si.CharLimit = 3
	si.SetWidth(40)

	bi := textinput.New()
	bi.Placeholder = "response body"
	bi.CharLimit = 1024
	bi.SetWidth(60)

	s := spinner.New()
	s.Spinner = spinner.Dot

	return AlterationsModel{
		client:       client,
		state:        alterationBrowse,
		patternInput: pi,
		statusInput:  si,
		bodyInput:    bi,
		spinner:      s,
		styles:       styles,
		keys:         keys,
	}
}

func (m AlterationsModel) Init() tea.Cmd {
	return tea.Batch(m.loadAlterations(), m.spinner.Tick)
}

func (m AlterationsModel) Update(msg tea.Msg) (AlterationsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case alterationsLoadedMsg:
		m.loading = false
		m.alterations = msg.alterations
		if m.alterations == nil {
			m.alterations = []types.Alteration{}
		}
		m.err = nil
		if m.cursor >= len(m.alterations) && m.cursor > 0 {
			m.cursor = len(m.alterations) - 1
		}
		return m, nil

	case alterationAddedMsg:
		m.state = alterationBrowse
		m.loading = true
		return m, m.loadAlterations()

	case alterationDeletedMsg:
		m.loading = true
		return m, m.loadAlterations()

	case allAlterationsDeletedMsg:
		m.state = alterationBrowse
		m.loading = true
		return m, m.loadAlterations()

	case apiErrMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		if m.err != nil && m.state == alterationBrowse {
			m.err = nil
		}
		return m.handleKey(msg)
	}

	return m.updateInputs(msg)
}

func (m AlterationsModel) handleKey(msg tea.KeyPressMsg) (AlterationsModel, tea.Cmd) {
	switch m.state {
	case alterationBrowse:
		return m.handleBrowseKey(msg)
	case alterationAdd:
		return m.handleAddKey(msg)
	case alterationConfirmDeleteAll:
		return m.handleConfirmDeleteAllKey(msg)
	}
	return m, nil
}

func (m AlterationsModel) handleBrowseKey(msg tea.KeyPressMsg) (AlterationsModel, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Add):
		m.state = alterationAdd
		m.patternInput.SetValue("")
		m.statusInput.SetValue("")
		m.bodyInput.SetValue("")
		m.focusedField = 0
		cmd := m.patternInput.Focus()
		m.statusInput.Blur()
		m.bodyInput.Blur()
		return m, cmd

	case key.Matches(msg, m.keys.Delete):
		if len(m.alterations) > 0 && m.cursor < len(m.alterations) {
			m.loading = true
			idx := m.cursor
			return m, m.deleteAlteration(idx)
		}

	case key.Matches(msg, m.keys.DeleteAll):
		if len(m.alterations) > 0 {
			m.state = alterationConfirmDeleteAll
		}

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.alterations)-1 {
			m.cursor++
		}

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		return m, m.loadAlterations()
	}
	return m, nil
}

func (m AlterationsModel) handleAddKey(msg tea.KeyPressMsg) (AlterationsModel, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.state = alterationBrowse
		m.patternInput.Blur()
		m.statusInput.Blur()
		m.bodyInput.Blur()
		return m, nil

	case key.Matches(msg, m.keys.NextField):
		m.focusedField = (m.focusedField + 1) % 3
		m.patternInput.Blur()
		m.statusInput.Blur()
		m.bodyInput.Blur()
		var cmd tea.Cmd
		switch m.focusedField {
		case 0:
			cmd = m.patternInput.Focus()
		case 1:
			cmd = m.statusInput.Focus()
		case 2:
			cmd = m.bodyInput.Focus()
		}
		return m, cmd

	case key.Matches(msg, m.keys.Confirm):
		pattern := strings.TrimSpace(m.patternInput.Value())
		statusStr := strings.TrimSpace(m.statusInput.Value())
		body := m.bodyInput.Value()

		if pattern == "" {
			m.err = fmt.Errorf("URL pattern cannot be empty")
			return m, nil
		}
		statusCode, err := strconv.Atoi(statusStr)
		if err != nil || statusCode < 100 || statusCode > 599 {
			m.err = fmt.Errorf("invalid status code: must be 100-599")
			return m, nil
		}

		m.patternInput.Blur()
		m.statusInput.Blur()
		m.bodyInput.Blur()
		m.loading = true
		return m, m.addAlteration(types.Alteration{
			URLPattern: pattern,
			StatusCode: statusCode,
			Body:       body,
		})
	}

	var cmd tea.Cmd
	switch m.focusedField {
	case 0:
		m.patternInput, cmd = m.patternInput.Update(msg)
	case 1:
		m.statusInput, cmd = m.statusInput.Update(msg)
	case 2:
		m.bodyInput, cmd = m.bodyInput.Update(msg)
	}
	return m, cmd
}

func (m AlterationsModel) handleConfirmDeleteAllKey(msg tea.KeyPressMsg) (AlterationsModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.state = alterationBrowse
		m.loading = true
		return m, m.deleteAllAlterations()
	default:
		m.state = alterationBrowse
		return m, nil
	}
}

func (m AlterationsModel) updateInputs(msg tea.Msg) (AlterationsModel, tea.Cmd) {
	var cmd tea.Cmd
	if m.state == alterationAdd {
		switch m.focusedField {
		case 0:
			m.patternInput, cmd = m.patternInput.Update(msg)
		case 1:
			m.statusInput, cmd = m.statusInput.Update(msg)
		case 2:
			m.bodyInput, cmd = m.bodyInput.Update(msg)
		}
	}
	return m, cmd
}

func (m AlterationsModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render("Alterations"))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString(m.spinner.View() + " Loading...")
		return b.String()
	}

	// Confirm delete all
	if m.state == alterationConfirmDeleteAll {
		b.WriteString(m.styles.ErrorText.Render("Delete all alterations? (y/n)"))
		b.WriteString("\n\n")
	}

	// Add form
	if m.state == alterationAdd {
		b.WriteString(m.styles.InputLabel.Render("Add Alteration"))
		b.WriteString("\n")

		labels := []string{"  URL Pattern: ", "  Status Code: ", "  Body: "}
		inputs := []string{m.patternInput.View(), m.statusInput.View(), m.bodyInput.View()}
		for i := range 3 {
			label := labels[i]
			if i == m.focusedField {
				label = m.styles.InputLabel.Render("▸" + labels[i][1:])
			}
			b.WriteString(label)
			b.WriteString(inputs[i])
			b.WriteString("\n")
		}
		b.WriteString(m.styles.HelpStyle.Render("  tab: next field • enter: submit • esc: cancel"))
		b.WriteString("\n\n")
	}

	// Alteration list
	b.WriteString(m.styles.InputLabel.Render("Response Alterations"))
	b.WriteString("\n")
	if len(m.alterations) == 0 {
		b.WriteString(m.styles.HelpStyle.Render("  No alterations configured"))
		b.WriteString("\n")
	} else {
		for i, a := range m.alterations {
			bodyPreview := m.truncate(a.Body, 30)
			if bodyPreview == "" {
				bodyPreview = "(empty)"
			}
			line := fmt.Sprintf("  %s → %d %s", m.truncate(a.URLPattern, 40), a.StatusCode, bodyPreview)
			if i == m.cursor && m.state == alterationBrowse {
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

func (m AlterationsModel) truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func (m AlterationsModel) HelpKeys() []key.Binding {
	switch m.state {
	case alterationAdd:
		return []key.Binding{m.keys.NextField, m.keys.Confirm, m.keys.Cancel}
	case alterationConfirmDeleteAll:
		return nil
	default:
		bindings := []key.Binding{m.keys.Add, m.keys.Refresh}
		if len(m.alterations) > 0 {
			bindings = append(bindings, m.keys.Delete, m.keys.DeleteAll, m.keys.Up, m.keys.Down)
		}
		return bindings
	}
}

// Tea commands

func (m AlterationsModel) loadAlterations() tea.Cmd {
	return func() tea.Msg {
		alts, err := m.client.ListAlterations()
		if err != nil {
			return apiErrMsg{err}
		}
		return alterationsLoadedMsg{alts}
	}
}

func (m AlterationsModel) addAlteration(a types.Alteration) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.AddAlteration(a); err != nil {
			return apiErrMsg{err}
		}
		return alterationAddedMsg{}
	}
}

func (m AlterationsModel) deleteAlteration(index int) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.RemoveAlteration(index); err != nil {
			return apiErrMsg{err}
		}
		return alterationDeletedMsg{}
	}
}

func (m AlterationsModel) deleteAllAlterations() tea.Cmd {
	return func() tea.Msg {
		if err := m.client.RemoveAllAlterations(); err != nil {
			return apiErrMsg{err}
		}
		return allAlterationsDeletedMsg{}
	}
}
