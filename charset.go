package rmime

import (
	"io"
	"strings"

	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/ianaindex"
)

func charsetReader(label string, inp io.Reader) (io.Reader, error) {
	label = strings.ToLower(strings.TrimSpace(label))
	enc, err := ianaindex.MIME.Encoding(label)
	if err != nil {
		enc, err = ianaindex.IANA.Encoding(label)
		if err != nil {
			enc, err = htmlindex.Get(label)
			if err != nil {
				return nil, wrapf(err, "charset %s", label)
			}
		}
	}
	if enc == nil {
		return inp, nil
	}
	d := enc.NewDecoder()
	return d.Reader(inp), nil
}
