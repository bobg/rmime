package rmime

import (
	"bufio"
	"io"
	"strings"
)

// TextPlainReader decodes text/plain body parts using the "flowed" and "delsp" features.
// Note that Part.Body() already returns a TextPlainReader when Part.Type() is "text/plain".
// If flowed==false, TextPlainReader simply returns its input, r.
func TextPlainReader(r io.Reader, flowed, delsp bool) io.Reader {
	if !flowed {
		return r
	}

	s := bufio.NewScanner(r)
	pr, pw := io.Pipe()

	go func() {
		var (
			para       []string
			quoteDepth int
		)

		// TODO: reflow paragraphs
		emitPara := func() {
			if len(para) == 0 {
				return
			}
			for _, line := range para {
				if quoteDepth > 0 {
					io.WriteString(pw, strings.Repeat(">", quoteDepth))
					io.WriteString(pw, " ")
				}
				io.WriteString(pw, line)
				io.WriteString(pw, "\n")
			}
			para = nil
			quoteDepth = 0
		}

		for s.Scan() {
			line := s.Text()

			var isSignatureLine bool
			if line == "-- " {
				isSignatureLine = true
			}

			initLen := len(line)
			line = strings.TrimLeftFunc(line, func(c rune) bool { return c == '>' })
			lineQuoteDepth := initLen - len(line)

			if len(line) > 0 && line[0] == ' ' {
				line = line[1:]
			}

			if !isSignatureLine && line == "-- " {
				// second check
				isSignatureLine = true
			}

			var flowed bool
			if !isSignatureLine && len(line) > 0 && line[len(line)-1] == ' ' {
				flowed = true
				if delsp {
					line = line[:len(line)-1]
				}
			}
			if !flowed && strings.IndexFunc(line, func(c rune) bool { return c != ' ' }) < 0 {
				// space-only line is flowed
				flowed = true
			}

			if !flowed || lineQuoteDepth != quoteDepth {
				// end-of-flowed-paragraph conditions
				emitPara()
			}

			if flowed {
				para = append(para, line)
				quoteDepth = lineQuoteDepth
			} else {
				io.WriteString(pw, line)
				io.WriteString(pw, "\n")
			}
		}
		emitPara()
		pw.Close()
	}()

	return pr
}
