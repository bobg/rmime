package rmime

import (
	"fmt"
	"io"

	"github.com/bobg/errors"
)

// WriteTo implements the io.WriterTo interface.
func (m *Message) WriteTo(w io.Writer) (int64, error) {
	return (*Part)(m).WriteTo(w)
}

// WriteTo implements the io.WriterTo interface.
func (p *Part) WriteTo(w io.Writer) (int64, error) {
	n, err := p.Header.WriteTo(w)
	if err != nil {
		return n, errors.Wrap(err, "writing part header")
	}

	switch p.MajorType() {
	case "multipart":
		params := p.Params()
		boundary := params["boundary"]
		if boundary == "" {
			boundary = "x"
		}

		body := p.B.(*Multipart)
		n2, err := fmt.Fprint(w, body.Preamble) // note, this assumes Preamble ends in a newline
		n += int64(n2)
		if err != nil {
			return n, errors.Wrap(err, "writing multipart body preamble")
		}

		for _, subpart := range body.Parts {
			n2, err = fmt.Fprintf(w, "--%s\n", boundary)
			n += int64(n2)
			if err != nil {
				return n, errors.Wrap(err, "writing multipart body boundary")
			}

			n3, err := subpart.WriteTo(w)
			n += n3
			if err != nil {
				return n, errors.Wrap(err, "writing multipart subpart")
			}
		}
		n2, err = fmt.Fprintf(w, "--%s--\n", boundary)
		n += int64(n2)
		if err != nil {
			return n, errors.Wrap(err, "writing multipart body closing boundary")
		}
		n2, err = fmt.Fprint(w, body.Postamble)
		n += int64(n2)
		return n, errors.Wrap(err, "writing multipart body postamble")

	case "message":
		switch p.MinorType() {
		case "rfc822", "news": // message/news == message/rfc822 per RFC5537
			body := p.B.(*Message)
			n2, err := body.WriteTo(w)
			n += n2
			return n, errors.Wrap(err, "writing message/rfc822 body")

		case "delivery-status":
			body, ok := p.B.(*DeliveryStatus)
			if !ok {
				return n, fmt.Errorf("unexpected body type %T for message/delivery-status part", p.B)
			}
			n2, err := body.WriteTo(w)
			n += n2
			return n, errors.Wrap(err, "writing message/delivery-status body")

		default: // xxx
			return n, errors.Wrapf(ErrUnimplemented, "writing message/%s part", p.MinorType())
		}

	default:
		body, ok := p.B.(string)
		if !ok {
			return n, fmt.Errorf("unexpected body type %T for non-multipart, non-message part", p.B)
		}
		n2, err := fmt.Fprint(w, body)
		n += int64(n2)
		return n, errors.Wrap(err, "writing body of non-multipart, non-message part")
	}
}

// WriteTo implements the io.WriterTo interface.
func (h Header) WriteTo(w io.Writer) (int64, error) {
	var n int64
	for _, f := range h.Fields {
		n1, err := f.WriteTo(w)
		n += n1
		if err != nil {
			return n, errors.Wrap(err, "writing header field")
		}
	}
	n2, err := fmt.Fprint(w, "\n")
	n += int64(n2)
	return n, errors.Wrap(err, "writing final newline after header")
}

// WriteTo implements the io.WriterTo interface.
func (f Field) WriteTo(w io.Writer) (int64, error) {
	var n int64
	for _, v := range f.V {
		n2, err := fmt.Fprintf(w, "%s:%s\n", f.N, v)
		n += int64(n2)
		if err != nil {
			return n, errors.Wrapf(err, "writing field %q", f.N)
		}
	}
	return n, nil
}

// WriteTo implements io.WriterTo.
func (ds *DeliveryStatus) WriteTo(w io.Writer) (int64, error) {
	n, err := ds.Message.WriteTo(w)
	if err != nil {
		return n, errors.Wrap(err, "writing per-message header fields")
	}
	for _, recip := range ds.Recipients {
		n2, err := recip.WriteTo(w)
		n += n2
		if err != nil {
			return n, errors.Wrap(err, "writing per-recipient header fields")
		}
	}
	return n, nil
}
