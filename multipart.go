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
	lineReaders, linesErrPtr := seqs.LongLines(r)
	next, stop := iter.Pull(lineReaders)

	s := &mpStream{
		lineNext: next,
		lineStop: stop,
		boundary: boundary,
		linesErr: linesErrPtr,
		stage:    stagePreamble,
	}

	preamble := &preambleReaderImpl{s: s}
	parts := s.partSeq()
	postamble := &postambleReaderImpl{s: s}

	return preamble, parts, postamble, &s.err
}

type mStage int

const (
	stagePreamble mStage = iota
	stagePart
	stagePostamble
	stageDone
)

type mpStream struct {
	lineNext func() (io.Reader, bool)
	lineStop func()
	boundary string
	linesErr *error
	err      error
	buf      []byte
	stage    mStage
	final    bool
}

func (s *mpStream) readNextLine() ([]byte, bool) {
	if s.err != nil {
		return nil, false
	}
	r, ok := s.lineNext()
	if !ok {
		s.lineStop()
		if s.linesErr != nil && *s.linesErr != nil {
			s.err = *s.linesErr
		}
		return nil, false
	}
	line, err := io.ReadAll(r)
	if err != nil {
		s.err = err
		s.lineStop()
		return nil, false
	}
	return line, true
}

type preambleReaderImpl struct {
	s *mpStream
}

func (pr *preambleReaderImpl) Read(p []byte) (int, error) {
	if pr.s.stage > stagePreamble {
		return 0, io.EOF
	}
	if len(pr.s.buf) > 0 {
		n := copy(p, pr.s.buf)
		pr.s.buf = pr.s.buf[n:]
		return n, nil
	}

	line, ok := pr.s.readNextLine()
	if !ok {
		pr.s.stage = stageDone
		if pr.s.err != nil {
			return 0, pr.s.err
		}
		return 0, io.EOF
	}

	match, final := isBoundary(line, pr.s.boundary)
	if match {
		pr.s.stage = stagePart
		pr.s.final = final
		return 0, io.EOF
	}

	line = append(line, '\n')
	n := copy(p, line)
	if n < len(line) {
		pr.s.buf = append(pr.s.buf, line[n:]...)
	}
	return n, nil
}

func (s *mpStream) partSeq() iter.Seq[Reader] {
	return func(yield func(Reader) bool) {
		if s.stage == stagePreamble {
			if err := s.drainPreamble(); err != nil {
				return
			}
		}

		for s.stage == stagePart && !s.final {
			pr := &singlePartReaderImpl{s: s}
			bufPR := bufio.NewReader(pr)
			if !yield(bufPR) {
				return
			}
			if err := pr.drain(); err != nil {
				return
			}
		}
		s.stage = stagePostamble
	}
}

func (s *mpStream) drainPreamble() error {
	pr := &preambleReaderImpl{s: s}
	buf := make([]byte, 1024)
	for s.stage == stagePreamble {
		_, err := pr.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

type singlePartReaderImpl struct {
	s    *mpStream
	done bool
}

func (pr *singlePartReaderImpl) Read(p []byte) (int, error) {
	if pr.done || pr.s.stage != stagePart {
		return 0, io.EOF
	}
	if len(pr.s.buf) > 0 {
		n := copy(p, pr.s.buf)
		pr.s.buf = pr.s.buf[n:]
		return n, nil
	}

	line, ok := pr.s.readNextLine()
	if !ok {
		pr.done = true
		pr.s.stage = stageDone
		if pr.s.err != nil {
			return 0, pr.s.err
		}
		pr.s.err = errors.Wrap(io.ErrUnexpectedEOF, "while reading multipart part")
		return 0, pr.s.err
	}

	match, final := isBoundary(line, pr.s.boundary)
	if match {
		pr.done = true
		pr.s.final = final
		if final {
			pr.s.stage = stagePostamble
		}
		return 0, io.EOF
	}

	line = append(line, '\n')
	n := copy(p, line)
	if n < len(line) {
		pr.s.buf = append(pr.s.buf, line[n:]...)
	}
	return n, nil
}

func (pr *singlePartReaderImpl) drain() error {
	if pr.done {
		return nil
	}
	buf := make([]byte, 1024)
	for !pr.done {
		_, err := pr.Read(buf)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
	return nil
}

type postambleReaderImpl struct {
	s *mpStream
}

func (po *postambleReaderImpl) Read(p []byte) (int, error) {
	if po.s.stage < stagePostamble {
		if err := po.s.drainParts(); err != nil {
			return 0, err
		}
	}
	if po.s.stage > stagePostamble {
		return 0, io.EOF
	}
	if len(po.s.buf) > 0 {
		n := copy(p, po.s.buf)
		po.s.buf = po.s.buf[n:]
		return n, nil
	}

	line, ok := po.s.readNextLine()
	if !ok {
		po.s.stage = stageDone
		if po.s.err != nil {
			return 0, po.s.err
		}
		return 0, io.EOF
	}

	line = append(line, '\n')
	n := copy(p, line)
	if n < len(line) {
		po.s.buf = append(po.s.buf, line[n:]...)
	}
	return n, nil
}

func (s *mpStream) drainParts() error {
	if s.stage == stagePreamble {
		if err := s.drainPreamble(); err != nil {
			return err
		}
	}
	for s.stage == stagePart && !s.final {
		pr := &singlePartReaderImpl{s: s}
		if err := pr.drain(); err != nil {
			return err
		}
	}
	s.stage = stagePostamble
	return nil
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
