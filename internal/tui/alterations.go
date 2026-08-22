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

// Every alteration message carries the kind it belongs to: two instances of
// this model are alive at once and App fans non-key messages out to both.
type alterationsLoadedMsg struct {
	kind        types.AlterationKind
	alterations []types.Alteration
}
type alterationAddedMsg struct{ kind types.AlterationKind }
type alterationDeletedMsg struct{ kind types.AlterationKind }
type allAlterationsDeletedMsg struct{ kind types.AlterationKind }

func alterationMsgKind(msg tea.Msg) (types.AlterationKind, bool) {
	switch msg := msg.(type) {
	case alterationsLoadedMsg:
		return msg.kind, true
	case alterationAddedMsg:
		return msg.kind, true
	case alterationDeletedMsg:
		return msg.kind, true
	case allAlterationsDeletedMsg:
		return msg.kind, true
	}
	return "", false
}

type AlterationsModel struct {
	client       *api.Client
	kind         types.AlterationKind
	state        alterationState
	alterations  []types.Alteration
	cursor       int
	patternInput textinput.Model
	middleInput  textinput.Model // status code (response) or rewrite URL (request)
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

func NewAlterationsModel(client *api.Client, styles Styles, keys KeyMap, kind types.AlterationKind) AlterationsModel {
	pi := textinput.New()
	pi.Placeholder = "URL pattern regex"
	pi.CharLimit = 256
	pi.SetWidth(60)

	mi := textinput.New()
	if kind == types.AlterationRequest {
		mi.Placeholder = "rewrite URL (regex substitution)"
		mi.CharLimit = 256
		mi.SetWidth(60)
	} else {
		mi.Placeholder = "status code (e.g. 503)"
		mi.CharLimit = 3
		mi.SetWidth(40)
	}

	bi := textinput.New()
	bi.Placeholder = string(kind) + " body"
	bi.CharLimit = 1024
	bi.SetWidth(60)

	s := spinner.New()
	s.Spinner = spinner.Dot

	return AlterationsModel{
		client:       client,
		kind:         kind,
		state:        alterationBrowse,
		patternInput: pi,
		middleInput:  mi,
		bodyInput:    bi,
		spinner:      s,
		styles:       styles,
		keys:         keys,
	}
}

// title is the display name for this model's kind, e.g. "Request Alterations".
func (m AlterationsModel) title() string {
	if m.kind == types.AlterationRequest {
		return "Request Alterations"
	}
	return "Response Alterations"
}

func (m AlterationsModel) Init() tea.Cmd {
	return tea.Batch(m.loadAlterations(), m.spinner.Tick)
}

func (m AlterationsModel) Update(msg tea.Msg) (AlterationsModel, tea.Cmd) {
	if kind, ok := alterationMsgKind(msg); ok && kind != m.kind {
		return m, nil
	}

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
		m.middleInput.SetValue("")
		m.bodyInput.SetValue("")
		m.focusedField = 0
		cmd := m.patternInput.Focus()
		m.middleInput.Blur()
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
		m.middleInput.Blur()
		m.bodyInput.Blur()
		return m, nil

	case key.Matches(msg, m.keys.NextField):
		m.focusedField = (m.focusedField + 1) % 3
		m.patternInput.Blur()
		m.middleInput.Blur()
		m.bodyInput.Blur()
		var cmd tea.Cmd
		switch m.focusedField {
		case 0:
			cmd = m.patternInput.Focus()
		case 1:
			cmd = m.middleInput.Focus()
		case 2:
			cmd = m.bodyInput.Focus()
		}
		return m, cmd

	case key.Matches(msg, m.keys.Confirm):
		pattern := strings.TrimSpace(m.patternInput.Value())
		middle := strings.TrimSpace(m.middleInput.Value())
		body := m.bodyInput.Value()

		if pattern == "" {
			m.err = fmt.Errorf("URL pattern cannot be empty")
			return m, nil
		}

		alteration := types.Alteration{URLPattern: pattern, Body: body}
		if m.kind == types.AlterationRequest {
			if middle == "" && body == "" {
				m.err = fmt.Errorf("set a rewrite URL, a body, or both")
				return m, nil
			}
			alteration.RewriteURL = middle
		} else {
			statusCode, err := strconv.Atoi(middle)
			if err != nil || statusCode < 100 || statusCode > 599 {
				m.err = fmt.Errorf("invalid status code: must be 100-599")
				return m, nil
			}
			alteration.StatusCode = statusCode
		}

		m.patternInput.Blur()
		m.middleInput.Blur()
		m.bodyInput.Blur()
		m.loading = true
		return m, m.addAlteration(alteration)
	}

	var cmd tea.Cmd
	switch m.focusedField {
	case 0:
		m.patternInput, cmd = m.patternInput.Update(msg)
	case 1:
		m.middleInput, cmd = m.middleInput.Update(msg)
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
			m.middleInput, cmd = m.middleInput.Update(msg)
		case 2:
			m.bodyInput, cmd = m.bodyInput.Update(msg)
		}
	}
	return m, cmd
}

func (m AlterationsModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render(m.title()))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString(m.spinner.View() + " Loading...")
		return b.String()
	}

	// Confirm delete all
	if m.state == alterationConfirmDeleteAll {
		b.WriteString(m.styles.ErrorText.Render("Delete all " + strings.ToLower(m.title()) + "? (y/n)"))
		b.WriteString("\n\n")
	}

	// Add form
	if m.state == alterationAdd {
		b.WriteString(m.styles.InputLabel.Render("Add " + strings.TrimSuffix(m.title(), "s")))
		b.WriteString("\n")

		middleLabel := "  Status Code: "
		if m.kind == types.AlterationRequest {
			middleLabel = "  Rewrite URL: "
		}
		labels := []string{"  URL Pattern: ", middleLabel, "  Body: "}
		inputs := []string{m.patternInput.View(), m.middleInput.View(), m.bodyInput.View()}
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
	b.WriteString(m.styles.InputLabel.Render(m.title()))
	b.WriteString("\n")
	if len(m.alterations) == 0 {
		b.WriteString(m.styles.HelpStyle.Render("  No " + strings.ToLower(m.title()) + " configured"))
		b.WriteString("\n")
	} else {
		for i, a := range m.alterations {
			line := fmt.Sprintf("  %s → %s", m.truncate(a.URLPattern, 40), m.describe(a))
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

// describe renders the right-hand side of a list row: what the rule does.
func (m AlterationsModel) describe(a types.Alteration) string {
	bodyPreview := m.truncate(a.Body, 30)
	if bodyPreview == "" {
		bodyPreview = "(empty)"
	}
	if m.kind == types.AlterationRequest {
		if a.RewriteURL != "" {
			return a.RewriteURL
		}
		return "body: " + bodyPreview
	}
	return fmt.Sprintf("%d %s", a.StatusCode, bodyPreview)
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
		alts, err := m.client.ListAlterations(m.kind)
		if err != nil {
			return apiErrMsg{err}
		}
		return alterationsLoadedMsg{kind: m.kind, alterations: alts}
	}
}

func (m AlterationsModel) addAlteration(a types.Alteration) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.AddAlteration(m.kind, a); err != nil {
			return apiErrMsg{err}
		}
		return alterationAddedMsg{kind: m.kind}
	}
}

func (m AlterationsModel) deleteAlteration(index int) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.RemoveAlteration(m.kind, index); err != nil {
			return apiErrMsg{err}
		}
		return alterationDeletedMsg{kind: m.kind}
	}
}

func (m AlterationsModel) deleteAllAlterations() tea.Cmd {
	return func() tea.Msg {
		if err := m.client.RemoveAllAlterations(m.kind); err != nil {
			return apiErrMsg{err}
		}
		return allAlterationsDeletedMsg{kind: m.kind}
	}
}
