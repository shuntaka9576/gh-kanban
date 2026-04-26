package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/shuntaka9576/kanban/internal/gh"
)

func TestRenderItemMarkdown_Issue(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	c1 := time.Date(2026, 4, 2, 10, 30, 0, 0, time.UTC)
	c2 := time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)
	ctx := &gh.ItemContext{
		ContentType:   gh.ContentIssue,
		RepoNameOwner: "acme/app",
		Number:        42,
		Title:         "Investigate flaky test",
		Body:          "Steps to reproduce...\n1. run\n2. observe",
		URL:           "https://github.com/acme/app/issues/42",
		State:         "OPEN",
		Author:        "alice",
		CreatedAt:     created,
		Assignees:     []string{"bob", "carol"},
		Labels:        []string{"bug", "priority:high"},
		Comments: []gh.Comment{
			{Author: "bob", Body: "I can repro on macOS.", CreatedAt: c1},
			{Author: "", Body: "lgtm", CreatedAt: c2},
		},
	}

	got := renderItemMarkdown(ctx)
	mustContain(t, got,
		"# #42 Investigate flaky test",
		"- **Type**: Issue",
		"- **Repository**: acme/app",
		"- **URL**: https://github.com/acme/app/issues/42",
		"- **State**: OPEN",
		"- **Author**: @alice",
		"- **Created**: 2026-04-01T09:00:00Z",
		"- **Assignees**: @bob @carol",
		"- **Labels**: bug, priority:high",
		"## Body",
		"Steps to reproduce...",
		"## Comments",
		"### 1. @bob — 2026-04-02T10:30:00Z",
		"I can repro on macOS.",
		"### 2. @ghost — 2026-04-03T11:00:00Z",
		"lgtm",
	)
}

func TestRenderItemMarkdown_DraftIssueSkipsComments(t *testing.T) {
	t.Parallel()
	ctx := &gh.ItemContext{
		ContentType: gh.ContentDraftIssue,
		Title:       "Brainstorm",
		Body:        "ideas",
	}
	got := renderItemMarkdown(ctx)
	mustContain(t, got, "# [Draft] Brainstorm", "## Body", "ideas")
	if strings.Contains(got, "## Comments") {
		t.Fatalf("draft issue rendering should not include Comments section, got:\n%s", got)
	}
}

func TestRenderItemMarkdown_PRWithCappedComments(t *testing.T) {
	t.Parallel()
	ctx := &gh.ItemContext{
		ContentType:    gh.ContentPullRequest,
		Title:          "Refactor queue",
		Number:         99,
		CommentsCapped: true,
	}
	got := renderItemMarkdown(ctx)
	mustContain(t, got,
		"# [PR] #99 Refactor queue",
		"## Comments (truncated to first 100)",
		"_(no comments)_",
	)
}

func mustContain(t *testing.T, haystack string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(haystack, n) {
			t.Fatalf("output missing %q\n--- output ---\n%s", n, haystack)
		}
	}
}
