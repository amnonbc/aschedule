package htmltable

import (
	"fmt"
	"io"
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

func (t *Table) Render() {
	fmt.Fprint(t.w, "<table>\n")
	fmt.Fprint(t.w, "  <tr>\n")
	for _, h := range t.header {
		fmt.Fprintf(t.w, "    <th>%s</th>\n", h)
	}
	fmt.Fprint(t.w, "  </tr>\n")

	for _, r := range t.rows {
		fmt.Fprint(t.w, "  <tr>\n")
		for _, c := range r {
			fmt.Fprintf(t.w, "    <td>%s</td>\n", c)
		}
		fmt.Fprint(t.w, "  </tr>\n")

	}
	fmt.Fprint(t.w, "</table>\n")
}
