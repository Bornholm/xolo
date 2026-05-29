package model

import "time"

// PipelineBundle is the portable export format for a virtual model pipeline.
// It embeds the pipeline graph along with the VM metadata so the file is
// self-contained and can be used to recreate a VM from scratch.
type PipelineBundle struct {
	Version     string         `json:"version"`
	ExportedAt  time.Time      `json:"exportedAt"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Graph       *PipelineGraph `json:"graph"`
}
