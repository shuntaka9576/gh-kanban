package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/shuntaka9576/kanban/internal/gh"
)

func renderItemMarkdown(ctx *gh.ItemContext) string {
	if ctx == nil {
		return ""
	}

	var b strings.Builder

	header := ctx.Title
	if ctx.Number > 0 {
		header = fmt.Sprintf("#%d %s", ctx.Number, ctx.Title)
	}
	switch ctx.ContentType {
	case gh.ContentDraftIssue:
		header = "[Draft] " + header
	case gh.ContentPullRequest:
		header = "[PR] " + header
	}
	fmt.Fprintf(&b, "# %s\n\n", header)

	writeMeta(&b, "Type", string(ctx.ContentType))
	if ctx.RepoNameOwner != "" {
		writeMeta(&b, "Repository", ctx.RepoNameOwner)
	}
	if ctx.URL != "" {
		writeMeta(&b, "URL", ctx.URL)
	}
	if ctx.State != "" {
		writeMeta(&b, "State", ctx.State)
	}
	if ctx.Author != "" {
		writeMeta(&b, "Author", "@"+ctx.Author)
	}
	if !ctx.CreatedAt.IsZero() {
		writeMeta(&b, "Created", ctx.CreatedAt.UTC().Format(time.RFC3339))
	}
	if len(ctx.Assignees) > 0 {
		writeMeta(&b, "Assignees", "@"+strings.Join(ctx.Assignees, " @"))
	}
	if len(ctx.Labels) > 0 {
		writeMeta(&b, "Labels", strings.Join(ctx.Labels, ", "))
	}

	b.WriteString("\n## Body\n\n")
	if strings.TrimSpace(ctx.Body) == "" {
		b.WriteString("_(no body)_\n")
	} else {
		b.WriteString(strings.TrimRight(ctx.Body, "\n"))
		b.WriteString("\n")
	}

	if ctx.ContentType == gh.ContentDraftIssue {
		return b.String()
	}

	b.WriteString("\n## Comments")
	if ctx.CommentsCapped {
		b.WriteString(" (truncated to first 100)")
	}
	b.WriteString("\n\n")

	if len(ctx.Comments) == 0 {
		b.WriteString("_(no comments)_\n")
		return b.String()
	}
	for i, c := range ctx.Comments {
		login := c.Author
		if login == "" {
			login = "ghost"
		}
		ts := ""
		if !c.CreatedAt.IsZero() {
			ts = " — " + c.CreatedAt.UTC().Format(time.RFC3339)
		}
		fmt.Fprintf(&b, "### %d. @%s%s\n\n", i+1, login, ts)
		body := strings.TrimRight(c.Body, "\n")
		if body == "" {
			body = "_(empty)_"
		}
		b.WriteString(body)
		b.WriteString("\n\n")
	}
	return b.String()
}

func writeMeta(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "- **%s**: %s\n", key, value)
}
