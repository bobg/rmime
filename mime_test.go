package rmime

import (
	"bytes"
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
			t.Errorf("got error %v, want %v", err, c.wantErr)
		} else {
			j, _ := json.Marshal(m)
			t.Log(string(j))
		}

		buf := new(bytes.Buffer)
		_, err = m.WriteTo(buf)
		if err != nil {
			t.Fatal(err)
		}
		got := buf.String()
		if got != c.inp {
			t.Errorf("message re-rendering mismatch, got:\n%s\n\nwant:\n%s", got, c.inp)
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
