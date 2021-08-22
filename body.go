package rmime

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

// ErrUnimplemented is the error indicating an unimplemented feature.
var ErrUnimplemented = errors.New("unimplemented")

// DeliveryStatus is the type of a parsed message/delivery-status body part.
// TODO: further parse message/delivery-status bodies
// (parse out specific fields).
type DeliveryStatus struct {
	Message    *Header   `json:"message"`
	Recipients []*Header `json:"recipients"`
}

// ReadBody reads a message body from r after the header has been read
// and parsed. The type of the result depends on the content-type
// specified in the header:
//   message/rfc822:          *Message
//   message/delivery-status: *DeliveryStatus
//   multipart/*:             *Multipart
//   */*:                     string
func ReadBody(r Reader, header *Header) (interface{}, error) {
	switch header.MajorType() {
	case "message":
		switch header.MinorType() {
		case "rfc822", "news": // message/news == message/rfc822 per RFC5537
			return ReadMessage(r)

		case "external-body":
			return nil, ErrUnimplemented

		case "partial":
			return nil, ErrUnimplemented

		case "delivery-status":
			// A message-level set of header fields, followed by one or more
			// per-recipient sets of header fields (see RFC 3464).
			perMessage, err := ReadHeader(r, "text/plain")
			if err != nil {
				return nil, err
			}
			var perRecipient []*Header
			for {
				h, err := ReadHeader(r, "text/plain")
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return nil, err
				}
				if len(h.Fields) == 0 {
					break
				}
				perRecipient = append(perRecipient, h)
			}
			return &DeliveryStatus{perMessage, perRecipient}, nil

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
