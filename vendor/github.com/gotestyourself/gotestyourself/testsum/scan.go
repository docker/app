/*Package testsum is DEPRECATED.

This functionality is now available from `go tool test2json` in the stdlib.

Package testsum provides functions for parsing `go test -v` output and returning
a summary of the test run.

Build the executable:

    go build -o gotestsum ./testsum/cmd

Usage:

    go test -v ./... | gotestsum

Example output:

    === RUN   TestPass
    --- PASS: TestPass (0.00s)
    === RUN   TestSkip
    --- SKIP: TestSkip (0.00s)
            example_test.go:11:
    === RUN   TestFail
    Some test output
    --- FAIL: TestFail (0.00s)
            example_test.go:22: some log output
    FAIL
    exit status 1
    FAIL    example.com/gotestyourself/testpkg        0.002s
    ======== 3 tests, 1 skipped, 1 failed in 2.28 seconds ========
    --- FAIL: TestFail
    Some test output

            example_test.go:22: some log output
*/
package testsum

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Failure test including output
type Failure struct {
	name   string
	output string
	logs   string
}

func (f Failure) String() string {
	buf := bytes.NewBufferString("--- FAIL: " + f.name)
	if f.output != "" {
		buf.WriteString("\n" + f.output)
	}
	buf.WriteString("\n" + f.logs + "\n")
	return buf.String()
}

// Summary information from a `go test -v` run. Includes counts of tests and
// a list of failed tests with the test output
type Summary struct {
	Total    int
	Skipped  int
	Elapsed  time.Duration
	Failures []Failure
}

// FormatLine returns a line with counts of tests, skipped, and failed
func (s *Summary) FormatLine() string {
	bar := "========"
	buf := bytes.NewBufferString(fmt.Sprintf(bar+" %d tests", s.Total))
	if s.Skipped != 0 {
		buf.WriteString(fmt.Sprintf(", %d skipped", s.Skipped))
	}
	if len(s.Failures) > 0 {
		buf.WriteString(fmt.Sprintf(", %d failed", len(s.Failures)))
	}
	buf.WriteString(fmt.Sprintf(" in %0.2f seconds", s.Elapsed.Seconds()))
	buf.WriteString(" " + bar)
	return buf.String()
}

// FormatFailures returns a string with all the test failure and the test output.
// Returns a empty string if there are no failures.
func (s *Summary) FormatFailures() string {
	formatted := []string{}
	for _, failure := range s.Failures {
		formatted = append(formatted, failure.String())
	}
	return strings.Join(formatted, "\n")
}

func (s *Summary) addFailure(failure *Failure) {
	if failure != nil {
		s.Failures = append(s.Failures, *failure)
	}
}

// Scan reads lines from the reader, echos them to the writer, and parses the
// lines for `go test -v` output. It returns a summary of the test run.
func Scan(in io.Reader, out io.Writer) (*Summary, error) {
	summary := &Summary{}
	state := newScanState()
	start := time.Now()
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		if _, err := out.Write([]byte(line + "\n")); err != nil {
			return summary, errors.Wrapf(err, "failed to echo output")
		}

		parseLine(summary, state, line)
	}
	if err := scanner.Err(); err != nil {
		return summary, errors.Wrapf(err, "failed to scan input")
	}
	summary.Elapsed = time.Since(start)
	return summary, nil
}

type state int

const (
	stateNone = iota
	stateRun
	stateFail
)

type scanState struct {
	buffer      *bytes.Buffer
	currentFail *Failure
	state       state
}

func newScanState() *scanState {
	return &scanState{buffer: new(bytes.Buffer)}
}

func (s *scanState) end() {
	s.state = stateNone
	s.currentFail = nil
	s.buffer.Reset()
}

func (s *scanState) addLine(line string) {
	s.buffer.WriteString(line + "\n")
}

func (s *scanState) start(name string) {
	s.state = stateRun
	s.currentFail = &Failure{name: name}
}

func (s *scanState) getFailure() *Failure {
	defer s.end()

	if s.state != stateFail {
		return nil
	}
	failure := s.currentFail
	failure.logs = s.buffer.String()
	return failure
}

func (s *scanState) setFailed() {
	s.state = stateFail
	s.currentFail.output = s.buffer.String()
	s.buffer.Reset()
}

var runPrefixLength = len("=== RUN ")

func parseLine(summary *Summary, state *scanState, line string) {
	switch {
	// Nested tests start with the same line prefix so only start a new test
	// if not already in a run state
	case state.state != stateRun && strings.HasPrefix(line, "=== RUN   "):
		summary.addFailure(state.getFailure())
		state.start(strings.TrimSpace(line[runPrefixLength:]))
		summary.Total++
	case strings.HasPrefix(line, "--- PASS: "):
		state.end()
	case strings.HasPrefix(line, "--- SKIP: "):
		summary.Skipped++
		state.end()
	case strings.HasPrefix(line, "--- FAIL: "):
		state.setFailed()
	case isEndOfTestRun(line):
		summary.addFailure(state.getFailure())
	default:
		state.addLine(line)
	}
}

func isEndOfTestRun(line string) bool {
	return line == "FAIL" || line == "PASS"
}
