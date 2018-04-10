package skip_test

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/skip"
)

var apiVersion = ""

type env struct{}

func (e env) hasFeature(_ string) bool { return false }

var testEnv = env{}

func MissingFeature() bool { return false }

var t = &testing.T{}

func ExampleIf() {
	//   --- SKIP: TestName (0.00s)
	//           skip.go:19: MissingFeature
	skip.If(t, MissingFeature)

	//   --- SKIP: TestName (0.00s)
	//           skip.go:19: MissingFeature: coming soon
	skip.If(t, MissingFeature, "coming soon")
}

func ExampleIfCondition() {
	//   --- SKIP: TestName (0.00s)
	//           skip.go:19: apiVersion < version("v1.24")
	skip.IfCondition(t, apiVersion < version("v1.24"))

	//   --- SKIP: TestName (0.00s)
	//           skip.go:19: !textenv.hasFeature("build"): coming soon
	skip.IfCondition(t, !testEnv.hasFeature("build"), "coming soon")
}

func version(v string) string {
	return v
}
