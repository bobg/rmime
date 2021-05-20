package rmime

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestMime(t *testing.T) {
	cases := []struct {
		inp           string
		wantMsgID     string
		wantInReplyTo []string
		wantErr       error
	}{
		{
			inp:       simpleMsg,
			wantMsgID: "a@b",
			wantInReplyTo: []string{
				"c@d",
				"e@f",
			},
		},
		{inp: multipartMsg},
	}
	for _, c := range cases {
		r := strings.NewReader(c.inp)
		m, err := ReadMessage(r)
		if !errors.Is(err, c.wantErr) {
			t.Errorf("got error %v, want %v", err, c.wantErr)
		} else {
			j, _ := json.Marshal(m)
			t.Log(string(j))
		}

		if c.wantErr != nil {
			continue
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

		got = m.MessageID()
		if got != c.wantMsgID {
			t.Errorf("got MessageID <%s>, want <%s>", got, c.wantMsgID)
		}

		gotIRT := m.InReplyTo()
		if !reflect.DeepEqual(gotIRT, c.wantInReplyTo) {
			t.Errorf("got InReplyTo %v, want %v", gotIRT, c.wantInReplyTo)
		}
	}
}

const simpleMsg = `From: foo
To: bar
Message-Id: <a@b>
In-Reply-To: <c@d> <e@f>

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
