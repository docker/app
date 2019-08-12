package remotes

import (
	"sync"

	"github.com/docker/distribution/reference"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// FixupEvent is an event that is raised by the Fixup Logic
type FixupEvent struct {
	SourceImage    string
	DestinationRef reference.Named
	EventType      FixupEventType
	Message        string
	Error          error
	Progress       ProgressSnapshot
}

// FixupEventType is the the type of event raised by the Fixup logic
type FixupEventType string

const (
	// FixupEventTypeCopyImageStart is raised when the Fixup logic starts copying an
	// image
	FixupEventTypeCopyImageStart = FixupEventType("CopyImageStart")

	// FixupEventTypeCopyImageEnd is raised when the Fixup logic stops copying an
	// image. Error might be populated
	FixupEventTypeCopyImageEnd = FixupEventType("CopyImageEnd")

	// FixupEventTypeProgress is raised when Fixup logic reports progression
	FixupEventTypeProgress = FixupEventType("Progress")
)

type descriptorProgress struct {
	ocischemav1.Descriptor
	done     bool
	action   string
	err      error
	children []*descriptorProgress
	mut      sync.RWMutex
}

func (p *descriptorProgress) markDone() {
	p.mut.Lock()
	defer p.mut.Unlock()
	p.done = true
}

func (p *descriptorProgress) setAction(a string) {
	p.mut.Lock()
	defer p.mut.Unlock()
	p.action = a
}

func (p *descriptorProgress) setError(err error) {
	p.mut.Lock()
	defer p.mut.Unlock()
	p.err = err
}

func (p *descriptorProgress) addChild(child *descriptorProgress) {
	p.mut.Lock()
	defer p.mut.Unlock()
	p.children = append(p.children, child)
}

func (p *descriptorProgress) snapshot() DescriptorProgressSnapshot {
	p.mut.RLock()
	defer p.mut.RUnlock()
	result := DescriptorProgressSnapshot{
		Descriptor: p.Descriptor,
		Done:       p.done,
		Action:     p.action,
		Error:      p.err,
	}
	if len(p.children) != 0 {
		result.Children = make([]DescriptorProgressSnapshot, len(p.children))
		for ix, child := range p.children {
			result.Children[ix] = child.snapshot()
		}
	}
	return result
}

type progress struct {
	roots []*descriptorProgress
	mut   sync.RWMutex
}

func (p *progress) addRoot(root *descriptorProgress) {
	p.mut.Lock()
	defer p.mut.Unlock()
	p.roots = append(p.roots, root)
}

func (p *progress) snapshot() ProgressSnapshot {
	p.mut.RLock()
	defer p.mut.RUnlock()
	result := ProgressSnapshot{}
	if len(p.roots) != 0 {
		result.Roots = make([]DescriptorProgressSnapshot, len(p.roots))
		for ix, root := range p.roots {
			result.Roots[ix] = root.snapshot()
		}
	}
	return result
}

// DescriptorProgressSnapshot describes the current progress of a descriptor
type DescriptorProgressSnapshot struct {
	ocischemav1.Descriptor
	Done     bool
	Action   string
	Error    error
	Children []DescriptorProgressSnapshot
}

// ProgressSnapshot describes the current progress of a Fixup operation
type ProgressSnapshot struct {
	Roots []DescriptorProgressSnapshot
}
