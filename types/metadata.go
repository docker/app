package types

// AppMetadata is the format of the data found inside the metadata.yml file
type AppMetadata struct {
	Version     string
	Name        string
	Description string
	Author      string
	Targets     ApplicationTarget
}

// ApplicationTarget represents which platform(s) / orchestrator(s) the
// app package is designed to run on
type ApplicationTarget struct {
	Swarm      bool
	Kubernetes bool
}
