package rmime

import (
	"bufio"
	"io"
)

// reader is an io.Reader and an io.ByteReader.
type reader interface {
	io.Reader
	io.ByteReader
}

func ensureReader(r io.Reader) reader {
	if rr, ok := r.(reader); ok {
		return rr
	}
	return bufio.NewReader(r)
}
