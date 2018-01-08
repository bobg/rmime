package rmime

import (
	"io"

	"golang.org/x/text/encoding/ianaindex"
)

// Map from common non-canonical charset names to canonical ones.
var canonicalCharsets = map[string]string{
	"ascii": "us-ascii",
}

func charsetReader(label string, inp io.Reader) (io.Reader, error) {
	if l, ok := canonicalCharsets[label]; ok {
		label = l
	}
	enc, err := ianaindex.MIME.Encoding(label)
	if err != nil {
		enc, err = ianaindex.IANA.Encoding(label)
		if err != nil {
			return nil, wrapf(err, "charset %s", label)
		}
	}
	if enc == nil {
		return inp, nil
	}
	d := enc.NewDecoder()
	return d.Reader(inp), nil
}
