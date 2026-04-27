package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shuntaka9576/kanban/internal/gh"
)

func newSizedModel(t *testing.T, w, h int) Model {
	t.Helper()
	m := New(nil, gh.ProjectSpec{Title: "Sample Project"}, `"Sample Project"`)
	out, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return out.(Model)
}

func TestView_LoadingState(t *testing.T) {
	m := newSizedModel(t, 120, 40)
	got := m.View()
	if got == "" {
		t.Fatal("loading view returned empty string")
	}
	// Skeleton: a "Resolving …" placeholder is centred in the board frame.
	if !strings.Contains(got, "Resolving") {
		t.Fatalf("loading view missing 'Resolving' placeholder:\n%s", got)
	}
	if !strings.Contains(got, "Sample Project") {
		t.Fatalf("loading view should reference the requested project label:\n%s", got)
	}
	// Footer status spinner should reference resolving in the bottom-right.
	if !strings.Contains(got, "resolving project from GitHub") {
		t.Fatalf("loading view missing footer status:\n%s", got)
	}
	// Help should still be visible on the same footer line.
	if !strings.Contains(got, "h/l col") {
		t.Fatalf("loading view missing help row:\n%s", got)
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

	out, _ := m.Update(bootstrapMsg{project: project})
	got := out.(Model).View()

	for _, want := range []string{"Sample Project", "Todo (1)", "Doing (1)", "Done (1)", "investigate xyz"} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered board missing %q in:\n%s", want, got)
		}
	}
}

func TestView_PaginationBadgeWhilePaging(t *testing.T) {
	m := newSizedModel(t, 200, 40)
	project := &gh.Project{
		ID: "P_1", Number: 2, Title: "Sample Project",
		Status: gh.SingleSelectField{
			ID: "F", Options: []gh.SingleSelectOption{{ID: "todo", Name: "Todo"}},
		},
		Items: []gh.Item{{ID: "i1", Title: "first", StatusOptionID: "todo"}},
	}

	// Bootstrap reports totalCount=4, so the footer should show progress as
	// "1 / 4 items (25%)" while paginating.
	out, _ := m.Update(bootstrapMsg{project: project, nextCursor: "CUR", totalItems: 4})
	got := out.(Model).View()

	if !strings.Contains(got, "1 / 4 items (25%)") {
		t.Fatalf("expected percentage progress 'loaded 1 / 4 items (25%%)' in footer; view:\n%s", got)
	}
}

func TestProgressLabel(t *testing.T) {
	cases := []struct {
		loaded, total int
		want          string
	}{
		{1, 4, "loaded 1 / 4 items (25%)"},
		{50, 100, "loaded 50 / 100 items (50%)"},
		{100, 100, "loaded 100 items, fetching more…"}, // server says we are caught up
		{1, 0, "loaded 1 items, fetching more…"},       // no totalCount available
	}
	for _, tc := range cases {
		got := progressLabel(tc.loaded, tc.total)
		if got != tc.want {
			t.Errorf("progressLabel(%d,%d) = %q, want %q", tc.loaded, tc.total, got, tc.want)
		}
	}
}

func TestAppendItemsGrowsColumns(t *testing.T) {
	m := newSizedModel(t, 120, 40)
	project := &gh.Project{
		ID: "P_1", Number: 2, Title: "Sample Project",
		Status: gh.SingleSelectField{
			ID: "F",
			Options: []gh.SingleSelectOption{
				{ID: "todo", Name: "Todo"},
				{ID: "doing", Name: "Doing"},
			},
		},
		Items: []gh.Item{{ID: "i1", Title: "first", StatusOptionID: "todo"}},
	}

	out, _ := m.Update(bootstrapMsg{project: project, nextCursor: "CUR"})
	out, _ = out.(Model).Update(itemsPageMsg{
		items: []gh.Item{
			{ID: "i2", Title: "second", StatusOptionID: "todo"},
			{ID: "i3", Title: "third", StatusOptionID: "doing"},
			{ID: "i4", Title: "ghost", StatusOptionID: "unknown-option"},
		},
		nextCursor: "",
	})

	got := out.(Model)
	if got.loadedItems != 4 {
		t.Fatalf("loadedItems = %d, want 4", got.loadedItems)
	}
	if got.paginating {
		t.Fatalf("expected paginating=false after final page")
	}
	// columns: Todo, Doing, No Status (auto-created for ghost)
	if len(got.columns) != 3 {
		t.Fatalf("expected 3 columns, got %d (%+v)", len(got.columns), got.columns)
	}
	if got.columns[0].name != "Todo" || len(got.columns[0].items) != 2 {
		t.Fatalf("Todo column wrong: %+v", got.columns[0])
	}
	if got.columns[1].name != "Doing" || len(got.columns[1].items) != 1 {
		t.Fatalf("Doing column wrong: %+v", got.columns[1])
	}
	if got.columns[2].name != "No Status" || len(got.columns[2].items) != 1 {
		t.Fatalf("No Status column wrong: %+v", got.columns[2])
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

	out, _ := m.Update(bootstrapMsg{project: project})
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
