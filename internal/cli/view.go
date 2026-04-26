package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shuntaka9576/kanban/internal/gh"
	"github.com/shuntaka9576/kanban/internal/tui"
)

type ViewCmd struct {
	User    string `short:"u" help:"GitHub user login that owns the project. Mutually exclusive with --org."`
	Org     string `short:"o" help:"GitHub organization login that owns the project. Mutually exclusive with --user."`
	Project string `short:"p" required:"" help:"Project title (exact match)."`
}

func (c *ViewCmd) Run() error {
	client, err := gh.NewClient(gh.InitParams{
		UserLogin: c.User,
		OrgLogin:  c.Org,
	})
	if err != nil {
		if errors.Is(err, gh.ErrInvalidClientType) {
			return fmt.Errorf("specify exactly one of --user/-u or --org/-o")
		}
		return err
	}

	projects, err := client.ListProjects()
	if err != nil {
		return err
	}

	var match *gh.ProjectSummary
	for i := range projects {
		if projects[i].Title == c.Project {
			match = &projects[i]
			break
		}
	}
	if match == nil {
		return fmt.Errorf("project %q not found%s", c.Project, formatProjectList(projects))
	}

	model := tui.New(client, *match)
	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return err
	}
	return nil
}

func formatProjectList(projects []gh.ProjectSummary) string {
	if len(projects) == 0 {
		return ". No projects found for the given owner."
	}
	titles := make([]string, len(projects))
	for i, p := range projects {
		titles[i] = fmt.Sprintf("  - %s (#%d)", p.Title, p.Number)
	}
	sort.Strings(titles)
	return ". Available projects:\n" + strings.Join(titles, "\n")
}
