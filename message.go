package rmime

import "io"

// Message is a message-part that can serve as a top-level e-mail
// message.
type Message Part

// ReadMessage reads a message from r.
func ReadMessage(r io.Reader) (*Message, error) {
	part, err := ReadPart(r, nil)
	return (*Message)(part), err
}
