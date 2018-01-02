package rmime

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMime(t *testing.T) {
	cases := []struct {
		inp     string
		wantErr error
	}{
		{inp: simpleMsg},
		{inp: multipartMsg},
	}
	for _, c := range cases {
		r := strings.NewReader(c.inp)
		m, err := ReadMessage(r)
		if err != c.wantErr {
			t.Errorf("got %s, want %s", err, c.wantErr)
		} else {
			j, _ := json.Marshal(m)
			t.Log(string(j))
		}
	}
}

const simpleMsg = `From: foo
To: bar

hello
`

const multipartMsg = `From: foo
To: bar
Content-Type: multipart/mixed; boundary="xyz"

preamble
--xyz
Content-Type: text/richtext

hello
--xyz

goodbye
--xyz--
postamble
`
