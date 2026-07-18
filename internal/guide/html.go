package guide

import (
	"errors"
	"strconv"

	"golang.org/x/net/html"
)

const (
	maximumColumnSpan         = 1000
	maximumExpandedTableCells = 10_000
)

var errGuideTableExpansionLimit = errors.New("guide table expansion exceeds the cell limit")

type pendingSpan struct {
	value     string
	remaining int
}

func expandRowGroup(rows []*html.Node, cellText func(*html.Node) string) ([][]string, error) {
	if len(rows) > maximumExpandedTableCells {
		return nil, errGuideTableExpansionLimit
	}
	pending := make(map[int]pendingSpan)
	grid := make([][]string, 0, len(rows))
	expandedCells := 0
	for rowIndex, rowNode := range rows {
		row := []string{}
		column := 0
		setExpandedCell := func(column int, value string) error {
			growth := column + 1 - len(row)
			if growth > 0 {
				if expandedCells > maximumExpandedTableCells-growth {
					return errGuideTableExpansionLimit
				}
				expandedCells += growth
			}
			row = setCell(row, column, value)
			return nil
		}
		consumeSpan := func() (bool, error) {
			span, ok := pending[column]
			if !ok {
				return false, nil
			}
			if err := setExpandedCell(column, span.value); err != nil {
				return false, err
			}
			span.remaining--
			if span.remaining == 0 {
				delete(pending, column)
			} else {
				pending[column] = span
			}
			column++
			return true, nil
		}
		for cell := rowNode.FirstChild; cell != nil; cell = cell.NextSibling {
			if cell.Type != html.ElementNode || (cell.Data != "th" && cell.Data != "td") {
				continue
			}
			for {
				consumed, err := consumeSpan()
				if err != nil {
					return nil, err
				}
				if !consumed {
					break
				}
			}
			if column >= maximumColumnSpan {
				return nil, errGuideTableExpansionLimit
			}
			value := cellText(cell)
			columnSpan := min(positiveAttribute(cell, "colspan"), maximumColumnSpan)
			if columnSpan > maximumColumnSpan-column {
				return nil, errGuideTableExpansionLimit
			}
			rowSpan := rowSpanAttribute(cell, len(rows)-rowIndex)
			for offset := 0; offset < columnSpan; offset++ {
				if err := setExpandedCell(column+offset, value); err != nil {
					return nil, err
				}
				if rowSpan > 1 {
					pending[column+offset] = pendingSpan{value: value, remaining: rowSpan - 1}
				}
			}
			column += columnSpan
		}
		for {
			consumed, err := consumeSpan()
			if err != nil {
				return nil, err
			}
			if consumed {
				continue
			}
			maxColumn := -1
			for pendingColumn := range pending {
				if pendingColumn > maxColumn {
					maxColumn = pendingColumn
				}
			}
			if column > maxColumn {
				break
			}
			column++
		}
		grid = append(grid, row)
	}
	return grid, nil
}

func rowSpanAttribute(node *html.Node, rowsRemaining int) int {
	value, err := strconv.Atoi(attribute(node, "rowspan"))
	if err != nil || value < 0 {
		return 1
	}
	if value == 0 || value > rowsRemaining {
		return rowsRemaining
	}
	return value
}

func setCell(row []string, column int, value string) []string {
	for len(row) <= column {
		row = append(row, "")
	}
	row[column] = value
	return row
}

func positiveAttribute(node *html.Node, name string) int {
	value, err := strconv.Atoi(attribute(node, name))
	if err != nil || value < 1 {
		return 1
	}
	return value
}

func attribute(node *html.Node, name string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == name {
			return attribute.Val
		}
	}
	return ""
}

func walk(node *html.Node, visit func(*html.Node)) {
	visit(node)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walk(child, visit)
	}
}
