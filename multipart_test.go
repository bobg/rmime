package rmime

import (
	"io"
	"strings"
	"testing"
)

func TestMultipartReaders(t *testing.T) {
	msg := "preamble line 1\npreamble line 2\n--boundary\npart 1 line 1\npart 1 line 2\n--boundary\npart 2 line 1\n--boundary--\npostamble line 1\npostamble line 2\n"

	t.Run("FullSequentialRead", func(t *testing.T) {
		r := strings.NewReader(msg)
		preambleR, parts, postambleR, errptr := multipartReaders(r, "boundary")

		preamble, err := io.ReadAll(preambleR)
		if err != nil {
			t.Fatalf("reading preamble: %v", err)
		}
		if got, want := string(preamble), "preamble line 1\npreamble line 2\n"; got != want {
			t.Errorf("preamble got %q, want %q", got, want)
		}

		var partContents []string
		for partR := range parts {
			content, err := io.ReadAll(partR)
			if err != nil {
				t.Fatalf("reading part: %v", err)
			}
			partContents = append(partContents, string(content))
		}

		if len(partContents) != 2 {
			t.Fatalf("got %d parts, want 2", len(partContents))
		}
		if got, want := partContents[0], "part 1 line 1\npart 1 line 2\n"; got != want {
			t.Errorf("part 0 got %q, want %q", got, want)
		}
		if got, want := partContents[1], "part 2 line 1\n"; got != want {
			t.Errorf("part 1 got %q, want %q", got, want)
		}

		postamble, err := io.ReadAll(postambleR)
		if err != nil {
			t.Fatalf("reading postamble: %v", err)
		}
		if got, want := string(postamble), "postamble line 1\npostamble line 2\n"; got != want {
			t.Errorf("postamble got %q, want %q", got, want)
		}

		if err := *errptr; err != nil {
			t.Errorf("errptr got %v, want nil", err)
		}
	})

	t.Run("SkipPreamble", func(t *testing.T) {
		r := strings.NewReader(msg)
		_, parts, postambleR, errptr := multipartReaders(r, "boundary")

		var count int
		for partR := range parts {
			_, err := io.ReadAll(partR)
			if err != nil {
				t.Fatalf("reading part: %v", err)
			}
			count++
		}
		if count != 2 {
			t.Errorf("got %d parts, want 2", count)
		}

		postamble, err := io.ReadAll(postambleR)
		if err != nil {
			t.Fatalf("reading postamble: %v", err)
		}
		if got, want := string(postamble), "postamble line 1\npostamble line 2\n"; got != want {
			t.Errorf("postamble got %q, want %q", got, want)
		}

		if err := *errptr; err != nil {
			t.Errorf("errptr got %v, want nil", err)
		}
	})

	t.Run("SkipToPostamble", func(t *testing.T) {
		r := strings.NewReader(msg)
		_, _, postambleR, errptr := multipartReaders(r, "boundary")

		postamble, err := io.ReadAll(postambleR)
		if err != nil {
			t.Fatalf("reading postamble: %v", err)
		}
		if got, want := string(postamble), "postamble line 1\npostamble line 2\n"; got != want {
			t.Errorf("postamble got %q, want %q", got, want)
		}

		if err := *errptr; err != nil {
			t.Errorf("errptr got %v, want nil", err)
		}
	})
}
