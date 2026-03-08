package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/mitchgrogg/rita-devtools-tui/internal/api"
	"github.com/mitchgrogg/rita-devtools-tui/internal/tui"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	apiURL := flag.String("api", "", "rita-mitm API base URL (or set RITA_MITM_URL)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("rita-devtools-tui %s (%s)\n", version, commit)
		os.Exit(0)
	}

	if *apiURL == "" {
		*apiURL = os.Getenv("RITA_MITM_URL")
	}
	if *apiURL == "" {
		url, err := promptURL()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		*apiURL = url
	}

	if !strings.HasPrefix(*apiURL, "http://") && !strings.HasPrefix(*apiURL, "https://") {
		fmt.Fprintf(os.Stderr, "Error: API URL must include a scheme (e.g. http://%s)\n", *apiURL)
		os.Exit(1)
	}

	*apiURL = strings.TrimSpace(*apiURL)
	client := api.New(*apiURL)
	app := tui.NewApp(client)
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// promptURL runs a mini bubbletea program to prompt for the API URL with
// proper line-editing support (arrow keys, backspace, etc.).
func promptURL() (string, error) {
	m := promptModel{}
	m.input = textinput.New()
	m.input.Prompt = "rita-mitm API URL: "
	m.input.Placeholder = "http://host:8082"
	m.input.SetWidth(60)
	m.input.Focus()

	p := tea.NewProgram(&m)
	result, err := p.Run()
	if err != nil {
		return "", err
	}
	pm := result.(*promptModel)
	if pm.cancelled {
		return "", fmt.Errorf("cancelled")
	}
	val := strings.TrimSpace(pm.input.Value())
	if val == "" {
		return "", fmt.Errorf("API URL is required")
	}
	return val, nil
}

type promptModel struct {
	input     textinput.Model
	done      bool
	cancelled bool
}

func (m *promptModel) Init() tea.Cmd {
	return m.input.Focus()
}

func (m *promptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *promptModel) View() tea.View {
	if m.done || m.cancelled {
		return tea.NewView("")
	}
	return tea.NewView(m.input.View())
}
