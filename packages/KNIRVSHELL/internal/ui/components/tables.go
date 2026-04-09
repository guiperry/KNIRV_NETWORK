package components

import (
	"strings"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/ui"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// Table is a component that displays a table
type Table struct {
	table  table.Model
	styles ui.Styles
	width  int
	height int
}

// NewTable creates a new table
func NewTable(styles ui.Styles, columns []table.Column, rows []table.Row, width, height int) Table {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
		table.WithWidth(width),
	)

	// Set styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(styles.TableBorder.GetBorderStyle()).
		BorderForeground(styles.TableBorder.GetForeground()).
		BorderBottom(true).
		Bold(true).
		Foreground(styles.TableHeader.GetForeground())
	s.Selected = s.Selected.
		Foreground(styles.ListItemSelected.GetForeground()).
		Background(styles.ListItemSelected.GetBackground()).
		Bold(true)
	t.SetStyles(s)

	return Table{
		table:  t,
		styles: styles,
		width:  width,
		height: height,
	}
}

// SetRows sets the table rows
func (t *Table) SetRows(rows []table.Row) {
	t.table.SetRows(rows)
}

// SetColumns sets the table columns
func (t *Table) SetColumns(columns []table.Column) {
	t.table.SetColumns(columns)
}

// SetWidth sets the table width
func (t *Table) SetWidth(width int) {
	t.width = width
	t.table.SetWidth(width)
}

// SetHeight sets the table height
func (t *Table) SetHeight(height int) {
	t.height = height
	t.table.SetHeight(height)
}

// SelectedRow returns the selected row
func (t *Table) SelectedRow() table.Row {
	return t.table.SelectedRow()
}

// Cursor returns the cursor position
func (t *Table) Cursor() int {
	return t.table.Cursor()
}

// SetCursor sets the cursor position
func (t *Table) SetCursor(cursor int) {
	t.table.SetCursor(cursor)
}

// Update handles user input
func (t *Table) Update(msg tea.Msg) (Table, tea.Cmd) {
	var cmd tea.Cmd
	t.table, cmd = t.table.Update(msg)
	return *t, cmd
}

// View renders the table
func (t Table) View() string {
	return t.table.View()
}

// SimpleTable is a component that displays a simple table without selection
type SimpleTable struct {
	headers []string
	rows    [][]string
	styles  ui.Styles
	width   int
}

// NewSimpleTable creates a new simple table
func NewSimpleTable(styles ui.Styles, headers []string, width int) SimpleTable {
	return SimpleTable{
		headers: headers,
		rows:    [][]string{},
		styles:  styles,
		width:   width,
	}
}

// AddRow adds a row to the table
func (t *SimpleTable) AddRow(row []string) {
	t.rows = append(t.rows, row)
}

// SetRows sets the table rows
func (t *SimpleTable) SetRows(rows [][]string) {
	t.rows = rows
}

// SetWidth sets the table width
func (t *SimpleTable) SetWidth(width int) {
	t.width = width
}

// View renders the simple table
func (t SimpleTable) View() string {
	if len(t.headers) == 0 {
		return ""
	}

	// Calculate column widths
	colCount := len(t.headers)
	colWidths := make([]int, colCount)

	// Start with header widths
	for i, header := range t.headers {
		colWidths[i] = len(header)
	}

	// Check row widths
	for _, row := range t.rows {
		for i, cell := range row {
			if i < colCount && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Adjust column widths to fit the table width
	totalWidth := 0
	for _, width := range colWidths {
		totalWidth += width
	}

	// Add spacing and borders
	totalWidth += (colCount * 3) + 1

	// If total width exceeds table width, adjust column widths
	if totalWidth > t.width && colCount > 0 {
		excess := totalWidth - t.width
		reduction := excess / colCount
		remainder := excess % colCount

		for i := range colWidths {
			colWidths[i] -= reduction
			if i < remainder {
				colWidths[i]--
			}
			if colWidths[i] < 3 {
				colWidths[i] = 3
			}
		}
	}

	// Build the table
	var sb strings.Builder

	// Build the header
	sb.WriteString("┌")
	for i, width := range colWidths {
		sb.WriteString(strings.Repeat("─", width+2))
		if i < colCount-1 {
			sb.WriteString("┬")
		}
	}
	sb.WriteString("┐\n")

	// Header row
	sb.WriteString("│")
	for i, header := range t.headers {
		if i < colCount {
			cell := header
			if len(cell) > colWidths[i] {
				cell = cell[:colWidths[i]-3] + "..."
			}
			sb.WriteString(" " + t.styles.TableHeader.Render(
				cell+strings.Repeat(" ", colWidths[i]-len(cell)),
			) + " ")
			if i < colCount-1 {
				sb.WriteString("│")
			}
		}
	}
	sb.WriteString("│\n")

	// Header/body separator
	sb.WriteString("├")
	for i, width := range colWidths {
		sb.WriteString(strings.Repeat("─", width+2))
		if i < colCount-1 {
			sb.WriteString("┼")
		}
	}
	sb.WriteString("┤\n")

	// Body rows
	for rowIdx, row := range t.rows {
		sb.WriteString("│")
		for i := 0; i < colCount; i++ {
			var cell string
			if i < len(row) {
				cell = row[i]
			} else {
				cell = ""
			}

			if len(cell) > colWidths[i] {
				cell = cell[:colWidths[i]-3] + "..."
			}

			style := t.styles.TableCell
			if rowIdx%2 == 1 {
				style = style.Copy().Background(t.styles.Theme.HighlightLow)
			}

			sb.WriteString(" " + style.Render(
				cell+strings.Repeat(" ", colWidths[i]-len(cell)),
			) + " ")

			if i < colCount-1 {
				sb.WriteString("│")
			}
		}
		sb.WriteString("│\n")
	}

	// Bottom border
	sb.WriteString("└")
	for i, width := range colWidths {
		sb.WriteString(strings.Repeat("─", width+2))
		if i < colCount-1 {
			sb.WriteString("┴")
		}
	}
	sb.WriteString("┘")

	return sb.String()
}
