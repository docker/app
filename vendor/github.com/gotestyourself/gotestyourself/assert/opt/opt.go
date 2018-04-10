/*Package opt provides common go-cmp.Options for use with assert.DeepEqual.
 */
package opt

import (
	"time"

	gocmp "github.com/google/go-cmp/cmp"
)

// DurationWithThreshold returns a gocmp.Comparer for comparing time.Duration. The
// Comparer returns true if the difference between the two Duration values is
// within the threshold and neither value is zero.
func DurationWithThreshold(threshold time.Duration) gocmp.Option {
	return gocmp.Comparer(cmpDuration(threshold))
}

func cmpDuration(threshold time.Duration) func(x, y time.Duration) bool {
	return func(x, y time.Duration) bool {
		if x == 0 || y == 0 {
			return false
		}
		delta := x - y
		return delta <= threshold && delta >= -threshold
	}
}

// TimeWithThreshold returns a gocmp.Comparer for comparing time.Time. The
// Comparer returns true if the difference between the two Time values is
// within the threshold and neither value is zero.
func TimeWithThreshold(threshold time.Duration) gocmp.Option {
	return gocmp.Comparer(cmpTime(threshold))
}

func cmpTime(threshold time.Duration) func(x, y time.Time) bool {
	return func(x, y time.Time) bool {
		if x.IsZero() || y.IsZero() {
			return false
		}
		delta := x.Sub(y)
		return delta <= threshold && delta >= -threshold
	}
}
