package rmime

import (
	"io"
)

// WriteTo implements the io.WriterTo interface.
func (m *Message) WriteTo(w io.Writer) (int64, error) {
	return (*Part)(m).WriteTo(w)
}

// WriteTo implements the io.WriterTo interface.
func (p *Part) WriteTo(w io.Writer) (int64, error) {
	n, err := p.Header.WriteTo(w)
	if err != nil {
		return n, err
	}

	switch p.MajorType() {
	case "multipart":
		params := p.Params()
		boundary := params["boundary"]
		if boundary == "" {
			boundary = "x"
		}

		body := p.B.(*Multipart)
		n2, err := w.Write([]byte(body.Preamble)) // note, this assumes Preamble ends in a newline
		n += int64(n2)
		if err != nil {
			return n, err
		}

		for _, subpart := range body.Parts {
			n2, err = w.Write([]byte("--"))
			n += int64(n2)
			if err != nil {
				return n, err
			}
			n2, err = w.Write([]byte(boundary))
			n += int64(n2)
			if err != nil {
				return n, err
			}
			n2, err = w.Write([]byte("\n"))
			n += int64(n2)
			if err != nil {
				return n, err
			}

			n3, err := subpart.WriteTo(w)
			n += n3
			if err != nil {
				return n, err
			}
		}
		n2, err = w.Write([]byte("--"))
		n += int64(n2)
		if err != nil {
			return n, err
		}
		n2, err = w.Write([]byte(boundary))
		n += int64(n2)
		if err != nil {
			return n, err
		}
		n2, err = w.Write([]byte("--"))
		n += int64(n2)
		if err != nil {
			return n, err
		}
		n2, err = w.Write([]byte("\n"))
		n += int64(n2)
		if err != nil {
			return n, err
		}
		n2, err = w.Write([]byte(body.Postamble))
		n += int64(n2)
		return n, err

	case "message":
		switch p.MinorType() {
		case "rfc822", "news": // message/news == message/rfc822 per RFC5537
			body := p.B.(*Message)
			n2, err := body.WriteTo(w)
			n += n2
			return n, err

		case "delivery-status":
			body := p.B.(*DeliveryStatus)
			n2, err := body.WriteTo(w)
			n += n2
			return n, err

		default:
			return n, ErrUnimplemented
		}

	default:
		body := p.B.(string)
		n2, err := w.Write([]byte(body))
		n += int64(n2)
		return n, err
	}
}

// WriteTo implements the io.WriterTo interface.
func (h Header) WriteTo(w io.Writer) (int64, error) {
	var n int64
	for _, f := range h.Fields {
		n1, err := f.WriteTo(w)
		n += n1
		if err != nil {
			return n, err
		}
	}
	_, err := w.Write([]byte("\n"))
	n++
	return n, err
}

// WriteTo implements the io.WriterTo interface.
func (f Field) WriteTo(w io.Writer) (int64, error) {
	var n int64
	for _, v := range f.V {
		n2, err := w.Write([]byte(f.N))
		n += int64(n2)
		if err != nil {
			return n, err
		}
		n2, err = w.Write([]byte(":"))
		n += int64(n2)
		if err != nil {
			return n, err
		}
		n2, err = w.Write([]byte(v))
		n += int64(n2)
		if err != nil {
			return n, err
		}
		n2, err = w.Write([]byte("\n"))
		n += int64(n2)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// WriteTo implements io.WriterTo.
func (ds *DeliveryStatus) WriteTo(w io.Writer) (int64, error) {
	n, err := ds.Message.WriteTo(w)
	if err != nil {
		return n, err
	}
	for _, recip := range ds.Recipients {
		n2, err := recip.WriteTo(w)
		n += n2
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
