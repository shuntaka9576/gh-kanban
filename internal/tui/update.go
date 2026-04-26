package tui

import (
	"fmt"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/browser"
	"github.com/shuntaka9576/kanban/internal/gh"
)

type projectLoadedMsg struct {
	project *gh.Project
	err     error
}

type itemMovedMsg struct {
	itemID string
	err    error
}

type itemYankedMsg struct {
	itemID   string
	title    string
	comments int
	capped   bool
	err      error
}

type tickMsg struct{}

type clearStatusMsg struct{}

const (
	spinnerInterval = 120 * time.Millisecond
	statusLifetime  = 4 * time.Second
)

func tickCmd() tea.Cmd {
	return tea.Tick(spinnerInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return clearStatusMsg{} })
}

func loadProjectCmd(client *gh.Client, projectID string) tea.Cmd {
	return func() tea.Msg {
		p, err := client.FetchProjectBoard(projectID)
		if err == nil && p != nil && p.Status.ID == "" {
			err = errNoStatusField
		}
		return projectLoadedMsg{project: p, err: err}
	}
}

var errNoStatusField = errStatusField("project has no Status SingleSelect field; gh-kanban requires it")

type errStatusField string

func (e errStatusField) Error() string { return string(e) }

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadProjectCmd(m.client, m.summary.ID), tickCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		if m.loading || m.yanking != "" {
			m.spinnerFrame++
			return m, tickCmd()
		}
		return m, nil

	case projectLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.setProject(msg.project)
		return m, nil

	case itemMovedMsg:
		m.movingItem = ""
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		return m.refresh()

	case itemYankedMsg:
		m.yanking = ""
		if msg.err != nil {
			m.err = msg.err
			return m, clearStatusAfter(statusLifetime)
		}
		more := ""
		if msg.capped {
			more = " (truncated to first 100)"
		}
		m.status = fmt.Sprintf("✔ Copied %q (%d comments%s) as Markdown.", msg.title, msg.comments, more)
		return m, clearStatusAfter(statusLifetime)

	case clearStatusMsg:
		m.status = ""
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "R":
		return m.refresh()
	}

	if m.loading || len(m.columns) == 0 {
		return m, nil
	}

	col := &m.columns[m.focusCol]

	switch msg.String() {
	case "h", "left":
		m.focusCol = m.leftCol()

	case "l", "right":
		m.focusCol = m.rightCol()

	case "j", "down":
		if len(col.items) > 0 {
			col.cursor = (col.cursor + 1) % len(col.items)
		}

	case "k", "up":
		if len(col.items) > 0 {
			col.cursor = (col.cursor - 1 + len(col.items)) % len(col.items)
		}

	case "n":
		return m.moveItem(m.rightCol())

	case "b":
		return m.moveItem(m.leftCol())

	case "O":
		if m.project != nil && m.project.URL != "" {
			_ = browser.OpenURL(m.project.URL)
		}

	case "o":
		if it := m.currentItem(); it != nil && it.URL != "" {
			_ = browser.OpenURL(it.URL)
		}

	case "y":
		return m.yankItem()
	}
	return m, nil
}

func (m Model) moveItem(targetCol int) (tea.Model, tea.Cmd) {
	if m.project == nil || m.project.Status.ID == "" {
		return m, nil
	}
	if targetCol == m.focusCol {
		return m, nil
	}
	target := m.columns[targetCol]
	if target.optionID == noStatusOptionID {
		return m, nil
	}
	item := m.currentItem()
	if item == nil {
		return m, nil
	}

	projectID := m.project.ID
	fieldID := m.project.Status.ID
	itemID := item.ID
	optionID := target.optionID
	client := m.client

	m.movingItem = itemID

	cmd := func() tea.Msg {
		err := client.UpdateItemStatus(projectID, itemID, fieldID, optionID)
		return itemMovedMsg{itemID: itemID, err: err}
	}
	return m, cmd
}

func (m Model) yankItem() (tea.Model, tea.Cmd) {
	item := m.currentItem()
	if item == nil {
		return m, nil
	}
	itemID := item.ID
	client := m.client

	m.yanking = itemID
	m.status = "Fetching item context..."

	cmd := tea.Batch(
		func() tea.Msg {
			ctx, err := client.FetchItemContext(itemID)
			if err != nil {
				return itemYankedMsg{itemID: itemID, err: err}
			}
			md := renderItemMarkdown(ctx)
			if err := clipboard.WriteAll(md); err != nil {
				return itemYankedMsg{itemID: itemID, err: fmt.Errorf("clipboard: %w", err)}
			}
			return itemYankedMsg{
				itemID:   itemID,
				title:    ctx.Title,
				comments: len(ctx.Comments),
				capped:   ctx.CommentsCapped,
			}
		},
		tickCmd(),
	)
	return m, cmd
}

func (m Model) refresh() (tea.Model, tea.Cmd) {
	projectID := m.summary.ID
	if projectID == "" && m.project != nil {
		projectID = m.project.ID
	}
	if projectID == "" {
		return m, nil
	}
	m.loading = true
	return m, tea.Batch(loadProjectCmd(m.client, projectID), tickCmd())
}
