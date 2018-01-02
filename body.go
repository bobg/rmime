package rmime

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

var ErrUnimplemented = errors.New("unimplemented")

// ReadBody reads a message body from r after the header has been read
// and parsed. The type of the result depends on the content-type
// specified in the header:
//   message/rfc822: *Message
//   multipart/*:    *Multipart
//   */*:            string
func ReadBody(r io.Reader, header *Header) (interface{}, error) {
	switch header.MajorType() {
	case "message":
		switch header.MinorType() {
		case "rfc822":
			return ReadMessage(r)

		case "external-body":
			return nil, ErrUnimplemented

		case "partial":
			return nil, ErrUnimplemented

		case "delivery-status":
			return nil, ErrUnimplemented

		default:
			return nil, fmt.Errorf("unknown message subtype %s", header.MinorType())
		}

	case "multipart":
		return readMultipart(r, header)
	}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}
