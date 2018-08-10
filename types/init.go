package types

const defaultComposefileVersion = "3.6"

// InitialComposeFile represents an initial composefile (used by the init command)
type InitialComposeFile struct {
	Version  string
	Services map[string]InitialService
}

// InitialService represents an initial service (used by the init command)
type InitialService struct {
	Image string
}

// NewInitialComposeFile returns an empty InitialComposeFile object
func NewInitialComposeFile() InitialComposeFile {
	return InitialComposeFile{
		Version:  defaultComposefileVersion,
		Services: map[string]InitialService{},
	}
}
