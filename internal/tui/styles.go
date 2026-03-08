package tui

import "charm.land/lipgloss/v2"

type Styles struct {
	ActiveTab   lipgloss.Style
	InactiveTab lipgloss.Style
	TabBar      lipgloss.Style
	Title       lipgloss.Style
	SelectedRow lipgloss.Style
	NormalRow   lipgloss.Style
	StatusBar   lipgloss.Style
	ErrorText   lipgloss.Style
	SuccessText lipgloss.Style
	HelpStyle   lipgloss.Style
	InputLabel  lipgloss.Style
	Border      lipgloss.Style
}

func NewStyles() Styles {
	accent := lipgloss.Color("#7571F9")
	errClr := lipgloss.Color("#FF6666")
	successClr := lipgloss.Color("#66FF66")
	muted := lipgloss.Color("#666666")
	bgHl := lipgloss.Color("#2A2950")

	return Styles{
		ActiveTab: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(accent).
			Padding(0, 2),
		InactiveTab: lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 2),
		TabBar: lipgloss.NewStyle().
			MarginBottom(1),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			MarginBottom(1),
		SelectedRow: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Background(bgHl),
		NormalRow: lipgloss.NewStyle(),
		StatusBar: lipgloss.NewStyle().
			Foreground(muted).
			MarginTop(1),
		ErrorText: lipgloss.NewStyle().
			Foreground(errClr).
			Bold(true),
		SuccessText: lipgloss.NewStyle().
			Foreground(successClr),
		HelpStyle: lipgloss.NewStyle().
			Foreground(muted),
		InputLabel: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent),
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(1, 2),
	}
}
