package inspect

import (
	"fmt"
	"sort"
	"strings"

	composetypes "github.com/docker/cli/cli/compose/types"
)

type portRange struct {
	start uint32
	end   *uint32
}

func newPort(start uint32) *portRange {
	return &portRange{start: start}
}

func (p *portRange) add(end uint32) bool {
	if p.end == nil {
		if p.start+1 == end {
			p.end = &end
			return true
		}
		return false
	}
	if *p.end+1 == end {
		p.end = &end
		return true

	}
	return false
}

func (p portRange) String() string {
	res := fmt.Sprintf("%d", p.start)
	if p.end != nil {
		res += fmt.Sprintf("-%d", *p.end)
	}
	return res
}

// getPorts identifies all the published port ranges, merges them
// if they are consecutive, and return a string with all the published
// ports.
func getPorts(ports []composetypes.ServicePortConfig) string {
	var (
		portRanges    []*portRange
		lastPortRange *portRange
	)
	sort.Slice(ports, func(i int, j int) bool { return ports[i].Published < ports[j].Published })
	for _, port := range ports {
		if port.Published > 0 {
			if lastPortRange == nil {
				lastPortRange = newPort(port.Published)
			} else if !lastPortRange.add(port.Published) {
				portRanges = append(portRanges, lastPortRange)
				lastPortRange = newPort(port.Published)
			}
		}
	}
	if lastPortRange != nil {
		portRanges = append(portRanges, lastPortRange)
	}
	output := make([]string, len(portRanges))
	for i, p := range portRanges {
		output[i] = p.String()
	}
	return strings.Join(output, ",")
}
