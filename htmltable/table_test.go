package htmltable

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const expected = `
<table>
  <tr>
	<th>aa</th>
	<th>bb</th>
  </tr>
  <tr>
	<td>1</td>
	<td>2</td>
  </tr>
  <tr>
	<td>3</td>
	<td>4</td>
  </tr>
</table>
`

func strip(s string) string {
	o := strings.ReplaceAll(s, " ", "")
	o = strings.ReplaceAll(o, "\t", "")
	return strings.TrimSpace(o)
}

func TestTable_Render(t *testing.T) {
	out := new(bytes.Buffer)

	tb := NewWriter(out)
	tb.SetHeader([]string{"aa", "bb"})
	tb.Append([]string{"1", "2"})
	tb.Append([]string{"3", "4"})
	tb.Render()
	out2 := strip(out.String())
	expected2 := strip(expected)
	assert.Equal(t, expected2, out2)
}
