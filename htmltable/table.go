package htmltable

// htmltable generates the html tables using the same interface as
// github.com/olekukonko/tablewriter.

import (
	"fmt"
	"io"
	"strings"
)

type Table struct {
	w      io.Writer
	header []string
	rows   [][]string
}

func NewWriter(w io.Writer) *Table {
	return &Table{w: w}
}

func (t *Table) SetHeader(h []string) {
	t.header = h
}

func (t *Table) Append(row []string) {
	t.rows = append(t.rows, row)
}

func printTag(w io.Writer, indent int, tag string, txt string) {
	fmt.Fprintf(w, "%s<%s>%s</%s>\n", strings.Repeat(" ", indent), tag, txt, tag)
}

func (t *Table) Render() {
	fmt.Fprint(t.w, "<table>\n")
	fmt.Fprint(t.w, "  <tr>\n")
	for _, h := range t.header {
		printTag(t.w, 4, "th", h)
	}
	fmt.Fprint(t.w, "  </tr>\n")

	for _, r := range t.rows {
		fmt.Fprint(t.w, "  <tr>\n")
		for _, c := range r {
			printTag(t.w, 4, "td", c)
		}
		fmt.Fprint(t.w, "  </tr>\n")

	}
	fmt.Fprint(t.w, "</table>\n")
}
