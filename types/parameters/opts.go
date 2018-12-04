package parameters

// Options contains loading options for parameters
type Options struct {
	prefix string
}

// WithPrefix adds the given prefix when loading parameters
func WithPrefix(prefix string) func(*Options) {
	return func(o *Options) {
		o.prefix = prefix
	}
}
