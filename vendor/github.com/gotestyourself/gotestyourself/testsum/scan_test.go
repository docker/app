package testsum

import (
	"bytes"
	"io/ioutil"
	"runtime"
	"strings"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/assert/opt"
)

var cmpSummary = gocmp.Options{
	gocmp.AllowUnexported(Failure{}),
	gocmp.FilterPath(fieldpath("Elapsed"), cmpElapsed()),
}

func fieldpath(spec string) func(gocmp.Path) bool {
	return func(path gocmp.Path) bool {
		return path.String() == spec
	}
}

func cmpElapsed() gocmp.Option {
	// appveyor reports 0 seconds elapsed.
	if runtime.GOOS == "windows" {
		return gocmp.Ignore()
	}
	return opt.DurationWithThreshold(time.Millisecond)
}

func TestScanNoFailures(t *testing.T) {
	source := `=== RUN   TestRunCommandSuccess
--- PASS: TestRunCommandSuccess (0.00s)
=== RUN   TestRunCommandWithCombined
--- PASS: TestRunCommandWithCombined (0.00s)
=== RUN   TestRunCommandWithTimeoutFinished
--- PASS: TestRunCommandWithTimeoutFinished (0.00s)
=== RUN   TestRunCommandWithTimeoutKilled
--- PASS: TestRunCommandWithTimeoutKilled (1.25s)
=== RUN   TestRunCommandWithErrors
--- PASS: TestRunCommandWithErrors (0.00s)
=== RUN   TestRunCommandWithStdoutStderr
--- PASS: TestRunCommandWithStdoutStderr (0.00s)
=== RUN   TestRunCommandWithStdoutStderrError
--- PASS: TestRunCommandWithStdoutStderrError (0.00s)
=== RUN   TestSkippedBecauseSomething
--- SKIP: TestSkippedBecauseSomething (0.00s)
        scan_test.go:39: because blah
PASS
ok      github.com/gotestyourself/gotestyourself/icmd   1.256s
`

	out := new(bytes.Buffer)
	summary, err := Scan(strings.NewReader(source), out)
	assert.NilError(t, err)

	expected := &Summary{Total: 8, Skipped: 1, Elapsed: 10 * time.Microsecond}
	assert.Check(t, is.DeepEqual(summary, expected, cmpSummary))
	assert.Equal(t, source, out.String())
}

func TestScanWithFailure(t *testing.T) {
	source := `=== RUN   TestRunCommandWithCombined
--- PASS: TestRunCommandWithCombined (0.00s)
=== RUN   TestRunCommandWithStdoutStderrError
--- PASS: TestRunCommandWithStdoutStderrError (0.00s)
=== RUN   TestThisShouldFail
Some output
More output
--- FAIL: TestThisShouldFail (0.00s)
        dummy_test.go:11: test is bad
        dummy_test.go:12: another failure
FAIL
exit status 1
FAIL    github.com/gotestyourself/gotestyourself/testsum        0.002s
`

	out := new(bytes.Buffer)
	summary, err := Scan(strings.NewReader(source), out)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(source, out.String()))

	expected := &Summary{
		Total:   3,
		Elapsed: 10 * time.Microsecond,
		Failures: []Failure{
			{
				name:   "TestThisShouldFail",
				output: "Some output\nMore output\n",
				logs:   "        dummy_test.go:11: test is bad\n        dummy_test.go:12: another failure\n",
			},
		},
	}
	assert.Check(t, is.DeepEqual(expected, summary, cmpSummary))
}

func TestScanWithNested(t *testing.T) {
	source := `=== RUN   TestNested
=== RUN   TestNested/a
=== RUN   TestNested/b
=== RUN   TestNested/c
--- PASS: TestNested (0.00s)
    --- PASS: TestNested/a (0.00s)
        dummy_test.go:27: Doing something for a
    --- PASS: TestNested/b (0.00s)
        dummy_test.go:27: Doing something for b
    --- PASS: TestNested/c (0.00s)
        dummy_test.go:27: Doing something for c
PASS
`

	summary, err := Scan(strings.NewReader(source), ioutil.Discard)
	assert.NilError(t, err)

	expected := &Summary{Total: 1, Elapsed: 10 * time.Microsecond}
	assert.Check(t, is.DeepEqual(expected, summary, cmpSummary))
}

func TestScanWithNestedFailures(t *testing.T) {
	source := `=== RUN   TestNested
=== RUN   TestNested/a
Output from  a
=== RUN   TestNested/b
Output from  b
=== RUN   TestNested/c
Output from  c
--- FAIL: TestNested (0.00s)
    --- FAIL: TestNested/a (0.00s)
        dummy_test.go:28: Doing something for a
    --- FAIL: TestNested/b (0.00s)
        dummy_test.go:28: Doing something for b
    --- FAIL: TestNested/c (0.00s)
        dummy_test.go:28: Doing something for c
FAIL
exit status 1
`

	summary, err := Scan(strings.NewReader(source), ioutil.Discard)
	assert.NilError(t, err)

	expectedOutput := `=== RUN   TestNested/a
Output from  a
=== RUN   TestNested/b
Output from  b
=== RUN   TestNested/c
Output from  c
`
	expectedLogs := `    --- FAIL: TestNested/a (0.00s)
        dummy_test.go:28: Doing something for a
    --- FAIL: TestNested/b (0.00s)
        dummy_test.go:28: Doing something for b
    --- FAIL: TestNested/c (0.00s)
        dummy_test.go:28: Doing something for c
`

	expected := &Summary{
		Total:   1,
		Elapsed: 10 * time.Microsecond,
		Failures: []Failure{
			{name: "TestNested", output: expectedOutput, logs: expectedLogs},
		},
	}
	assert.Check(t, is.DeepEqual(expected, summary, cmpSummary))
}

func TestSummaryFormatLine(t *testing.T) {
	var testcases = []struct {
		summary  Summary
		expected string
	}{
		{
			summary:  Summary{Total: 15, Elapsed: time.Minute},
			expected: "======== 15 tests in 60.00 seconds ========",
		},
		{
			summary:  Summary{Total: 100, Skipped: 3},
			expected: "======== 100 tests, 3 skipped in 0.00 seconds ========",
		},
		{
			summary: Summary{
				Total:    100,
				Failures: []Failure{{}},
				Elapsed:  3555 * time.Millisecond,
			},
			expected: "======== 100 tests, 1 failed in 3.56 seconds ========",
		},
		{
			summary: Summary{
				Total:    100,
				Skipped:  3,
				Failures: []Failure{{}},
				Elapsed:  42,
			},
			expected: "======== 100 tests, 3 skipped, 1 failed in 0.00 seconds ========",
		},
	}

	for _, testcase := range testcases {
		assert.Equal(t, testcase.expected, testcase.summary.FormatLine())
	}
}
