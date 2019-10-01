package yaml

import (
	"bytes"
	"testing"

	"gotest.tools/assert"
)

func TestDecoderYamlBomb(t *testing.T) {
	var v map[interface{}]interface{}
	data := []byte(`version: "3"
services: &services ["lol","lol","lol","lol","lol","lol","lol","lol","lol"]
b: &b [*services,*services,*services,*services,*services,*services,*services,*services,*services]
c: &c [*b,*b,*b,*b,*b,*b,*b,*b,*b]
d: &d [*c,*c,*c,*c,*c,*c,*c,*c,*c]
e: &e [*d,*d,*d,*d,*d,*d,*d,*d,*d]
f: &f [*e,*e,*e,*e,*e,*e,*e,*e,*e]
g: &g [*f,*f,*f,*f,*f,*f,*f,*f,*f]
h: &h [*g,*g,*g,*g,*g,*g,*g,*g,*g]
i: &i [*h,*h,*h,*h,*h,*h,*h,*h,*h]`)
	d := NewDecoder(bytes.NewBuffer(data))
	err := d.Decode(&v)
	assert.ErrorContains(t, err, "yaml: document contains excessive aliasing")
}

func TestUnmarshalYamlBomb(t *testing.T) {
	var v map[interface{}]interface{}
	data := []byte(`version: "3"
services: &services ["lol","lol","lol","lol","lol","lol","lol","lol","lol"]
b: &b [*services,*services,*services,*services,*services,*services,*services,*services,*services]
c: &c [*b,*b,*b,*b,*b,*b,*b,*b,*b]
d: &d [*c,*c,*c,*c,*c,*c,*c,*c,*c]
e: &e [*d,*d,*d,*d,*d,*d,*d,*d,*d]
f: &f [*e,*e,*e,*e,*e,*e,*e,*e,*e]
g: &g [*f,*f,*f,*f,*f,*f,*f,*f,*f]
h: &h [*g,*g,*g,*g,*g,*g,*g,*g,*g]
i: &i [*h,*h,*h,*h,*h,*h,*h,*h,*h]`)
	err := Unmarshal(data, &v)
	assert.ErrorContains(t, err, "yaml: document contains excessive aliasing")
}
