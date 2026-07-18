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
			grid, err := expandRowGroup([]*html.Node{expandedTestRow(test.spans...)}, expandedTestCellText)
			if err != nil {
				t.Fatal(err)
			}
			if len(grid) != 1 || len(grid[0]) != test.want {
				t.Fatalf("grid dimensions = %d x %d, want 1 x %d", len(grid), len(grid[0]), test.want)
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
		if _, err := expandRowGroup([]*html.Node{expandedTestRow(spans...)}, expandedTestCellText); !errors.Is(err, errGuideTableExpansionLimit) {
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
	grid, err := expandRowGroup(rows, expandedTestCellText)
	if err != nil {
		t.Fatal(err)
	}
	if len(grid) != len(rows) {
		t.Fatalf("expanded rows = %d, want %d", len(grid), len(rows))
	}
	rows = append(rows, expandedTestRow(1))
	if _, err := expandRowGroup(rows, expandedTestCellText); !errors.Is(err, errGuideTableExpansionLimit) {
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
