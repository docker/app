package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/docker/app/internal"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/opts"
	"github.com/docker/distribution/reference"
	swarmtypes "github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/stringid"
	"github.com/pkg/errors"
)

var (
	listColumns = []struct {
		header string
		value  func(s *swarmtypes.Service) string
	}{
		{"ID", func(s *swarmtypes.Service) string { return stringid.TruncateID(s.ID) }},
		{"NAME", func(s *swarmtypes.Service) string { return s.Spec.Name }},
		{"MODE", func(s *swarmtypes.Service) string {
			if s.Spec.Mode.Replicated != nil {
				return "replicated"
			}
			return "global"
		}},
		{"REPLICAS", func(s *swarmtypes.Service) string {
			if s.Spec.Mode.Replicated != nil {
				return fmt.Sprintf("%d/%d", s.ServiceStatus.RunningTasks, s.ServiceStatus.DesiredTasks)
			}
			return ""
		}},
		{"IMAGE", func(s *swarmtypes.Service) string {
			ref, err := reference.ParseAnyReference(s.Spec.TaskTemplate.ContainerSpec.Image)
			if err != nil {
				return "N/A"
			}
			if namedRef, ok := ref.(reference.Named); ok {
				return reference.FamiliarName(namedRef)
			}
			return reference.FamiliarString(ref)
		}},
		{"PORTS", func(s *swarmtypes.Service) string {
			return Ports(s.Endpoint.Ports)
		}},
	}
)

type portRange struct {
	pStart   uint32
	pEnd     uint32
	tStart   uint32
	tEnd     uint32
	protocol swarmtypes.PortConfigProtocol
}

func (pr portRange) String() string {
	var (
		pub string
		tgt string
	)

	if pr.pEnd > pr.pStart {
		pub = fmt.Sprintf("%d-%d", pr.pStart, pr.pEnd)
	} else {
		pub = fmt.Sprintf("%d", pr.pStart)
	}
	if pr.tEnd > pr.tStart {
		tgt = fmt.Sprintf("%d-%d", pr.tStart, pr.tEnd)
	} else {
		tgt = fmt.Sprintf("%d", pr.tStart)
	}
	return fmt.Sprintf("*:%s->%s/%s", pub, tgt, pr.protocol)
}

// Ports formats port configuration. This function is copied et adapted from docker CLI
// see https://github.com/docker/cli/blob/d6edc912ce/cli/command/service/formatter.go#L655
func Ports(servicePorts []swarmtypes.PortConfig) string {
	if servicePorts == nil {
		return ""
	}

	pr := portRange{}
	ports := []string{}

	sort.Slice(servicePorts, func(i, j int) bool {
		if servicePorts[i].Protocol == servicePorts[j].Protocol {
			return servicePorts[i].PublishedPort < servicePorts[j].PublishedPort
		}
		return servicePorts[i].Protocol < servicePorts[j].Protocol
	})

	for _, p := range servicePorts {
		if p.PublishMode == swarmtypes.PortConfigPublishModeIngress {
			prIsRange := pr.tEnd != pr.tStart
			tOverlaps := p.TargetPort <= pr.tEnd

			// Start a new port-range if:
			// - the protocol is different from the current port-range
			// - published or target port are not consecutive to the current port-range
			// - the current port-range is a _range_, and the target port overlaps with the current range's target-ports
			if p.Protocol != pr.protocol || p.PublishedPort-pr.pEnd > 1 || p.TargetPort-pr.tEnd > 1 || prIsRange && tOverlaps {
				// start a new port-range, and print the previous port-range (if any)
				if pr.pStart > 0 {
					ports = append(ports, pr.String())
				}
				pr = portRange{
					pStart:   p.PublishedPort,
					pEnd:     p.PublishedPort,
					tStart:   p.TargetPort,
					tEnd:     p.TargetPort,
					protocol: p.Protocol,
				}
				continue
			}
			pr.pEnd = p.PublishedPort
			pr.tEnd = p.TargetPort
		}
	}
	if pr.pStart > 0 {
		ports = append(ports, pr.String())
	}
	return strings.Join(ports, ", ")
}

func statusAction(instanceName string) error {
	cli, err := getCli()
	if err != nil {
		return err
	}
	services, _ := runningServices(cli, instanceName)
	if err := printServices(cli.Out(), services); err != nil {
		return err
	}
	return nil
}

func statusJSONAction(instanceName string) error {
	cli, err := getCli()
	if err != nil {
		return err
	}
	services, _ := runningServices(cli, instanceName)
	js, err := json.MarshalIndent(services, "", "    ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cli.Out(), string(js))
	return nil
}

func getCli() (command.Cli, error) {
	cli, err := setupDockerContext()
	if err != nil {
		return nil, errors.Wrap(err, "unable to restore docker context")
	}
	return cli, nil
}

func runningServices(cli command.Cli, instanceName string) ([]swarmtypes.Service, error) {
	orchestratorRaw := os.Getenv(internal.DockerStackOrchestratorEnvVar)
	orchestrator, err := cli.StackOrchestrator(orchestratorRaw)
	if err != nil {
		return nil, err
	}
	return stack.GetServices(cli, getFlagset(orchestrator), orchestrator, options.Services{
		Filter:    opts.NewFilterOpt(),
		Namespace: instanceName,
	})
}

func printServices(out io.Writer, services []swarmtypes.Service) error {
	w := tabwriter.NewWriter(out, 0, 0, 1, ' ', 0)
	printHeaders(w)

	for _, service := range services {
		printValues(w, &service)
	}
	return w.Flush()
}

func printHeaders(w io.Writer) {
	var headers []string
	for _, column := range listColumns {
		headers = append(headers, column.header)
	}
	fmt.Fprintln(w, strings.Join(headers, "\t"))
}

func printValues(w io.Writer, service *swarmtypes.Service) {
	var values []string
	for _, column := range listColumns {
		values = append(values, column.value(service))
	}
	fmt.Fprintln(w, strings.Join(values, "\t"))
}
