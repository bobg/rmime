package rmime

import (
	"fmt"
	"testing"
)

func TestFieldName(t *testing.T) {
	cases := []struct {
		inp, want string
	}{
		{inp: "foo", want: "Foo"},
		{inp: "Foo", want: "Foo"},
		{inp: "FOO", want: "Foo"},
		{inp: "fOO", want: "Foo"},

		{inp: "foo-bar", want: "Foo-Bar"},
		{inp: "Foo-bar", want: "Foo-Bar"},
		{inp: "FOO-bar", want: "Foo-Bar"},
		{inp: "fOO-bar", want: "Foo-Bar"},

		{inp: "foo-Bar", want: "Foo-Bar"},
		{inp: "Foo-Bar", want: "Foo-Bar"},
		{inp: "FOO-Bar", want: "Foo-Bar"},
		{inp: "fOO-Bar", want: "Foo-Bar"},

		{inp: "foo-BAR", want: "Foo-Bar"},
		{inp: "Foo-BAR", want: "Foo-Bar"},
		{inp: "FOO-BAR", want: "Foo-Bar"},
		{inp: "fOO-BAR", want: "Foo-Bar"},

		{inp: "foo-bAR", want: "Foo-Bar"},
		{inp: "Foo-bAR", want: "Foo-Bar"},
		{inp: "FOO-bAR", want: "Foo-Bar"},
		{inp: "fOO-bAR", want: "Foo-Bar"},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case_%02d", i+1), func(t *testing.T) {
			f := Field{N: tc.inp}
			got := f.Name()
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
