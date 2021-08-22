package rmime

import "io"

// Message is a message-part that can serve as a top-level e-mail
// message.
type Message Part

// Reader is an io.Reader and an io.ByteReader.
type Reader interface {
	io.Reader
	io.ByteReader
}

// ReadMessage reads a message from r.
func ReadMessage(r Reader) (*Message, error) {
	part, err := ReadPart(r, nil)
	return (*Message)(part), err
}
