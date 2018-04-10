package types

// AppMetadata is the format of the data found inside the metadata.yml file
type AppMetadata struct {
	Version     string
	Application ApplicationInfo
	Targets     ApplicationTarget
}

// ApplicationInfo represents general user-provided information about
// a given app package
type ApplicationInfo struct {
	Name        string
	Description string
	Tag         string
	Labels      []string
	Author      string
}

// ApplicationTarget represents which platform(s) / orchestrator(s) the
// app package is designed to run on
type ApplicationTarget struct {
	Swarm      bool
	Kubernetes bool
}
