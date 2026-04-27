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
	spec         gh.ProjectSpec
	specLabel    string // human-friendly project label for the loading view (e.g. "Sprint Backlog" or "#2")
	project      *gh.Project
	columns      []column
	focusCol     int
	width        int
	height       int
	bootstrapped bool   // first page has been merged
	paginating   bool   // a follow-up items page is in flight
	nextCursor   string // cursor for the next items page (empty == no more)
	loadedItems  int    // total items currently held in columns
	totalItems   int    // ProjectV2.items.totalCount reported by GitHub
	spinnerFrame int
	movingItem   string
	yanking      string
	status       string
	err          error
}

func New(client *gh.Client, spec gh.ProjectSpec, specLabel string) Model {
	return Model{
		client:    client,
		spec:      spec,
		specLabel: specLabel,
	}
}

func (m *Model) setProject(p *gh.Project) {
	m.project = p
	m.columns = buildColumns(p)
	if m.focusCol >= len(m.columns) {
		m.focusCol = 0
	}
	if p == nil {
		m.loadedItems = 0
		return
	}
	m.loadedItems = len(p.Items)
}

func (m *Model) appendItems(items []gh.Item) {
	if len(items) == 0 || m.project == nil {
		return
	}
	m.project.Items = append(m.project.Items, items...)

	knownOption := make(map[string]int, len(m.columns))
	noStatusIdx := -1
	for i, col := range m.columns {
		if col.optionID == noStatusOptionID {
			noStatusIdx = i
			continue
		}
		knownOption[col.optionID] = i
	}

	known := make(map[string]struct{}, len(m.project.Status.Options))
	for _, opt := range m.project.Status.Options {
		known[opt.ID] = struct{}{}
	}

	for _, item := range items {
		if idx, ok := knownOption[item.StatusOptionID]; ok && item.StatusOptionID != noStatusOptionID {
			m.columns[idx].items = append(m.columns[idx].items, item)
			continue
		}
		// item belongs to "No Status" — either it has no value, or its option
		// is not declared on the project (deleted). Lazily create the column.
		if noStatusIdx == -1 {
			m.columns = append(m.columns, column{optionID: noStatusOptionID, name: "No Status"})
			noStatusIdx = len(m.columns) - 1
		}
		m.columns[noStatusIdx].items = append(m.columns[noStatusIdx].items, item)
	}

	m.loadedItems += len(items)
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
