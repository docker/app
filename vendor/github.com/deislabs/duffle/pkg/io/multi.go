package io

import "io"

type eofReadCloser struct{}

func (eofReadCloser) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (eofReadCloser) Close() error {
	return nil
}

type multiReadCloser struct {
	readclosers []io.ReadCloser
}

func (mr *multiReadCloser) Read(p []byte) (n int, err error) {
	for len(mr.readclosers) > 0 {
		// Optimization to flatten nested multiReaders (Issue 13558).
		if len(mr.readclosers) == 1 {
			if r, ok := mr.readclosers[0].(*multiReadCloser); ok {
				mr.readclosers = r.readclosers
				continue
			}
		}
		n, err = mr.readclosers[0].Read(p)
		if err == io.EOF {
			// Use eofReadCloser instead of nil to avoid nil panic
			// after performing flatten (Issue 18232).
			mr.readclosers[0] = eofReadCloser{} // permit earlier GC
			mr.readclosers = mr.readclosers[1:]
		}
		if n > 0 || err != io.EOF {
			if err == io.EOF && len(mr.readclosers) > 0 {
				// Don't return EOF yet. More readclosers remain.
				err = nil
			}
			return
		}
	}
	return 0, io.EOF
}

func (mr *multiReadCloser) Close() (err error) {
	for i := range mr.readclosers {
		if err := mr.readclosers[i].Close(); err != nil {
			return err
		}
	}
	return nil
}

// MultiReadCloser returns a Reader that's the logical concatenation of
// the provided input readers. They're read sequentially. Once all
// inputs have returned EOF, Read will return EOF.  If any of the readers
// return a non-nil, non-EOF error, Read will return that error.
func MultiReadCloser(readclosers ...io.ReadCloser) io.ReadCloser {
	r := make([]io.ReadCloser, len(readclosers))
	copy(r, readclosers)
	return &multiReadCloser{r}
}
