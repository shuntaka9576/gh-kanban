package tui

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shuntaka9576/kanban/internal/gh"
)

func TestBuildColumns(t *testing.T) {
	t.Parallel()

	project := &gh.Project{
		ID: "PROJ",
		Status: gh.SingleSelectField{
			ID: "FIELD",
			Options: []gh.SingleSelectOption{
				{ID: "todo", Name: "Todo"},
				{ID: "doing", Name: "Doing"},
				{ID: "done", Name: "Done"},
			},
		},
		Items: []gh.Item{
			{ID: "i1", Title: "A", StatusOptionID: "todo"},
			{ID: "i2", Title: "B", StatusOptionID: "doing"},
			{ID: "i3", Title: "C", StatusOptionID: "todo"},
			{ID: "i4", Title: "D", StatusOptionID: ""},
			{ID: "i5", Title: "E", StatusOptionID: "ghost"},
		},
	}

	got := buildColumns(project)
	want := []column{
		{optionID: "todo", name: "Todo", items: []gh.Item{
			{ID: "i1", Title: "A", StatusOptionID: "todo"},
			{ID: "i3", Title: "C", StatusOptionID: "todo"},
		}},
		{optionID: "doing", name: "Doing", items: []gh.Item{
			{ID: "i2", Title: "B", StatusOptionID: "doing"},
		}},
		{optionID: "done", name: "Done"},
		{optionID: noStatusOptionID, name: "No Status", items: []gh.Item{
			{ID: "i4", Title: "D", StatusOptionID: ""},
			{ID: "i5", Title: "E", StatusOptionID: "ghost"},
		}},
	}

	if diff := cmp.Diff(want, got, cmp.AllowUnexported(column{}), cmpopts.EquateEmpty()); diff != "" {
		t.Fatalf("buildColumns mismatch (-want +got):\n%s", diff)
	}
}

func TestWindowItems(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name             string
		focused          bool
		cursor, total    int
		rows             int
		wantStart, wantE int
	}{
		{"empty", true, 0, 0, 5, 0, 0},
		{"fits without scroll", true, 3, 5, 10, 0, 5},
		{"unfocused shows top window", false, 99, 30, 5, 0, 5},
		{"cursor near top stays at 0", true, 1, 30, 6, 0, 6},
		{"cursor in middle centers", true, 15, 30, 6, 12, 18},
		{"cursor near bottom anchors at end", true, 29, 30, 6, 24, 30},
		{"cursor exactly at end", true, 9, 10, 4, 6, 10},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotS, gotE := windowItems(tc.focused, tc.cursor, tc.total, tc.rows)
			if gotS != tc.wantStart || gotE != tc.wantE {
				t.Fatalf("windowItems(%v,%d,%d,%d) = (%d,%d), want (%d,%d)",
					tc.focused, tc.cursor, tc.total, tc.rows, gotS, gotE, tc.wantStart, tc.wantE)
			}
		})
	}
}

func TestBuildColumns_NoOrphans(t *testing.T) {
	t.Parallel()

	project := &gh.Project{
		Status: gh.SingleSelectField{
			ID: "F",
			Options: []gh.SingleSelectOption{
				{ID: "todo", Name: "Todo"},
			},
		},
		Items: []gh.Item{
			{ID: "i1", StatusOptionID: "todo"},
		},
	}

	got := buildColumns(project)
	if len(got) != 1 {
		t.Fatalf("expected 1 column, got %d", len(got))
	}
	if got[0].name != "Todo" {
		t.Fatalf("expected Todo column, got %q", got[0].name)
	}
}
