package loader

import (
	"github.com/deislabs/duffle/pkg/bundle"

	"golang.org/x/crypto/openpgp/clearsign"
)

// DetectingLoader loads a file or data, and then determines the content.
//
// A DetectingLoader NEVER verifies a signature. If the bundle is signed, it will extract
// the body, and then parse. If it is raw JSON, it will parse.
//
// DetectingLoader is INSECURE, as it does not verify the bundle.
type DetectingLoader struct{}

// NewDetectingLoader creates a new loader that can detect the file type
//
// It will only detect between the '.cnab' format (signed bundles) and
// the '.json' (JSON) format.
func NewDetectingLoader() *DetectingLoader {
	return &DetectingLoader{}
}

// Load loads a file from the filesystem and parses it.
func (l *DetectingLoader) Load(filename string) (*bundle.Bundle, error) {
	data, err := loadData(filename)
	if err != nil {
		return nil, err
	}
	return l.LoadData(data)
}

// LoadData loads file from a byte slice and parses it.
func (l *DetectingLoader) LoadData(data []byte) (*bundle.Bundle, error) {
	// clearsign.Decode provides a safe way of dealing with clearsigned
	// blocks. It will return a block ONLY IF it finds a clearsign header
	// at the beginning of a line, which is not valid JSON (and therefore)
	// can't occur in a legitimate bundle.json).
	block, _ := clearsign.Decode(data)
	if block != nil {
		data = block.Bytes
	}
	// Delegate parsing to an unsigned_loader
	return NewUnsignedLoader().LoadData(data)
}
