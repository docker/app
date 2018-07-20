package slices

// ContainsString checks wether the given string is in the specified slice
func ContainsString(strings []string, s string) bool {
	for _, e := range strings {
		if e == s {
			return true
		}
	}
	return false
}
