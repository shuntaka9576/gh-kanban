package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shuntaka9576/kanban/internal/gh"
)

func newSizedModel(t *testing.T, w, h int) Model {
	t.Helper()
	m := New(nil, gh.ProjectSummary{ID: "P_1", Number: 2, Title: "Sample Project"})
	out, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return out.(Model)
}

func TestView_LoadingState(t *testing.T) {
	m := newSizedModel(t, 120, 40)
	got := m.View()
	if got == "" {
		t.Fatal("loading view returned empty string")
	}
	if !strings.Contains(got, "Loading") {
		t.Fatalf("loading view missing 'Loading' label:\n%s", got)
	}
	if !strings.Contains(got, "Sample Project") {
		t.Fatalf("loading view should reference project title:\n%s", got)
	}
}

func TestView_RendersBoardAfterLoad(t *testing.T) {
	m := newSizedModel(t, 120, 40)
	project := &gh.Project{
		ID:     "P_1",
		Number: 2,
		Title:  "Sample Project",
		URL:    "https://github.com/orgs/acme/projects/2",
		Status: gh.SingleSelectField{
			ID: "F_status",
			Options: []gh.SingleSelectOption{
				{ID: "todo", Name: "Todo"},
				{ID: "doing", Name: "Doing"},
				{ID: "done", Name: "Done"},
			},
		},
		Items: []gh.Item{
			{ID: "i1", Title: "investigate xyz", Number: 11, ContentType: gh.ContentIssue, StatusOptionID: "todo"},
			{ID: "i2", Title: "implement queue", Number: 12, ContentType: gh.ContentIssue, StatusOptionID: "doing"},
			{ID: "i3", Title: "ship release", Number: 13, ContentType: gh.ContentIssue, StatusOptionID: "done"},
		},
	}

	out, _ := m.Update(projectLoadedMsg{project: project})
	got := out.(Model).View()

	for _, want := range []string{"Sample Project", "Todo (1)", "Doing (1)", "Done (1)", "investigate xyz"} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered board missing %q in:\n%s", want, got)
		}
	}
}

func TestView_RendersOverflowIndicators(t *testing.T) {
	m := newSizedModel(t, 80, 16)
	items := make([]gh.Item, 0, 30)
	for i := 0; i < 30; i++ {
		items = append(items, gh.Item{
			ID:             "i" + itoa(i),
			Title:          "card " + itoa(i),
			Number:         i,
			ContentType:    gh.ContentIssue,
			StatusOptionID: "todo",
		})
	}
	project := &gh.Project{
		ID:     "P_1",
		Number: 2,
		Title:  "Sample Project",
		Status: gh.SingleSelectField{
			ID:      "F_status",
			Options: []gh.SingleSelectOption{{ID: "todo", Name: "Todo"}},
		},
		Items: items,
	}

	out, _ := m.Update(projectLoadedMsg{project: project})
	got := out.(Model).View()

	if !strings.Contains(got, "more") {
		t.Fatalf("expected '+ N more' indicator when items exceed visible rows. View:\n%s", got)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
