package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/shuntaka9576/kanban/internal/gh"
)

const (
	titleLines = 1
	helpLines  = 1
	bodyTotal  = 6
	minColW    = 22
	minBoardH  = 8
)

var (
	columnStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	focusedColumnStyle = columnStyle.
				BorderForeground(lipgloss.Color("40")).
				Foreground(lipgloss.Color("40"))

	cardStyle = lipgloss.NewStyle()

	selectedCardStyle = cardStyle.
				Foreground(lipgloss.Color("207")).
				Bold(true)

	movingCardStyle = cardStyle.
			Foreground(lipgloss.Color("208")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	titleStyle = lipgloss.NewStyle().
			Bold(true)

	bodyStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Always render the same skeleton (title / board / body / footer). Empty
	// states show a placeholder so the user sees the frame at t=0 and the
	// async progress is reported in the bottom-right footer cell.

	boardLines := m.height - titleLines - bodyTotal - helpLines - 3
	if boardLines < minBoardH {
		boardLines = minBoardH
	}

	header := m.renderHeader()

	var board string
	if m.bootstrapped && m.project != nil && len(m.columns) > 0 {
		board = m.renderBoard(boardLines)
	} else {
		board = m.renderBoardPlaceholder(boardLines)
	}

	body := m.renderBody(bodyTotal)
	footer := m.renderFooter()

	return strings.Join([]string{header, board, body, footer}, "\n")
}

func (m Model) renderHeader() string {
	if m.project != nil {
		return titleStyle.Render(fmt.Sprintf("%s  #%d", m.project.Title, m.project.Number))
	}
	label := m.specLabel
	if label == "" {
		label = "project"
	}
	return titleStyle.Render(label)
}

func (m Model) renderBoardPlaceholder(boardLines int) string {
	width := m.width - 2
	if width < 20 {
		width = 20
	}
	contentH := boardLines - 2
	if contentH < 3 {
		contentH = 3
	}

	var msg string
	if m.err != nil {
		msg = fmt.Sprintf("Error: %v", m.err)
	} else if !m.bootstrapped {
		label := m.specLabel
		if label == "" {
			label = "project"
		}
		msg = "Resolving " + label + "…"
	} else {
		msg = "No items in this project."
	}

	body := lipgloss.Place(width, contentH, lipgloss.Center, lipgloss.Center, mutedStyle.Render(msg))
	return columnStyle.Width(width).Render(body)
}

func (m Model) statusMessage() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("⚠ %v", m.err))
	}
	spin := spinnerFrames[m.spinnerFrame%len(spinnerFrames)]
	switch {
	case !m.bootstrapped:
		return mutedStyle.Render(spin + " resolving project from GitHub…")
	case m.yanking != "":
		return mutedStyle.Render(spin + " yanking…")
	case m.paginating:
		return mutedStyle.Render(spin + " " + progressLabel(m.loadedItems, m.totalItems))
	case m.status != "":
		return mutedStyle.Render(m.status)
	default:
		return mutedStyle.Render(fmt.Sprintf("✓ ready · %d items", m.loadedItems))
	}
}

func progressLabel(loaded, total int) string {
	if total <= 0 || loaded >= total {
		return fmt.Sprintf("loaded %d items, fetching more…", loaded)
	}
	pct := loaded * 100 / total
	return fmt.Sprintf("loaded %d / %d items (%d%%)", loaded, total, pct)
}

func (m Model) renderFooter() string {
	help := helpText()
	status := m.statusMessage()

	width := m.width
	if width <= 0 {
		return helpStyle.Render(help)
	}

	helpW := lipgloss.Width(help)
	statusW := lipgloss.Width(status)

	// If they fit on one line, place help left and status right with a gap.
	if helpW+statusW+1 <= width {
		gap := width - helpW - statusW
		if gap < 1 {
			gap = 1
		}
		return helpStyle.Render(help) + strings.Repeat(" ", gap) + status
	}
	// Narrow terminal: drop the help text in favour of the status line.
	return lipgloss.PlaceHorizontal(width, lipgloss.Right, status)
}

func helpText() string {
	return "h/l col  j/k cursor  n/b move  o open  O project  y yank-md  R reload  q quit"
}

func (m Model) renderBoard(boardLines int) string {
	visibleCount := max(1, m.width/minColW)
	if visibleCount > len(m.columns) {
		visibleCount = len(m.columns)
	}
	firstCol := 0
	if len(m.columns) > visibleCount {
		half := visibleCount / 2
		firstCol = m.focusCol - half
		if firstCol < 0 {
			firstCol = 0
		}
		if firstCol+visibleCount > len(m.columns) {
			firstCol = len(m.columns) - visibleCount
		}
	}
	colWidth := m.width / visibleCount
	if colWidth < minColW {
		colWidth = minColW
	}

	contentRows := boardLines - 2 // top + bottom border
	if contentRows < 4 {
		contentRows = 4
	}
	cardRows := contentRows - 2 // header + separator
	if cardRows < 1 {
		cardRows = 1
	}

	textW := colWidth - 4 // border + horizontal padding (1 each side)
	if textW < 6 {
		textW = 6
	}

	rendered := make([]string, 0, visibleCount)
	for offsetCol := 0; offsetCol < visibleCount; offsetCol++ {
		i := firstCol + offsetCol
		col := m.columns[i]
		focused := i == m.focusCol
		style := columnStyle
		if focused {
			style = focusedColumnStyle
		}

		startIdx, endIdx := windowItems(focused, col.cursor, len(col.items), cardRows)

		header := truncate(fmt.Sprintf("%s (%d)", col.name, len(col.items)), textW)
		sep := strings.Repeat("─", textW)
		lines := make([]string, 0, contentRows)
		lines = append(lines, header, sep)

		if startIdx > 0 {
			lines[1] = mutedStyle.Render(truncate(fmt.Sprintf("↑ %d above", startIdx), textW))
		}

		for j := startIdx; j < endIdx; j++ {
			item := col.items[j]
			label := cardLabel(item, textW)
			cs := cardStyle
			if focused && j == col.cursor {
				cs = selectedCardStyle
			}
			if m.movingItem != "" && item.ID == m.movingItem {
				cs = movingCardStyle
			}
			lines = append(lines, cs.Render(label))
		}
		if endIdx < len(col.items) {
			more := len(col.items) - endIdx
			// Replace last visible card line with "+N more" indicator.
			lines[len(lines)-1] = mutedStyle.Render(truncate(fmt.Sprintf("+ %d more", more), textW))
		}

		// Pad to exact contentRows so JoinHorizontal aligns and box height is stable.
		for len(lines) < contentRows {
			lines = append(lines, "")
		}
		if len(lines) > contentRows {
			lines = lines[:contentRows]
		}

		rendered = append(rendered, style.Width(colWidth-2).Render(strings.Join(lines, "\n")))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

// windowItems returns [start, end) indices of items to render so the cursor
// stays visible when the column is focused. Unfocused columns simply show the
// top `rows` items.
func windowItems(focused bool, cursor, total, rows int) (int, int) {
	if total == 0 {
		return 0, 0
	}
	if total <= rows || !focused {
		end := total
		if end > rows {
			end = rows
		}
		return 0, end
	}
	half := rows / 2
	start := cursor - half
	if start < 0 {
		start = 0
	}
	if start+rows > total {
		start = total - rows
	}
	return start, start + rows
}

func (m Model) renderBody(bodyHeight int) string {
	contentH := bodyHeight - 2 // border
	if contentH < 1 {
		contentH = 1
	}
	width := m.width - 2
	if width < 20 {
		width = 20
	}
	textW := width - 2 // padding
	if textW < 10 {
		textW = 10
	}

	item := m.currentItem()
	var parts []string
	if item == nil {
		parts = []string{"(no selection)"}
	} else {
		parts = append(parts, truncate(titleRender(*item), textW))
		if item.URL != "" {
			parts = append(parts, truncate(item.URL, textW))
		}
		meta := ""
		if len(item.Assignees) > 0 {
			meta = "@" + strings.Join(item.Assignees, " @")
		}
		if len(item.Labels) > 0 {
			if meta != "" {
				meta += "  "
			}
			meta += "[" + strings.Join(item.Labels, "][") + "]"
		}
		if meta != "" {
			parts = append(parts, mutedStyle.Render(truncate(meta, textW)))
		}
		if item.Body != "" {
			parts = append(parts, truncate(firstLine(item.Body), textW))
		}
	}

	for len(parts) < contentH {
		parts = append(parts, "")
	}
	if len(parts) > contentH {
		parts = parts[:contentH]
	}

	return bodyStyle.Width(width).Render(strings.Join(parts, "\n"))
}

func titleRender(item gh.Item) string {
	prefix := ""
	switch item.ContentType {
	case gh.ContentDraftIssue:
		prefix = "[draft] "
	case gh.ContentPullRequest:
		prefix = "[PR] "
	}
	if item.Number > 0 {
		return fmt.Sprintf("%s%s #%d", prefix, item.Title, item.Number)
	}
	return prefix + item.Title
}

func cardLabel(item gh.Item, width int) string {
	label := item.Title
	if label == "" {
		label = "(untitled)"
	}
	switch item.ContentType {
	case gh.ContentDraftIssue:
		label = "[d] " + label
	case gh.ContentPullRequest:
		label = "[p] " + label
	}
	return truncate(label, width)
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	for n := len(runes); n > 0; n-- {
		candidate := string(runes[:n]) + "…"
		if lipgloss.Width(candidate) <= width {
			return candidate
		}
	}
	return "…"
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}


