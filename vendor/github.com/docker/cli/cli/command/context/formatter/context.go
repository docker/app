package formatter

import (
	"github.com/docker/cli/cli/command/formatter"
)

const (
	// ClientContextTableFormat is the default client context format
	ClientContextTableFormat = "table {{.NameWithCurrent}}\t{{.Description}}\t{{.DockerEndpoint}}\t{{.KubernetesEndpoint}}\t{{.StackOrchestrator}}"

	dockerEndpointHeader     = "DOCKER ENDPOINT"
	kubernetesEndpointHeader = "KUBERNETES ENDPOINT"
	stackOrchestrastorHeader = "ORCHESTRATOR"
)

// ClientContext is a context for display
type ClientContext struct {
	Name               string
	Description        string
	DockerEndpoint     string
	KubernetesEndpoint string
	StackOrchestrator  string
	Current            bool
}

// ClientContextWrite writes formatted contexts using the Context
func ClientContextWrite(ctx formatter.Context, contexts []*ClientContext) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, context := range contexts {
			if err := format(&clientContextContext{c: context}); err != nil {
				return err
			}
		}
		return nil
	}
	return ctx.Write(newClientContextContext(), render)
}

type clientContextContext struct {
	formatter.HeaderContext
	c *ClientContext
}

func newClientContextContext() *clientContextContext {
	ctx := clientContextContext{}
	ctx.Header = formatter.SubHeaderContext{
		"NameWithCurrent":    formatter.NameHeader,
		"Description":        formatter.DescriptionHeader,
		"DockerEndpoint":     dockerEndpointHeader,
		"KubernetesEndpoint": kubernetesEndpointHeader,
		"StackOrchestrator":  stackOrchestrastorHeader,
	}
	return &ctx
}

func (c *clientContextContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(c)
}

func (c *clientContextContext) NameWithCurrent() string {
	if !c.c.Current {
		return c.c.Name
	}
	return c.c.Name + " *"
}

func (c *clientContextContext) Description() string {
	return c.c.Description
}

func (c *clientContextContext) DockerEndpoint() string {
	return c.c.DockerEndpoint
}

func (c *clientContextContext) KubernetesEndpoint() string {
	return c.c.KubernetesEndpoint
}

func (c *clientContextContext) StackOrchestrator() string {
	return c.c.StackOrchestrator
}
