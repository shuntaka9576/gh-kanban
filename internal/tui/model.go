package tui

import (
	"github.com/shuntaka9576/kanban/internal/gh"
)

const noStatusOptionID = ""

type column struct {
	optionID string
	name     string
	items    []gh.Item
	cursor   int
}

type Model struct {
	client       *gh.Client
	summary      gh.ProjectSummary
	project      *gh.Project
	columns      []column
	focusCol     int
	width        int
	height       int
	loading      bool
	spinnerFrame int
	movingItem   string
	yanking      string
	status       string
	err          error
}

func New(client *gh.Client, summary gh.ProjectSummary) Model {
	return Model{
		client:  client,
		summary: summary,
		loading: true,
	}
}

func (m *Model) setProject(p *gh.Project) {
	m.project = p
	m.columns = buildColumns(p)
	if m.focusCol >= len(m.columns) {
		m.focusCol = 0
	}
}

func buildColumns(p *gh.Project) []column {
	if p == nil {
		return nil
	}
	cols := make([]column, 0, len(p.Status.Options)+1)
	for _, opt := range p.Status.Options {
		cols = append(cols, column{optionID: opt.ID, name: opt.Name})
	}

	hasNoStatus := false
	for _, item := range p.Items {
		placed := false
		for i := range cols {
			if cols[i].optionID == item.StatusOptionID && item.StatusOptionID != noStatusOptionID {
				cols[i].items = append(cols[i].items, item)
				placed = true
				break
			}
		}
		if !placed {
			hasNoStatus = true
		}
	}

	if hasNoStatus {
		cols = append(cols, column{optionID: noStatusOptionID, name: "No Status"})
		for _, item := range p.Items {
			belongs := item.StatusOptionID == noStatusOptionID
			if !belongs {
				known := false
				for _, opt := range p.Status.Options {
					if opt.ID == item.StatusOptionID {
						known = true
						break
					}
				}
				belongs = !known
			}
			if belongs {
				idx := len(cols) - 1
				cols[idx].items = append(cols[idx].items, item)
			}
		}
	}
	return cols
}

func (m *Model) leftCol() int {
	if len(m.columns) == 0 {
		return 0
	}
	if m.focusCol-1 < 0 {
		return len(m.columns) - 1
	}
	return m.focusCol - 1
}

func (m *Model) rightCol() int {
	if len(m.columns) == 0 {
		return 0
	}
	if m.focusCol+1 >= len(m.columns) {
		return 0
	}
	return m.focusCol + 1
}

func (m *Model) currentItem() *gh.Item {
	if len(m.columns) == 0 {
		return nil
	}
	col := m.columns[m.focusCol]
	if len(col.items) == 0 {
		return nil
	}
	return &col.items[col.cursor]
}
