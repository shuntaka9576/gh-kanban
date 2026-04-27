package tui

import (
	"errors"
	"fmt"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/browser"
	"github.com/shuntaka9576/kanban/internal/gh"
)

type bootstrapMsg struct {
	project    *gh.Project
	nextCursor string
	totalItems int
	err        error
}

type itemsPageMsg struct {
	items      []gh.Item
	nextCursor string
	totalItems int
	err        error
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

var errNoStatusField = errors.New("project has no Status SingleSelect field; gh-kanban requires it")

func bootstrapCmd(client *gh.Client, spec gh.ProjectSpec) tea.Cmd {
	return func() tea.Msg {
		res, err := client.BootstrapBySpec(spec)
		if err != nil {
			return bootstrapMsg{err: err}
		}
		if res.Project != nil && res.Project.Status.ID == "" {
			return bootstrapMsg{err: errNoStatusField}
		}
		return bootstrapMsg{
			project:    res.Project,
			nextCursor: res.NextCursor,
			totalItems: res.TotalItems,
		}
	}
}

func fetchItemsPageCmd(client *gh.Client, projectID, statusFieldID, cursor string) tea.Cmd {
	return func() tea.Msg {
		page, err := client.FetchItemsPage(projectID, statusFieldID, cursor)
		if err != nil {
			return itemsPageMsg{err: err}
		}
		return itemsPageMsg{
			items:      page.Items,
			nextCursor: page.NextCursor,
			totalItems: page.TotalItems,
		}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(bootstrapCmd(m.client, m.spec), tickCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		if !m.bootstrapped || m.paginating || m.yanking != "" {
			m.spinnerFrame++
			return m, tickCmd()
		}
		return m, nil

	case bootstrapMsg:
		if msg.err != nil {
			m.err = msg.err
			m.bootstrapped = true // unblock keys; user can quit/refresh
			m.paginating = false
			return m, nil
		}
		m.err = nil
		m.bootstrapped = true
		m.setProject(msg.project)
		m.nextCursor = msg.nextCursor
		m.totalItems = msg.totalItems
		if m.nextCursor != "" {
			m.paginating = true
			return m, tea.Batch(
				fetchItemsPageCmd(m.client, m.project.ID, m.project.Status.ID, m.nextCursor),
				tickCmd(),
			)
		}
		m.paginating = false
		return m, nil

	case itemsPageMsg:
		if msg.err != nil {
			m.err = msg.err
			m.paginating = false
			return m, nil
		}
		m.appendItems(msg.items)
		m.nextCursor = msg.nextCursor
		if msg.totalItems > 0 {
			m.totalItems = msg.totalItems
		}
		if m.nextCursor != "" && m.project != nil {
			m.paginating = true
			return m, fetchItemsPageCmd(m.client, m.project.ID, m.project.Status.ID, m.nextCursor)
		}
		m.paginating = false
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

	if !m.bootstrapped || len(m.columns) == 0 {
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
	// Wipe board state but keep spec/specLabel so the loading view shows.
	m.project = nil
	m.columns = nil
	m.focusCol = 0
	m.bootstrapped = false
	m.paginating = false
	m.nextCursor = ""
	m.loadedItems = 0
	m.totalItems = 0
	m.err = nil
	return m, tea.Batch(bootstrapCmd(m.client, m.spec), tickCmd())
}
