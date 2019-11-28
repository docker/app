package rules

type Rule interface {
	Collect(path string, key string, value interface{})
	Accept(parent string, key string) bool
	Validate(value interface{}) []error
}
