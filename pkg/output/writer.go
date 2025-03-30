package output

import (
	"fmt"

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

// WriteResults writes all the analysis results to files
func (m *Manager) WriteResults(status analyzer.ValueStatus, generatedValues map[string]interface{}) error {
	// Always generate the generated-values.yaml file for compatibility
	generatedContent, err := yaml.Marshal(generatedValues)
	if err != nil {
		log.Error().Err(err).Msg("error while marshaling generated values")
		return err
	}

	if err = util.CreateOutputFile(generatedContent, m.Paths.GeneratedValuesPath); err != nil {
		log.Error().Err(err).Msg("unable to write generated values file")
		return err
	}

	log.Info().Msgf("Generated values written to: %s", m.Paths.GeneratedValuesPath)

	// Always generate optimized values (downstream without redundant values)
	// even if optimize flag is not set, for compatibility with tests
	if err := m.writeOptimizedValues(status); err != nil {
		return err
	}

	// Write analysis files
	if err := m.writeAnalysisFiles(status); err != nil {
		return err
	}

	log.Info().Msg("Values analysis completed successfully")
	return nil
}

// writeOptimizedValues writes the optimized values to a file
func (m *Manager) writeOptimizedValues(status analyzer.ValueStatus) error {
	log.Info().Msg("Generating optimized values.yaml")
	optimizedContent, err := yaml.Marshal(status.Optimized)
	if err != nil {
		log.Error().Err(err).Msg("error while marshaling optimized values")
		return err
	}

	if err = util.CreateOutputFile(optimizedContent, m.Paths.OptimizedValuesPath); err != nil {
		log.Error().Err(err).Msg("unable to write optimized values file")
		return err
	}

	log.Info().Msgf("Optimized values written to: %s", m.Paths.OptimizedValuesPath)
	return nil
}

// writeAnalysisFiles writes the unsupported and redundant values to files
func (m *Manager) writeAnalysisFiles(status analyzer.ValueStatus) error {
	// Output unsupported values
	if len(status.Unsupported) > 0 {
		count := analyzer.CountNestedKeys(status.Unsupported)
		log.Info().Msgf("Found %d unsupported values", count)
		unsupportedContent, err := yaml.Marshal(status.Unsupported)
		if err != nil {
			log.Error().Err(err).Msg("error while marshaling unsupported values")
			return err
		}

		if err = util.CreateOutputFile(unsupportedContent, m.Paths.UnsupportedValuesPath); err != nil {
			log.Error().Err(err).Msg("unable to write unsupported values file")
			return err
		}

		log.Info().Msgf("Unsupported values written to: %s", m.Paths.UnsupportedValuesPath)
	} else {
		log.Info().Msg("No unsupported values found")
	}

	// Output redundant values
	if len(status.Redundant) > 0 {
		count := analyzer.CountNestedKeys(status.Redundant)
		log.Info().Msgf("Found %d redundant values", count)
		redundantContent, err := yaml.Marshal(status.Redundant)
		if err != nil {
			log.Error().Err(err).Msg("error while marshaling redundant values")
			return err
		}

		if err = util.CreateOutputFile(redundantContent, m.Paths.RedundantValuesPath); err != nil {
			log.Error().Err(err).Msg("unable to write redundant values file")
			return err
		}

		log.Info().Msgf("Redundant values written to: %s", m.Paths.RedundantValuesPath)
		if m.Format == "stdout" {
			fmt.Println("\nRedundant values:")
			fmt.Println(string(redundantContent))
		}
	} else {
		log.Info().Msg("No redundant values found")
	}

	return nil
}
