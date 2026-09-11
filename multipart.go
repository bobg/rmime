package rmime

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"iter"
	"mime"

	"github.com/bobg/errors"
	"github.com/bobg/seqs"
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

	preambleReader, partReaders, postambleReader, errptr := multipartReaders(r, boundary)

	preamble, err := io.ReadAll(preambleReader)
	if err != nil {
		return nil, errors.Wrap(err, "reading multipart preamble")
	}

	var parts []*Part

	for partReader := range partReaders {
		part, err := ReadPart(partReader, header)
		if err != nil {
			return nil, errors.Wrap(err, "reading multipart part")
		}
		parts = append(parts, part)
	}

	postamble, err := io.ReadAll(postambleReader)
	if err != nil {
		return nil, errors.Wrap(err, "reading multipart postamble")
	}

	if err := *errptr; err != nil {
		return nil, errors.Wrap(err, "reading multipart")
	}

	return &Multipart{Preamble: string(preamble), Postamble: string(postamble), Parts: parts}, nil
}

func multipartReaders(r io.Reader, boundary string) (preambleReader io.Reader, partReaders iter.Seq[Reader], postambleReader io.Reader, errptr *error) {
	var outerErr error

	preamblePipeR, preamblePipeW := io.Pipe()
	postamblePipeR, postamblePipeW := io.Pipe()

	ch := make(chan Reader)

	go func() {
		defer close(ch)
		defer preamblePipeW.Close()
		defer postamblePipeW.Close()

		var final bool

		lineReaders, errptr := seqs.LongLines(r)
		for r := range lineReaders {
			line, err := io.ReadAll(r)
			if err != nil {
				outerErr = errors.Wrap(err, "reading line in preamble")
				return
			}

			var match bool
			match, final = isBoundary(line, boundary)
			if match {
				if err := preamblePipeW.Close(); err != nil {
					outerErr = errors.Wrap(err, "closing preamble pipe")
					return
				}
				break
			}

			line = append(line, '\n')
			if _, err := preamblePipeW.Write(line); err != nil {
				outerErr = errors.Wrap(err, "writing preamble")
				return
			}
		}

		for !final {
			partR, partW := io.Pipe()
			ch <- bufio.NewReader(partR)

			var err error
			final, err = copyUntilBoundary(lineReaders, partW, boundary)
			if err != nil {
				outerErr = errors.Wrap(err, "copying part")
				return
			}
		}

		for r := range lineReaders {
			line, err := io.ReadAll(r)
			if err != nil {
				outerErr = errors.Wrap(err, "reading line in postamble")
				return
			}

			line = append(line, '\n')
			if _, err := postamblePipeW.Write(line); err != nil {
				outerErr = errors.Wrap(err, "writing postamble")
				return
			}
		}

		if err := postamblePipeW.Close(); err != nil {
			outerErr = errors.Wrap(err, "closing postamble pipe")
			return
		}

		if err := *errptr; err != nil {
			outerErr = errors.Wrap(err, "reading multipart")
			return
		}
	}()

	return preamblePipeR, seqs.FromChan(ch), postamblePipeR, &outerErr
}

func copyUntilBoundary(lineReaders iter.Seq[io.Reader], w *io.PipeWriter, boundary string) (final bool, err error) {
	defer func() {
		if err2 := w.CloseWithError(err); err2 != nil {
			err2 = errors.Wrap(err2, "closing pipe after error")
			err = errors.Join(err, err2)
		}
	}()

	for r := range lineReaders {
		line, err := io.ReadAll(r)
		if err != nil {
			return false, errors.Wrap(err, "reading line in multipart part")
		}
		if match, final := isBoundary(line, boundary); match {
			return final, nil
		}
		line = append(line, '\n')
		if _, err := w.Write(line); err != nil {
			return false, errors.Wrap(err, "writing line of multipart part")
		}
	}

	return false, errors.Wrap(io.ErrUnexpectedEOF, "while reading multipart part")
}

func isBoundary(line []byte, boundary string) (match bool, final bool) {
	if len(line) < len(boundary)+2 {
		return false, false
	}
	if !bytes.Equal(line[:2], []byte("--")) {
		return false, false
	}
	if string(line[2:2+len(boundary)]) != boundary {
		return false, false
	}
	rest := line[2+len(boundary):]
	if len(rest) >= 2 && bytes.Equal(rest[:2], []byte("--")) {
		final = true
		rest = rest[2:]
	}
	// allow only LWSP and \r?\n
	if len(rest) > 0 && rest[len(rest)-1] == '\n' {
		rest = rest[:len(rest)-1]
		if len(rest) > 0 && rest[len(rest)-1] == '\r' {
			rest = rest[:len(rest)-1]
		}
	}
	for _, r := range rest {
		switch r {
		case ' ', '\f', '\t', '\v':
			// ignore
		default:
			return false, false
		}
	}
	return true, final
}
