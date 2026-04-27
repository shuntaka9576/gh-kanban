package cli

import (
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shuntaka9576/kanban/internal/gh"
	"github.com/shuntaka9576/kanban/internal/tui"
)

type ViewCmd struct {
	User    string `short:"u" help:"GitHub user login that owns the project. Mutually exclusive with --org."`
	Org     string `short:"o" help:"GitHub organization login that owns the project. Mutually exclusive with --user."`
	Project string `short:"p" help:"Project title (exact match). Provide either --project or --number."`
	Number  int    `short:"N" help:"Project number. Faster than --project because it skips the search query."`
}

func (c *ViewCmd) Run() error {
	if c.Project == "" && c.Number == 0 {
		return errors.New("specify --project/-p TITLE or --number/-N N")
	}
	if c.Project != "" && c.Number != 0 {
		return errors.New("--project and --number are mutually exclusive")
	}

	client, err := gh.NewClient(gh.InitParams{
		UserLogin: c.User,
		OrgLogin:  c.Org,
	})
	if err != nil {
		if errors.Is(err, gh.ErrInvalidClientType) {
			return errors.New("specify exactly one of --user/-u or --org/-o")
		}
		return err
	}

	spec := gh.ProjectSpec{
		Title:  c.Project,
		Number: c.Number,
	}
	label := specLabel(spec)

	model := tui.New(client, spec, label)
	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return err
	}
	return nil
}

func specLabel(spec gh.ProjectSpec) string {
	if spec.Number > 0 {
		if spec.Title != "" {
			return fmt.Sprintf("%q (#%d)", spec.Title, spec.Number)
		}
		return fmt.Sprintf("#%d", spec.Number)
	}
	return fmt.Sprintf("%q", spec.Title)
}
