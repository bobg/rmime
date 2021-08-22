package rmime

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"mime/quotedprintable"
	"strings"
)

// Part is a message part, consisting of a header and a body.
type Part struct {
	*Header
	B interface{} `json:"body"`
}

// ReadPart reads a message part from r after having read and parsed a
// header for it.
func ReadPart(r Reader, header *Header) (*Part, error) {
	defaultType := "text/plain"
	if header != nil && header.Type() == "multipart/digest" {
		defaultType = "message/rfc822"
	}
	innerHeader, err := ReadHeader(r, defaultType)
	if err != nil {
		return nil, err
	}
	body, err := ReadBody(r, innerHeader)
	if err != nil {
		return nil, err
	}
	return &Part{Header: innerHeader, B: body}, nil
}

// Body produces a reader over the decoded body. Transfer encoding is
// removed. Text is normalized to utf8 if possible, and CRLFs to LFs,
// and trailing blank lines are removed. Text/plain formatting per
// RFC3676 is also done. It is an error to call Body on non-leaf parts
// (multipart/*, message/*).
func (p *Part) Body() (io.Reader, error) {
	switch p.MajorType() {
	case "multipart", "message":
		return nil, fmt.Errorf("cannot call Body() on type %s", p.Type())
	}
	body, ok := p.B.(string)
	if !ok {
		return nil, fmt.Errorf("Content-Type is %s but body object is %T (want string)", p.Type(), p.B)
	}
	var r io.Reader
	r = strings.NewReader(body)
	switch p.Encoding() {
	case "quoted-printable":
		r = quotedprintable.NewReader(r)
	case "base64":
		r = base64.NewDecoder(base64.StdEncoding, r)
	}
	if p.MajorType() == "text" {
		var err error
		r, err = charsetReader(p.Charset(), r)
		if err != nil {
			return nil, err
		}
		if p.MinorType() == "plain" {
			params := p.Params()
			if params != nil {
				format := strings.ToLower(strings.TrimSpace(params["format"]))
				delsp := strings.ToLower(strings.TrimSpace(params["delsp"]))
				r = TextPlainReader(r, format == "flowed", delsp == "yes")
			}
		}
		s := bufio.NewScanner(r)
		pr, pw := io.Pipe()

		go func() {
			// Buffer blank lines, emit only when followed by a non-blank line.
			blankLines := 0
			for s.Scan() {
				line := s.Text()
				if line == "" {
					blankLines++
				} else {
					for blankLines > 0 {
						io.WriteString(pw, "\n")
						blankLines--
					}
					io.WriteString(pw, line)
					io.WriteString(pw, "\n")
				}
			}
			pw.Close()
		}()

		r = pr
	}
	return r, nil
}
