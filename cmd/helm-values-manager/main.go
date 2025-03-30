package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xunholy/helm-values-manager/pkg/analyzer"
	"github.com/xunholy/helm-values-manager/pkg/helm"
	"github.com/xunholy/helm-values-manager/pkg/output"
	"github.com/xunholy/helm-values-manager/pkg/util"
	"gopkg.in/yaml.v2"
)

// Command line flags
var (
	repo                 string
	chartName            string
	chartVersion         string
	kubeConfigFile       string
	context              string
	namespace            string
	revision             int
	outputFormat         string
	upstreamValuesFile   string
	downstreamValuesFile string
	outDir               string
	optimize             bool
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	defaultKubeConfigPath, err := util.FindKubeConfig()
	if err != nil {
		log.Warn().AnErr("kubeConfigPath", err).Msg("Unable to determine default kubeconfig path")
	}

	// Command-line flag definitions
	flag.StringVar(&repo, "repo", "", "chart repository url where to locate the requested chart")
	flag.StringVar(&chartName, "chart", "", "name of the Helm chart to fetch upstream values from")
	flag.StringVar(&chartVersion, "chart-version", "", "specific version of the Helm chart")
	flag.IntVar(&revision, "revision", 0, "specify a revision constraint for the chart revision to use")
	flag.StringVar(&kubeConfigFile, "kubeconfig", defaultKubeConfigPath, "path to the kubeconfig file")
	flag.StringVar(&context, "kube-context", "", "name of the kubeconfig context to use")
	flag.StringVar(&namespace, "namespace", "", "namespace scope for this request")
	flag.StringVar(&outputFormat, "output", "stdout", "output format. One of: (yaml,stdout)")
	flag.StringVar(&upstreamValuesFile, "upstream", "", "path to the upstream values.yaml file")
	flag.StringVar(&downstreamValuesFile, "downstream", "", "path to the downstream values.yaml file")
	flag.StringVar(&outDir, "outdir", "values-analysis", "directory to store output files")
	flag.BoolVar(&optimize, "optimize", false, "optimize values.yaml by removing redundant values")
}

func main() {
	flag.Parse()

	// Create output directory if it doesn't exist
	if err := util.EnsureDirectory(outDir); err != nil {
		log.Fatal().Err(err).Msgf("failed to create output directory: %s", outDir)
	}

	// Configure output paths
	paths := analyzer.NewPathOptions(outDir)

	// Determine source of upstream values
	upstreamPath := ""
	var upstreamValues map[string]interface{}
	var originalUpstreamYAML []byte
	var err error

	// Option 1: Use provided upstream file if specified
	if upstreamValuesFile != "" {
		log.Info().Msgf("Using provided upstream values file: %s", upstreamValuesFile)
		upstreamPath = upstreamValuesFile

		// Load upstream values
		originalUpstreamYAML, err = os.ReadFile(upstreamValuesFile)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to read upstream values file: %s", upstreamValuesFile)
		}

		if err := yaml.Unmarshal(originalUpstreamYAML, &upstreamValues); err != nil {
			log.Fatal().Err(err).Msg("failed to parse upstream values YAML")
		}
	} else if chartName != "" {
		// Option 2: Fetch upstream values from a Helm chart
		log.Info().Msgf("Fetching upstream values from chart: %s", chartName)

		// Special handling for known charts with lots of commented fields
		if strings.Contains(chartName, "cilium") {
			log.Warn().Msg("Note: The cilium chart has many commented values. For best results with cilium, use:")
			log.Warn().Msgf("helm show values %s > cilium-values.yaml", chartName)
			log.Warn().Msg("Then run: helm-values-manager --upstream cilium-values.yaml --downstream your-values.yaml")
		}

		// First get the raw YAML content to preserve comments
		var helmErr error
		originalUpstreamYAML, helmErr = helm.FetchChartValuesRaw(chartName, chartVersion)
		if helmErr != nil {
			log.Warn().Err(helmErr).Msg("Unable to fetch raw values YAML from Helm chart, comments will not be preserved")
			// Fallback to the regular method
			upstreamValues, err = helm.FetchChartValues(chartName, chartVersion)
			if err != nil {
				log.Fatal().Err(err).Msg("Unable to fetch values from Helm chart")
			}
		} else {
			// Parse the YAML content for processing
			if err := yaml.Unmarshal(originalUpstreamYAML, &upstreamValues); err != nil {
				log.Warn().Err(err).Msg("Error parsing raw YAML, fallback to regular fetch")
				// Fallback to the regular method
				upstreamValues, err = helm.FetchChartValues(chartName, chartVersion)
				if err != nil {
					log.Fatal().Err(err).Msg("Unable to fetch values from Helm chart")
				}
				// Clear the original YAML since it couldn't be parsed
				originalUpstreamYAML = nil
			}
		}

		// Save chart values to file
		upstreamPath = filepath.Join(outDir, "chart-values.yaml")
		// Save the original YAML if available, otherwise marshal from map
		var contentToSave []byte
		if originalUpstreamYAML != nil {
			contentToSave = originalUpstreamYAML
		} else {
			contentToSave, err = yaml.Marshal(upstreamValues)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to marshal chart values")
			}
		}

		if err := util.CreateOutputFile(contentToSave, upstreamPath); err != nil {
			log.Fatal().Err(err).Msg("Failed to write chart values to file")
		}
	} else if repo != "" {
		// Option 3: Use Helm release values
		log.Info().Msgf("Fetching values from Helm release: %s", repo)
		helmClient, err := helm.NewClient(context, namespace, kubeConfigFile)
		if err != nil {
			log.Fatal().Err(err).Msg("fetching helm client")
		}

		upstreamValues, err = helmClient.FetchReleaseValues(repo, revision)
		if err != nil {
			log.Fatal().Err(err).Msg("fetching helm repo")
		}

		// Save release values to file
		upstreamPath = filepath.Join(outDir, "upstream-values.yaml")
		marshaledValues, err := yaml.Marshal(upstreamValues)
		if err != nil {
			log.Fatal().Err(err).Msg("error while marshaling upstream values")
		}

		if err := util.CreateOutputFile(marshaledValues, upstreamPath); err != nil {
			log.Fatal().Err(err).Msg("unable to write upstream values file")
		}
	} else {
		// If no upstream source is provided, show usage
		log.Error().Msg("No upstream values source specified. Use one of: -upstream, -chart, or -repo")
		flag.Usage()
		os.Exit(2)
	}

	// We need a downstream values file to compare against
	if downstreamValuesFile == "" {
		log.Error().Msg("missing -downstream flag")
		flag.Usage()
		os.Exit(2)
	}

	// Load downstream values
	downstreamContent, err := os.ReadFile(downstreamValuesFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to read downstream values file: %s", downstreamValuesFile)
	}

	var downstreamValues map[string]interface{}
	if err := yaml.Unmarshal(downstreamContent, &downstreamValues); err != nil {
		log.Fatal().Err(err).Msg("failed to parse downstream values YAML")
	}

	// Process the values
	processValues(upstreamValues, downstreamValues, paths, originalUpstreamYAML)
}

// processValues analyzes upstream and downstream values and generates reports
func processValues(upstreamValues, downstreamValues map[string]interface{}, paths analyzer.PathOptions, originalUpstreamYAML []byte) {
	log.Info().Msg("Processing upstream and downstream values")

	// Create analyzer with original YAML for better analysis
	var valueAnalyzer *analyzer.Analyzer
	if originalUpstreamYAML != nil && len(originalUpstreamYAML) > 0 {
		valueAnalyzer = analyzer.NewAnalyzerWithOriginalYAML(upstreamValues, downstreamValues, originalUpstreamYAML)
		log.Info().Msg("Using original YAML for enhanced comment detection")
	} else {
		valueAnalyzer = analyzer.NewAnalyzer(upstreamValues, downstreamValues)
		log.Info().Msg("No original YAML available, comment detection will be limited")
	}

	// Analyze values
	valueStatus := valueAnalyzer.Analyze()

	// Create output manager and write results
	outputMgr := output.NewManager(paths, outputFormat, optimize)
	if err := outputMgr.WriteResults(valueStatus); err != nil {
		log.Fatal().Err(err).Msg("Failed to write analysis results")
	}
}
