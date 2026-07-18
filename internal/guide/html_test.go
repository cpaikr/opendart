package guide

import (
	"strconv"
	"testing"

	"golang.org/x/net/html"
)

func TestExpandRowGroupClampsColumnSpan(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		span int
		want int
	}{
		{name: "normal", span: 2, want: 2},
		{name: "clamped", span: maximumColumnSpan * 1000, want: maximumColumnSpan},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cell := &html.Node{
				Type: html.ElementNode,
				Data: "td",
				Attr: []html.Attribute{{Key: "colspan", Val: strconv.Itoa(test.span)}},
			}
			cell.AppendChild(&html.Node{Type: html.TextNode, Data: "value"})
			row := &html.Node{Type: html.ElementNode, Data: "tr"}
			row.AppendChild(cell)

			grid := expandRowGroup([]*html.Node{row}, func(node *html.Node) string {
				return node.FirstChild.Data
			})
			if len(grid) != 1 || len(grid[0]) != test.want {
				t.Fatalf("grid dimensions = %d x %d, want 1 x %d", len(grid), len(grid[0]), test.want)
			}
		})
	}
}
