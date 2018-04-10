package opt

import (
	"testing"
	"time"

	"github.com/gotestyourself/gotestyourself/assert"
)

func TestDurationWithThreshold(t *testing.T) {
	var testcases = []struct {
		name            string
		x, y, threshold time.Duration
		expected        bool
	}{
		{
			name:      "delta is threshold",
			threshold: time.Second,
			x:         3 * time.Second,
			y:         2 * time.Second,
			expected:  true,
		},
		{
			name:      "delta is negative threshold",
			threshold: time.Second,
			x:         2 * time.Second,
			y:         3 * time.Second,
			expected:  true,
		},
		{
			name:      "delta within threshold",
			threshold: time.Second,
			x:         300 * time.Millisecond,
			y:         100 * time.Millisecond,
			expected:  true,
		},
		{
			name:      "delta within negative threshold",
			threshold: time.Second,
			x:         100 * time.Millisecond,
			y:         300 * time.Millisecond,
			expected:  true,
		},
		{
			name:      "delta outside threshold",
			threshold: time.Second,
			x:         5 * time.Second,
			y:         300 * time.Millisecond,
		},
		{
			name:      "delta outside negative threshold",
			threshold: time.Second,
			x:         300 * time.Millisecond,
			y:         5 * time.Second,
		},
		{
			name:      "x is 0",
			threshold: time.Second,
			y:         5 * time.Millisecond,
		},
		{
			name:      "y is 0",
			threshold: time.Second,
			x:         5 * time.Millisecond,
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			actual := cmpDuration(testcase.threshold)(testcase.x, testcase.y)
			assert.Equal(t, actual, testcase.expected)
		})
	}
}

func TestTimeWithThreshold(t *testing.T) {
	var now = time.Now()

	var testcases = []struct {
		name      string
		x, y      time.Time
		threshold time.Duration
		expected  bool
	}{
		{
			name:      "delta is threshold",
			threshold: time.Minute,
			x:         now,
			y:         now.Add(time.Minute),
			expected:  true,
		},
		{
			name:      "delta is negative threshold",
			threshold: time.Minute,
			x:         now,
			y:         now.Add(-time.Minute),
			expected:  true,
		},
		{
			name:      "delta within threshold",
			threshold: time.Hour,
			x:         now,
			y:         now.Add(time.Minute),
			expected:  true,
		},
		{
			name:      "delta within negative threshold",
			threshold: time.Hour,
			x:         now,
			y:         now.Add(-time.Minute),
			expected:  true,
		},
		{
			name:      "delta outside threshold",
			threshold: time.Second,
			x:         now,
			y:         now.Add(time.Minute),
		},
		{
			name:      "delta outside negative threshold",
			threshold: time.Second,
			x:         now,
			y:         now.Add(-time.Minute),
		},
		{
			name:      "x is 0",
			threshold: time.Second,
			y:         now,
		},
		{
			name:      "y is 0",
			threshold: time.Second,
			x:         now,
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			actual := cmpTime(testcase.threshold)(testcase.x, testcase.y)
			assert.Equal(t, actual, testcase.expected)
		})
	}
}
