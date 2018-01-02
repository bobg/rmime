package rmime

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
)

// Multipart is the body of a multipart/* message. It contains a slice
// of *Part as its children.
type Multipart struct {
	Preamble, Postamble string
	Parts               []*Part
}

func readMultipart(r io.Reader, header *Header) (*Multipart, error) {
	f := header.findField("Content-Type")
	if f == nil {
		return nil, fmt.Errorf("no Content-Type field in multipart header")
	}
	_, params, err := mime.ParseMediaType(f.Value())
	if err != nil {
		return nil, err
	}
	boundary := params["boundary"]
	if boundary == "" {
		return nil, fmt.Errorf("no boundary parameter in multipart Content-Type field")
	}
	preamble, isFinal, err := readUntilBoundary(r, boundary)
	if err != nil {
		return nil, err
	}
	if isFinal {
		return nil, fmt.Errorf("final multipart boundary encountered before any others")
	}
	var parts []*Part
	for {
		content, isFinal, err := readUntilBoundary(r, boundary)
		if err != nil {
			return nil, err
		}
		inner := bytes.NewReader(content)
		part, err := ReadPart(inner, header)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
		if isFinal {
			postamble, err := ioutil.ReadAll(r)
			if err != nil {
				return nil, err
			}
			return &Multipart{Preamble: string(preamble), Postamble: string(postamble), Parts: parts}, nil
		}
	}
}

func readUntilBoundary(r io.Reader, boundary string) ([]byte, bool, error) {
	var result []byte
	for {
		line, err := readLine(r)
		if err != nil {
			return nil, false, err
		}
		if match, final := isBoundary(line, boundary); match {
			return result, final, nil
		}
		result = append(result, line...)
	}
}

func isBoundary(line []byte, boundary string) (match bool, final bool) {
	if len(line) < len(boundary)+2 {
		return false, false
	}
	if !bytes.Equal(line[:2], []byte("--")) {
		return false, false
	}
	if !bytes.Equal(line[2:2+len(boundary)], []byte(boundary)) {
		return false, false
	}
	rest := line[2+len(boundary):]
	if len(rest) >= 2 && bytes.Equal(rest[:2], []byte("--")) {
		final = true
		rest = rest[2:]
	}
	// allow only LWSP and \r?\n
	if rest[len(rest)-1] == '\n' {
		rest = rest[:len(rest)-1]
	}
	for _, r := range rest {
		switch r {
		case ' ', '\f', '\r', '\t', '\v':
			// ignore
		default:
			return false, false
		}
	}
	return true, final
}

// Requires and includes a terminating newline.
func readLine(r io.Reader) ([]byte, error) {
	var result []byte
	for {
		var b [1]byte
		_, err := io.ReadFull(r, b[:])
		if err != nil {
			return nil, err
		}
		result = append(result, b[0])
		if b[0] == '\n' {
			return result, nil
		}
	}
}
