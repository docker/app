package skip

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/assert/cmp"
)

type fakeSkipT struct {
	reason string
	logs   []string
}

func (f *fakeSkipT) Skip(args ...interface{}) {
	buf := new(bytes.Buffer)
	for _, arg := range args {
		buf.WriteString(fmt.Sprintf("%s", arg))
	}
	f.reason = buf.String()
}

func (f *fakeSkipT) Log(args ...interface{}) {
	f.logs = append(f.logs, fmt.Sprintf("%s", args[0]))
}

func version(v string) string {
	return v
}

func TestIfCondition(t *testing.T) {
	skipT := &fakeSkipT{}
	apiVersion := "v1.4"
	If(skipT, apiVersion < version("v1.6"))

	assert.Equal(t, `apiVersion < version("v1.6")`, skipT.reason)
	assert.Assert(t, cmp.Len(skipT.logs, 0))
}

func TestIfConditionWithMessage(t *testing.T) {
	skipT := &fakeSkipT{}
	apiVersion := "v1.4"
	If(skipT, apiVersion < "v1.6", "see notes")

	assert.Equal(t, `apiVersion < "v1.6": see notes`, skipT.reason)
	assert.Assert(t, cmp.Len(skipT.logs, 0))
}

func TestIfConditionMultiline(t *testing.T) {
	skipT := &fakeSkipT{}
	apiVersion := "v1.4"
	If(
		skipT,
		apiVersion < "v1.6")

	assert.Equal(t, `apiVersion < "v1.6"`, skipT.reason)
	assert.Assert(t, cmp.Len(skipT.logs, 0))
}

func TestIfConditionMultilineWithMessage(t *testing.T) {
	skipT := &fakeSkipT{}
	apiVersion := "v1.4"
	If(
		skipT,
		apiVersion < "v1.6",
		"see notes")

	assert.Equal(t, `apiVersion < "v1.6": see notes`, skipT.reason)
	assert.Assert(t, cmp.Len(skipT.logs, 0))
}

func TestIfConditionNoSkip(t *testing.T) {
	skipT := &fakeSkipT{}
	If(skipT, false)

	assert.Equal(t, "", skipT.reason)
	assert.Assert(t, cmp.Len(skipT.logs, 0))
}

func SkipBecauseISaidSo() bool {
	return true
}

func TestIf(t *testing.T) {
	skipT := &fakeSkipT{}
	If(skipT, SkipBecauseISaidSo)

	assert.Equal(t, "SkipBecauseISaidSo", skipT.reason)
}

func TestIfWithMessage(t *testing.T) {
	skipT := &fakeSkipT{}
	If(skipT, SkipBecauseISaidSo, "see notes")

	assert.Equal(t, "SkipBecauseISaidSo: see notes", skipT.reason)
}
