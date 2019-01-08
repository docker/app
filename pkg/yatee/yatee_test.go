package yatee

import (
	"testing"

	"github.com/docker/app/internal/yaml"
	"gotest.tools/assert"
)

func testEval(t *testing.T, input string, env map[string]interface{}, output interface{}) {
	v, err := eval(input, env, options{})
	assert.NilError(t, err)
	assert.Equal(t, v, output)
}

func TestEval(t *testing.T) {
	env := map[string]interface{}{"false": "false", "foo": "bar", "baz": "bam", "bar": "pim", "barbam": "poum"}
	testEval(t, "$foo ${baz}", env, "bar bam")
	testEval(t, "${foo}$baz", env, "barbam")
	testEval(t, "$(1 + 1)", env, int64(2))
	testEval(t, "$(1+2*3)", env, int64(9))
	testEval(t, "$(1+(2*3))", env, int64(7))
	testEval(t, "$(1+((2*3)))", env, int64(7))
	testEval(t, "$$$foo $$${baz}", env, "$bar $bam")
	testEval(t, "${$foo}", env, "pim")
	testEval(t, "${${foo}$baz}", env, "poum")
	testEval(t, "${false?foo:bar}", env, "bar")
	testEval(t, "${foo?foo:bar}", env, "foo")
}

func testProcess(t *testing.T, input, output, parameters, error string) {
	ps := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(parameters), ps)
	assert.NilError(t, err)
	s := make(map[string]interface{})
	merge(s, ps)
	res, err := Process(input, s)
	if error == "" {
		assert.NilError(t, err)
		sres, err := yaml.Marshal(res)
		assert.NilError(t, err)
		assert.Equal(t, output, string(sres))
	} else {
		assert.Equal(t, err.Error(), error)
	}
}

func TestProcess(t *testing.T) {
	parameters := `
foo: bar
loop: $loop
app:
  mode: debug
  release: false
  count: 2
ab:
  - a
  - b
`
	testProcess(t,
		`services:
  "@if $app.release":
     no1: 1
  "@if ! $app.release":
     yes1: 1
     "@else":
       no3: 3
  "@if $($app.count - 2)":
    no2: 2
    "@else":
      yes3: 3
  "@if !$($app.count - 2)":
    yes2: 2
`, `services:
  yes1: 1
  yes2: 2
  yes3: 3
`, parameters, "")

	testProcess(t,
		`switch:
  "@switch $app.mode":
    debug:
      yes1: 1
    other:
      no1: 1
    default:
      no2: 2
  "@switch ${app.mode}":
    default:
      yes2: 2
    release:
      no3: 3
`, `switch:
  yes1: 1
  yes2: 2
`, parameters, "")

	testProcess(t,
		`services:
  "@for i in 0..$app.count":
    app$i: $($i+1)
  "@for i in $ab":
    bapp$i: foo$i
`, `services:
  app0: 1
  app1: 2
  bappa: fooa
  bappb: foob
`, parameters, "")

	testProcess(t,
		`list:
  - v1
  - "@if (true) vtrue"
  - "@if (false) vfalse"
  - "@if (!false) vtrue2"
`, `list:
- v1
- vtrue
- vtrue2
`, parameters, "")

	testProcess(t,
		`services:
  "@for i in 0..2":
    "@for j in 0..2":
      foo$i$j: $($i*2 + $j)
      "@if $($i+$j%2)":
        bar$i$j: 1
`, `services:
  bar01: 1
  bar10: 1
  foo00: 0
  foo01: 1
  foo10: 2
  foo11: 3
`, parameters, "")

	testProcess(t,
		"services: $loop",
		"",
		parameters,
		"eval loop detected")
}

func testProcessWithOrder(t *testing.T, input, output, error string) {
	parameters := make(map[string]interface{})

	res, err := ProcessWithOrder(input, parameters)

	assert.NilError(t, err, "Error processing input: "+input)
	sres, err := yaml.Marshal(res)
	assert.NilError(t, err)
	assert.Equal(t, output, string(sres), "Input was:"+string(sres)+"\nOutput was:"+output)
}

func TestProcessWithOrder(t *testing.T) {
	// Test ordering is preserved inside nested structures
	testProcessWithOrder(t,
		`parent:
  bb: true
  aa: false
`, `parent:
  bb: true
  aa: false
`, "")

	// Test ordering is preserved at the top level
	testProcessWithOrder(t,
		`bbb:
  nested: true
aaa:
  nested: false
`, `bbb:
  nested: true
aaa:
  nested: false
`, "")
}
