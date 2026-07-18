package guide

import (
	"errors"
	"strconv"
	"testing"

	"golang.org/x/net/html"
)

func TestExpandRowGroupPreservesValidColumnSpans(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		spans []int
		want  int
	}{
		{name: "normal", spans: []int{2}, want: 2},
		{name: "maximum", spans: []int{maximumColumnSpan}, want: maximumColumnSpan},
		{name: "attribute clamped", spans: []int{maximumColumnSpan + 1}, want: maximumColumnSpan},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			grid, cells, err := expandRowGroup([]*html.Node{expandedTestRow(test.spans...)}, expandedTestCellText, maximumExpandedTableCells)
			if err != nil {
				t.Fatal(err)
			}
			if len(grid) != 1 || len(grid[0]) != test.want || cells != test.want {
				t.Fatalf("grid dimensions/cells = %d x %d / %d, want 1 x %d / %d", len(grid), len(grid[0]), cells, test.want, test.want)
			}
		})
	}
}

func TestExpandRowGroupRejectsExcessiveRowWidth(t *testing.T) {
	t.Parallel()
	for _, spans := range [][]int{
		{maximumColumnSpan, 1},
		{maximumColumnSpan - 1, 2},
	} {
		if _, _, err := expandRowGroup([]*html.Node{expandedTestRow(spans...)}, expandedTestCellText, maximumExpandedTableCells); !errors.Is(err, errGuideTableExpansionLimit) {
			t.Fatalf("expandRowGroup(%v) error = %v", spans, err)
		}
	}
}

func TestExpandRowGroupEnforcesAggregateTableBudget(t *testing.T) {
	t.Parallel()
	rows := make([]*html.Node, maximumExpandedTableCells/maximumColumnSpan)
	for index := range rows {
		rows[index] = expandedTestRow(maximumColumnSpan)
	}
	grid, cells, err := expandRowGroup(rows, expandedTestCellText, maximumExpandedTableCells)
	if err != nil {
		t.Fatal(err)
	}
	if len(grid) != len(rows) || cells != maximumExpandedTableCells {
		t.Fatalf("expanded rows/cells = %d / %d, want %d / %d", len(grid), cells, len(rows), maximumExpandedTableCells)
	}
	rows = append(rows, expandedTestRow(1))
	if _, _, err := expandRowGroup(rows, expandedTestCellText, maximumExpandedTableCells); !errors.Is(err, errGuideTableExpansionLimit) {
		t.Fatalf("over-budget expansion error = %v", err)
	}
}

func expandedTestRow(spans ...int) *html.Node {
	row := &html.Node{Type: html.ElementNode, Data: "tr"}
	for _, span := range spans {
		cell := &html.Node{
			Type: html.ElementNode,
			Data: "td",
			Attr: []html.Attribute{{Key: "colspan", Val: strconv.Itoa(span)}},
		}
		cell.AppendChild(&html.Node{Type: html.TextNode, Data: "value"})
		row.AppendChild(cell)
	}
	return row
}

func expandedTestCellText(node *html.Node) string { return node.FirstChild.Data }
