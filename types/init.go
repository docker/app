package types

// InitialComposeFile TODO
type InitialComposeFile struct {
    Version  string
    Services *map[string]InitialService
}

// InitialService TODO
type InitialService struct {
    Image string
}

// NewInitialComposeFile returns an empty InitialComposeFile object
func NewInitialComposeFile() InitialComposeFile {
    services := make(map[string]InitialService)

    return InitialComposeFile{
        Version:  "3.6",
        Services: &services,
    }
}
