package guide

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// CompatibilityExtraction captures only the HTML primitives that the guide
// port relies on. The production guide model is introduced in ordered work 2.
type CompatibilityExtraction struct {
	Heading string
	Tables  map[string][][]string
	Hidden  map[string]string
}

func ExtractCompatibilityFixture(input io.Reader) (CompatibilityExtraction, error) {
	root, err := html.Parse(input)
	if err != nil {
		return CompatibilityExtraction{}, fmt.Errorf("parse guide fixture: %w", err)
	}
	result := CompatibilityExtraction{
		Tables: make(map[string][][]string),
		Hidden: make(map[string]string),
	}
	walk(root, func(node *html.Node) {
		switch node.Type {
		case html.ElementNode:
			switch node.Data {
			case "h1":
				if result.Heading == "" {
					result.Heading = normalizedText(node)
				}
			case "table":
				caption := firstDescendant(node, "caption")
				if caption != nil {
					result.Tables[normalizedText(caption)] = expandedTable(node)
				}
			case "input":
				if attribute(node, "type") == "hidden" && attribute(node, "name") != "" {
					result.Hidden[attribute(node, "name")] = attribute(node, "value")
				}
			}
		}
	})
	if result.Heading == "" {
		return CompatibilityExtraction{}, fmt.Errorf("guide fixture has no heading")
	}
	return result, nil
}

type pendingSpan struct {
	value     string
	remaining int
}

func expandedTable(table *html.Node) [][]string {
	var rows []*html.Node
	walk(table, func(node *html.Node) {
		if node != table && node.Type == html.ElementNode && node.Data == "tr" {
			rows = append(rows, node)
		}
	})
	pending := make(map[int]pendingSpan)
	grid := make([][]string, 0, len(rows))
	for _, rowNode := range rows {
		row := []string{}
		column := 0
		consumeSpan := func() bool {
			span, ok := pending[column]
			if !ok {
				return false
			}
			row = setCell(row, column, span.value)
			span.remaining--
			if span.remaining == 0 {
				delete(pending, column)
			} else {
				pending[column] = span
			}
			column++
			return true
		}
		for cell := rowNode.FirstChild; cell != nil; cell = cell.NextSibling {
			if cell.Type != html.ElementNode || (cell.Data != "th" && cell.Data != "td") {
				continue
			}
			for consumeSpan() {
			}
			value := normalizedText(cell)
			columnSpan := positiveAttribute(cell, "colspan")
			rowSpan := positiveAttribute(cell, "rowspan")
			for offset := 0; offset < columnSpan; offset++ {
				row = setCell(row, column+offset, value)
				if rowSpan > 1 {
					pending[column+offset] = pendingSpan{value: value, remaining: rowSpan - 1}
				}
			}
			column += columnSpan
		}
		for {
			if consumeSpan() {
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
	return grid
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

func normalizedText(node *html.Node) string {
	var builder strings.Builder
	var visit func(*html.Node)
	visit = func(current *html.Node) {
		if current.Type == html.TextNode {
			builder.WriteString(current.Data)
			builder.WriteByte(' ')
		}
		if current.Type == html.ElementNode && current.Data == "br" {
			builder.WriteByte('\n')
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(node)
	lines := strings.Split(strings.ReplaceAll(builder.String(), "\u00a0", " "), "\n")
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		if fields := strings.Fields(line); len(fields) > 0 {
			normalized = append(normalized, strings.Join(fields, " "))
		}
	}
	return strings.Join(normalized, "\n")
}

func firstDescendant(root *html.Node, element string) *html.Node {
	var result *html.Node
	walk(root, func(node *html.Node) {
		if result == nil && node != root && node.Type == html.ElementNode && node.Data == element {
			result = node
		}
	})
	return result
}

func walk(node *html.Node, visit func(*html.Node)) {
	visit(node)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walk(child, visit)
	}
}
