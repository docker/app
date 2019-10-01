package yaml

import (
	"bytes"
	"io"

	"gopkg.in/yaml.v2"
)

// Unmarshal decodes the first document found within the in byte slice
// and assigns decoded values into the out value.
//
// See gopkg.in/yaml.v2 documentation
func Unmarshal(in []byte, out interface{}) error {
	d := yaml.NewDecoder(bytes.NewBuffer(in))
	err := d.Decode(out)
	if err == io.EOF {
		return nil
	}
	return err
}

// Marshal serializes the value provided into a YAML document. The structure
// of the generated document will reflect the structure of the value itself.
// Maps and pointers (to struct, string, int, etc) are accepted as the in value.
//
// See gopkg.in/yaml.v2 documentation
func Marshal(in interface{}) ([]byte, error) {
	return yaml.Marshal(in)
}

// NewDecoder returns a new decoder that reads from r.
//
// See gopkg.in/yaml.v2 documentation
func NewDecoder(r io.Reader) *yaml.Decoder {
	return yaml.NewDecoder(r)
}
