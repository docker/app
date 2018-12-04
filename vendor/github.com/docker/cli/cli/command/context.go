package command

import (
	"github.com/docker/cli/cli/context/store"
)

// ContextMetadata is a typed representation of what we put in Context metadata
type ContextMetadata struct {
	Description       string
	StackOrchestrator Orchestrator
}

// SetContextMetadata sets the metadata inside a stored context
func SetContextMetadata(ctx *store.ContextMetadata, metadata ContextMetadata) {
	ctx.Metadata = map[string]interface{}{
		"description":              metadata.Description,
		"defaultStackOrchestrator": string(metadata.StackOrchestrator),
	}
}

// GetContextMetadata extracts metadata from stored context metadata
func GetContextMetadata(ctx store.ContextMetadata) (ContextMetadata, error) {
	var result ContextMetadata
	if ctx.Metadata == nil {
		return result, nil
	}
	var err error
	if val, ok := ctx.Metadata["description"]; ok {
		result.Description, _ = val.(string)
	}
	if val, ok := ctx.Metadata["defaultStackOrchestrator"]; ok {
		v, _ := val.(string)
		if result.StackOrchestrator, err = normalize(v); err != nil {
			return result, err
		}
	}
	return result, nil
}
