package digest

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
)

// OfReader reads in a stream and spits out a truncated checksum with output similar to `shasum -a 256 build.tar.gz | awk '{print $1}'`.
//
// This is *incredibly* poor on performance as it reads in the entire stream to compute the checksum, and should be used sparingly.
func OfReader(r io.Reader) (io.Reader, string, error) {
	// write r to a buffer so we can also write to the sha256 hash.
	buf := new(bytes.Buffer)
	h := sha256.New()
	w := io.MultiWriter(buf, h)
	if _, err := io.Copy(w, r); err != nil {
		return nil, "", err
	}

	fulltag := h.Sum(nil)
	tag := fmt.Sprintf("%.20x", fulltag)
	return buf, tag, nil
}

// OfBuffer reads in a byte slice and spits out a truncated checksum with output similar to `shasum -a 256 build.tar.gz | awk '{print $1}'`.
func OfBuffer(b []byte) (string, error) {
	h := sha256.New()

	if _, err := h.Write(b); err != nil {
		return "", err
	}
	fulltag := h.Sum(nil)
	tag := fmt.Sprintf("%.20x", fulltag)
	return tag, nil
}
