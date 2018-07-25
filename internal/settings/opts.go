package settings

// Options contains loading options for settings
type Options struct {
	prefix string
}

// WithPrefix adds the given prefix when loading settings
func WithPrefix(prefix string) func(*Options) {
	return func(o *Options) {
		o.prefix = prefix
	}
}
