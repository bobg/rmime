package rmime

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"net/mail"
	"strings"
	"time"
)

// Header is a message or message-part header. It consists of a
// sequence of fields and a default content-type dictated by context:
// text/plain by default, message/rfc822 for child parts of a
// multipart/digest.
type Header struct {
	Fields      []*Field `json:"fields"`
	DefaultType string   `json:"default_type"`
}

type Address struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address,omitempty"`
}

// ErrHeaderSyntax is the error indicating bad mail header syntax.
var ErrHeaderSyntax = errors.New("bad syntax in header")

// ReadHeader reads a message or message-part header from r.
func ReadHeader(r io.Reader, defaultType string) (*Header, error) {
	result := &Header{DefaultType: defaultType}
	var latestField *Field
	for {
		line, err := readHeaderFieldLine(r)
		if err != nil {
			return nil, err
		}
		if len(line) == 0 {
			return result, nil
		}
		if isContinuationLine(line) {
			if latestField == nil {
				return nil, wrapf(ErrHeaderSyntax, "unexpected continuation line")
			}
			latestField.V = append(latestField.V, string(line))
			continue
		}
		split := bytes.SplitN(line, []byte{':'}, 2)
		if len(split) != 2 {
			return nil, ErrHeaderSyntax
		}
		// xxx check that split[0] is a legal field name
		latestField = &Field{N: string(split[0]), V: []string{string(split[1])}}
		result.Fields = append(result.Fields, latestField)
	}
}

// Type returns the content-type indicated by h in canonical form.
func (h Header) Type() string {
	f := h.findField("Content-Type")
	if f == nil {
		return h.DefaultType
	}
	t, _, err := mime.ParseMediaType(f.Value())
	if err != nil && err != mime.ErrInvalidMediaParameter {
		return h.DefaultType
	}
	if !strings.Contains(t, "/") {
		return h.DefaultType
	}
	return t
}

// MajorType returns the major part of the content type of h in
// canonical form.
func (h Header) MajorType() string {
	t := strings.SplitN(h.Type(), "/", 2)
	return t[0]
}

// MinorType returns the minor part of the content type of h in
// canonical form.
func (h Header) MinorType() string {
	t := strings.SplitN(h.Type(), "/", 2)
	return t[1]
}

func (h Header) Params() map[string]string {
	f := h.findField("Content-Type")
	if f == nil {
		return nil
	}
	_, params, err := mime.ParseMediaType(f.Value())
	if err != nil {
		return nil
	}
	return params
}

// Charset returns the character set indicated by h (when h has
// content-type text/*).
func (h Header) Charset() string {
	f := h.findField("Content-Type")
	if f == nil {
		return "us-ascii"
	}
	_, params, err := mime.ParseMediaType(f.Value())
	if err != nil {
		return "us-ascii"
	}
	if c, ok := params["charset"]; ok {
		return c
	}
	return "us-ascii"
}

// Subject returns the decoded subject text of h.
func (h Header) Subject() string {
	f := h.findField("Subject")
	if f == nil {
		return ""
	}
	dec := mime.WordDecoder{
		CharsetReader: charsetReader,
	}
	res, err := dec.DecodeHeader(f.Value())
	if err != nil {
		return ""
	}
	return res
}

// Time returns the parsed time of h, or the zero time if absent or
// unparseable.
func (h Header) Time() time.Time {
	f := h.findField("Date") // xxx Resent-Date?
	if f == nil {
		return time.Time{}
	}
	t, err := mail.ParseDate(f.Value())
	if err != nil {
		return time.Time{}
	}
	return t
}

// Encoding returns the content-transfer-encoding indicated by h.
func (h Header) Encoding() string {
	f := h.findField("Content-Transfer-Encoding")
	if f == nil {
		return "7bit"
	}
	return f.Value()
}

// Sender returns the parsed sender address.
func (h Header) Sender() *Address {
	f := h.findField("From") // xxx Resent-From? Sender?
	if f == nil {
		return nil
	}
	a, err := mail.ParseAddress(f.Value())
	if err != nil {
		return nil
	}
	return &Address{Name: a.Name, Address: a.Address}
}

var recipientFields = []string{"To", "Cc", "Bcc"} // xxx Resent-To, Resent-Cc, Resent-Bcc?

// Recipients returns the parsed recipient addresses.
func (h Header) Recipients() []*Address {
	var res []*Address
	for _, name := range recipientFields {
		f := h.findField(name)
		if f == nil {
			continue
		}
		as, err := mail.ParseAddressList(f.Value())
		if err != nil {
			continue
		}
		for _, a := range as {
			res = append(res, &Address{Name: a.Name, Address: a.Address})
		}
	}
	return res
}

func (h Header) findField(name string) *Field {
	for i := len(h.Fields) - 1; i >= 0; i-- {
		if h.Fields[i].Name() == name {
			return h.Fields[i]
		}
	}
	return nil
}

// Requires and removes a terminating \n (and an optional preceding
// \r).
func readHeaderFieldLine(r io.Reader) ([]byte, error) {
	var (
		result    []byte
		pendingCR bool
	)
	for {
		var b [1]byte
		_, err := io.ReadFull(r, b[:])
		if err != nil {
			return nil, err
		}
		if b[0] == '\n' {
			return result, nil
		}
		if pendingCR {
			result = append(result, '\r')
			pendingCR = false
		}
		if b[0] == '\r' {
			pendingCR = true
		} else {
			result = append(result, b[0])
		}
	}
}

func isContinuationLine(line []byte) bool {
	return len(line) > 0 && (line[0] == ' ' || line[0] == '\t')
}
