package analyzer

import (
	"reflect"
)

// ValueStatus stores analysis results
type ValueStatus struct {
	Redundant   map[string]interface{} `yaml:"redundant,omitempty"`
	Unsupported map[string]interface{} `yaml:"unsupported,omitempty"`
	Optimized   map[string]interface{} `yaml:"optimized,omitempty"`
}

// ChangeRequest represents a value that needs to be modified
type ChangeRequest struct {
	Path    string
	Content reflect.Value
}

// Changes represents a collection of changes
type Changes struct {
	Items []*ChangeRequest
}

// PathOptions contains paths for output files
type PathOptions struct {
	OutputDir             string
	OptimizedValuesPath   string
	UnsupportedValuesPath string
	RedundantValuesPath   string
}

// NewPathOptions creates a new PathOptions with the given output directory
func NewPathOptions(outputDir string) PathOptions {
	return PathOptions{
		OutputDir:             outputDir,
		OptimizedValuesPath:   outputDir + "/optimized-values.yaml",
		UnsupportedValuesPath: outputDir + "/unsupported-values.yaml",
		RedundantValuesPath:   outputDir + "/redundant-values.yaml",
	}
}
