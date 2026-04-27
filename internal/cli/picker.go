package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shuntaka9576/kanban/internal/gh"
)

type pickerModel struct {
	projects []gh.ProjectSummary
	cursor   int
	selected *gh.ProjectSummary
}

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "ctrl+c", "q", "esc":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.projects)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		p := m.projects[m.cursor]
		m.selected = &p
		return m, tea.Quit
	}
	return m, nil
}

func (m pickerModel) View() string {
	var b strings.Builder
	b.WriteString("Select a project (j/k or ↑/↓, enter to confirm, q to cancel)\n\n")
	for i, p := range m.projects {
		prefix := "  "
		if i == m.cursor {
			prefix = "▶ "
		}
		fmt.Fprintf(&b, "%s#%d  %s\n", prefix, p.Number, p.Title)
	}
	return b.String()
}

func pickProject(projects []gh.ProjectSummary) (*gh.ProjectSummary, error) {
	final, err := tea.NewProgram(pickerModel{projects: projects}).Run()
	if err != nil {
		return nil, err
	}
	return final.(pickerModel).selected, nil
}
