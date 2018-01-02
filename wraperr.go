package rmime

import "fmt"

type wraperr struct {
	e error
	w string
}

func (w wraperr) Error() string {
	return fmt.Sprintf("%s: %s", w.w, w.e)
}

func wrapf(e error, f string, args ...interface{}) error {
	if e == nil {
		return nil
	}
	return wraperr{e: e, w: fmt.Sprintf(f, args...)}
}

func rooterr(e error) error {
	if w, ok := e.(wraperr); ok {
		return rooterr(w.e)
	}
	return e
}
