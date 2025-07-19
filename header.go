package rmime

import (
	"bytes"
	"io"
	"mime"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/bobg/errors"
)

// Header is a message header or message-part header.
// It consists of a sequence of fields
// and a default content-type dictated by context:
// text/plain by default,
// message/rfc822 for the children of a multipart/digest.
type Header struct {
	Fields      []*Field `json:"fields"`
	DefaultType string   `json:"default_type"`
}

// Address is the type of an e-mail address.
type Address struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address,omitempty"`
}

// ErrHeaderSyntax is the error indicating bad mail header syntax.
var ErrHeaderSyntax = errors.New("bad syntax in header")

// ReadHeader reads a message header or message-part header from r,
// which must be positioned at the start of the header.
// The defaultType parameter sets the DefaultType field of the resulting Header.
// Pass "" to get the default defaultType of "text/plain".
func ReadHeader(r io.ByteReader, defaultType string) (*Header, error) {
	if defaultType == "" {
		defaultType = "text/plain"
	}
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
				return nil, errors.Wrapf(ErrHeaderSyntax, "unexpected continuation line")
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
	if err != nil && !errors.Is(err, mime.ErrInvalidMediaParameter) {
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

// Params parses the key=value pairs in the Content-Type field, if any.
// The return value may be nil.
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

// Disposition parses the Content-Disposition field.
// The first return value is normally "inline" or "attachment".
// The second is a map of associated key=value pairs, and may be nil.
// If there is no Content-Disposition field, this function returns "inline", nil.
func (h Header) Disposition() (string, map[string]string) {
	f := h.findField("Content-Disposition")
	if f == nil {
		return "inline", nil
	}
	tok, params, err := mime.ParseMediaType(f.Value())
	if err != nil {
		return "inline", nil
	}
	return tok, params
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

var trailingParenTextRegex = regexp.MustCompile(`\s*\([^()]*\)\s*$`)

// Time returns the parsed time of h, or the zero time if absent or
// unparseable.
func (h Header) Time() time.Time {
	f := h.findField("Date") // xxx Resent-Date?
	if f == nil {
		return time.Time{}
	}
	v := f.Value()

	t, err := mail.ParseDate(v)
	if err == nil {
		return t
	}

	// Try again, removing any trailing parenthesized string like " (Pacific Daylight Time)".
	v = trailingParenTextRegex.ReplaceAllString(v, "")
	t, err = mail.ParseDate(v)
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
func readHeaderFieldLine(r io.ByteReader) ([]byte, error) {
	var (
		result    []byte
		pendingCR bool
	)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == '\n' {
			return result, nil
		}
		if pendingCR {
			result = append(result, '\r')
			pendingCR = false
		}
		if b == '\r' {
			pendingCR = true
		} else {
			result = append(result, b)
		}
	}
}

func isContinuationLine(line []byte) bool {
	return len(line) > 0 && (line[0] == ' ' || line[0] == '\t')
}

// This regex is a cheat.
// Message IDs are allowed to contain interior angle-brackets (via quoting),
// but if any are encountered in the wild this software is surely not the only that will break.
var msgIDRegex = regexp.MustCompile(`<([^<>]+@[^<>]+)>`)

// MessageID returns the parsed contents of the Message-Id field
// or "" if not found or parseable.
//
// Per RFC2822, "the angle bracket characters are not part of the
// msg-id; the msg-id is what is contained between the two angle
// bracket characters."
func (h Header) MessageID() string {
	f := h.findField("Message-Id")
	if f == nil {
		return ""
	}
	if m := msgIDRegex.FindStringSubmatch(f.Value()); len(m) > 0 {
		return m[1]
	}
	return ""
}

// InReplyTo returns the list of message-ids in the In-Reply-To field(s).
// The message ids are parsed as in Header.MessageID.
func (h Header) InReplyTo() []string {
	return h.messageIDs("In-Reply-To")
}

// References returns the list of message-ids in the References field(s).
// The message ids are parsed as in Header.MessageID.
func (h Header) References() []string {
	return h.messageIDs("References")
}

func (h Header) messageIDs(fieldName string) []string {
	f := h.findField(fieldName)
	if f == nil {
		return nil
	}

	var result []string
	matches := msgIDRegex.FindAllStringSubmatch(f.Value(), -1)
	for _, m := range matches {
		result = append(result, m[1])
	}
	return result
}
