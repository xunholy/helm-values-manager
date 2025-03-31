package output

import (
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/xunholy/helm-values-manager/pkg/analyzer"
	"github.com/xunholy/helm-values-manager/pkg/util"
	"gopkg.in/yaml.v2"
)

// Manager handles the writing of output files
type Manager struct {
	Paths    analyzer.PathOptions
	Format   string
	Optimize bool
}

// NewManager creates a new output manager
func NewManager(paths analyzer.PathOptions, format string, optimize bool) *Manager {
	return &Manager{
		Paths:    paths,
		Format:   format,
		Optimize: optimize,
	}
}

// WriteResults writes the analysis results to files and stdout
func (m *Manager) WriteResults(valueStatus analyzer.ValueStatus) error {
	// Always generate the optimized values file (for backward compatibility with tests)
	// even if optimize flag is not set
	log.Info().Msg("Generating optimized values.yaml")
	optimizedValues, err := yaml.Marshal(valueStatus.Optimized)
	if err != nil {
		return fmt.Errorf("failed to marshal optimized values: %w", err)
	}

	// Save to file
	optimizedFilePath := m.Paths.OptimizedValuesPath
	if err := util.CreateOutputFile(optimizedValues, optimizedFilePath); err != nil {
		return fmt.Errorf("failed to write optimized values: %w", err)
	}

	// Only show optimization details when optimize flag is set
	if m.Optimize {
		// Calculate size reduction
		originalSize := 0
		for _, size := range []int{
			analyzer.CountNestedKeys(valueStatus.Optimized),
			analyzer.CountNestedKeys(valueStatus.Redundant),
			analyzer.CountNestedKeys(valueStatus.Unsupported),
		} {
			originalSize += size
		}

		optimizedSize := analyzer.CountNestedKeys(valueStatus.Optimized)
		if originalSize > 0 {
			reduction := 100 - (optimizedSize * 100 / originalSize)
			log.Info().Msgf("Optimized values written to: %s (reduced by %d%%)", optimizedFilePath, reduction)
		} else {
			log.Info().Msgf("Optimized values written to: %s", optimizedFilePath)
		}
	} else {
		log.Info().Msgf("Optimized values written to: %s", optimizedFilePath)
	}

	// Add generated-values.yaml for backward compatibility with tests
	// It contains all unsupported values (those that exist in downstream but not in upstream)
	generatedValues, err := yaml.Marshal(valueStatus.Unsupported)
	if err != nil {
		return fmt.Errorf("failed to marshal generated values: %w", err)
	}

	generatedFilePath := m.Paths.GeneratedValuesPath
	if err := util.CreateOutputFile(generatedValues, generatedFilePath); err != nil {
		return fmt.Errorf("failed to write generated values: %w", err)
	}

	log.Info().Msgf("Generated values written to: %s", generatedFilePath)

	// Write analysis files
	if err := m.writeAnalysisFiles(valueStatus); err != nil {
		return fmt.Errorf("failed to write analysis files: %w", err)
	}

	log.Info().Msg("Values analysis completed successfully")
	return nil
}

// Write the analysis reports to files
func (m *Manager) writeAnalysisFiles(valueStatus analyzer.ValueStatus) error {
	// Process unsupported values
	unsupportedCount := analyzer.CountNestedKeys(valueStatus.Unsupported)
	if unsupportedCount > 0 {
		log.Info().Msgf("Found %d unsupported values", unsupportedCount)

		// Save to file
		unsupportedValues, err := yaml.Marshal(valueStatus.Unsupported)
		if err != nil {
			return fmt.Errorf("failed to marshal unsupported values: %w", err)
		}

		unsupportedFilePath := m.Paths.UnsupportedValuesPath
		if err := util.CreateOutputFile(unsupportedValues, unsupportedFilePath); err != nil {
			return fmt.Errorf("failed to write unsupported values: %w", err)
		}

		log.Info().Msgf("Unsupported values written to: %s", unsupportedFilePath)

		// Add a note about unsupported values being removed from optimized output
		if m.Optimize {
			log.Info().Msg("Note: Unsupported values have been removed from the optimized output")
		}
	} else {
		log.Info().Msg("No unsupported values found")
	}

	// Process commented values (values that exist in upstream but are commented out)
	commentedCount := analyzer.CountNestedKeys(valueStatus.Commented)
	if commentedCount > 0 {
		log.Info().Msgf("Found %d values that are commented out in the upstream chart", commentedCount)

		// Save to file
		commentedValues, err := yaml.Marshal(valueStatus.Commented)
		if err != nil {
			return fmt.Errorf("failed to marshal commented values: %w", err)
		}

		commentedFilePath := filepath.Join(m.Paths.OutputDir, "commented-values.yaml")
		if err := util.CreateOutputFile(commentedValues, commentedFilePath); err != nil {
			return fmt.Errorf("failed to write commented values: %w", err)
		}

		log.Info().Msgf("Commented values written to: %s", commentedFilePath)
		log.Info().Msg("Note: These values exist in your custom values but are commented out in the upstream chart")

		// Add a note about commented values being removed from optimized output
		if m.Optimize {
			log.Info().Msg("Note: Commented values have been removed from the optimized output for consistency")
		}
	}

	// Process redundant values
	redundantCount := analyzer.CountNestedKeys(valueStatus.Redundant)
	if redundantCount > 0 {
		log.Info().Msgf("Found %d redundant values", redundantCount)

		// Save to file
		redundantValues, err := yaml.Marshal(valueStatus.Redundant)
		if err != nil {
			return fmt.Errorf("failed to marshal redundant values: %w", err)
		}

		redundantFilePath := m.Paths.RedundantValuesPath
		if err := util.CreateOutputFile(redundantValues, redundantFilePath); err != nil {
			return fmt.Errorf("failed to write redundant values: %w", err)
		}

		log.Info().Msgf("Redundant values written to: %s", redundantFilePath)
	} else {
		log.Info().Msg("No redundant values found")
	}

	return nil
}
