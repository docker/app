package subtest

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
)

func TestRunCallsCleanup(t *testing.T) {
	calls := []int{}
	Run(t, "test-run-cleanup", func(t TestContext) {
		cleanup := func(n int) func() {
			return func() {
				calls = append(calls, n)
			}
		}
		t.AddCleanup(cleanup(2))
		t.AddCleanup(cleanup(1))
		t.AddCleanup(cleanup(0))
	})
	assert.DeepEqual(t, calls, []int{0, 1, 2})
}
