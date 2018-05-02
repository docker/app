package types

// Maintainer represents one of the apps's maintainers
type Maintainer struct {
	Name  string
	Email string
}

// AppMetadata is the format of the data found inside the metadata.yml file
type AppMetadata struct {
	Version     string
	Name        string
	Description string
	Maintainers []Maintainer
	Targets     ApplicationTarget
}

// ApplicationTarget represents which platform(s) / orchestrator(s) the
// app package is designed to run on
type ApplicationTarget struct {
	Swarm      bool
	Kubernetes bool
}
