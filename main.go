package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/gonvenience/ytbx"
	"github.com/homeport/dyff/pkg/dyff"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
	"github.com/stretchr/objx"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// Command line flags
var (
	repo                  string
	chartName             string
	chartVersion          string
	kubeConfigFile        string
	context               string
	namespace             string
	revision              int
	output                string
	upstreamValuesFile    string
	downstreamValuesFile  string
	outDir                string
	optimize              bool
	outputFilePath        string
	optimizedValuesPath   string
	unsupportedValuesPath string
	redundantValuesPath   string
	settings              = cli.New()
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

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	defaultKubeConfigPath, err := findKubeConfig()
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
	flag.StringVar(&output, "output", "stdout", "output format. One of: (yaml,stdout)")
	flag.StringVar(&upstreamValuesFile, "upstream", "", "path to the upstream values.yaml file")
	flag.StringVar(&downstreamValuesFile, "downstream", "", "path to the downstream values.yaml file")
	flag.StringVar(&outDir, "outdir", "values-analysis", "directory to store output files")
	flag.BoolVar(&optimize, "optimize", false, "optimize values.yaml by removing redundant values")
}

func main() {
	flag.Parse()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatal().Err(err).Msgf("failed to create output directory: %s", outDir)
	}

	// Set output file paths with custom directory
	updateOutputPaths()

	// Determine source of upstream values
	upstreamPath := ""

	// Option 1: Use provided upstream file if specified
	if upstreamValuesFile != "" {
		log.Info().Msgf("Using provided upstream values file: %s", upstreamValuesFile)
		upstreamPath = upstreamValuesFile
	} else if chartName != "" {
		// Option 2: Fetch upstream values from a Helm chart
		log.Info().Msgf("Fetching upstream values from chart: %s", chartName)
		upstreamValues, err := fetchHelmChartValues(chartName, chartVersion)
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to fetch values from Helm chart")
		}

		upstreamPath = filepath.Join(outDir, "chart-values.yaml")
		marshaledValues, err := yaml.Marshal(upstreamValues)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to marshal chart values")
		}

		if err := CreateOutputFile(marshaledValues, upstreamPath); err != nil {
			log.Fatal().Err(err).Msg("Failed to write chart values to file")
		}
	} else if repo != "" {
		// Option 3: Use Helm release values
		log.Info().Msgf("Fetching values from Helm release: %s", repo)
		helm, err := NewHelmClient()
		if err != nil {
			log.Fatal().Err(err).Msg("fetching helm client")
		}

		rv, err := HelmFetch(helm)
		if err != nil {
			log.Fatal().Err(err).Msg("fetching helm repo")
		}

		upstreamPath = filepath.Join(outDir, "upstream-values.yaml")
		upstreamValues, err := yaml.Marshal(rv)
		if err != nil {
			log.Fatal().Err(err).Msg("error while marshaling upstream values")
		}

		if err := CreateOutputFile(upstreamValues, upstreamPath); err != nil {
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

	// Process the values files
	processValuesFiles(upstreamPath, downstreamValuesFile)
}

// fetchHelmChartValues gets values from a Helm chart repository
func fetchHelmChartValues(chartName, version string) (map[string]interface{}, error) {
	log.Info().Msgf("Fetching values from Helm chart: %s (version: %s)", chartName, version)

	// First check if it's a local file and try to load it as a values file
	if _, err := os.Stat(chartName); err == nil {
		// If it's a YAML file, just load it as values
		if strings.HasSuffix(chartName, ".yaml") || strings.HasSuffix(chartName, ".yml") {
			log.Info().Msgf("Loading values from YAML file: %s", chartName)
			yamlContent, err := ioutil.ReadFile(chartName)
			if err != nil {
				return nil, fmt.Errorf("failed to read values file: %w", err)
			}

			var values map[string]interface{}
			if err := yaml.Unmarshal(yamlContent, &values); err != nil {
				return nil, fmt.Errorf("failed to parse values file: %w", err)
			}

			return values, nil
		}

		// Otherwise, try to load as a chart
		return loadLocalChartValues(chartName)
	}

	// Otherwise, try to fetch from a repository
	settings := cli.New()
	actionConfig := new(action.Configuration)
	err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		log.Debug().Msgf(format, v...)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Helm configuration: %w", err)
	}

	client := action.NewPull()
	client.RepoURL = "" // Use default repo
	client.Version = version
	client.DestDir = os.TempDir()
	client.Untar = true

	chartPath, err := client.Run(chartName)
	if err != nil {
		return nil, fmt.Errorf("failed to pull chart: %w", err)
	}

	return loadLocalChartValues(chartPath)
}

// loadLocalChartValues loads values from a local chart directory or archive
func loadLocalChartValues(chartPath string) (map[string]interface{}, error) {
	log.Info().Msgf("Loading values from local chart: %s", chartPath)

	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	return chart.Values, nil
}

// updateOutputPaths updates all output paths to use the specified output directory
func updateOutputPaths() {
	outputFilePath = filepath.Join(outDir, "generated-values.yaml")
	optimizedValuesPath = filepath.Join(outDir, "optimized-values.yaml")
	unsupportedValuesPath = filepath.Join(outDir, "unsupported-values.yaml")
	redundantValuesPath = filepath.Join(outDir, "redundant-values.yaml")
}

// processValuesFiles analyzes upstream and downstream values files and generates reports
func processValuesFiles(upstreamPath, downstreamPath string) {
	log.Info().Msgf("Processing upstream values: %s", upstreamPath)
	log.Info().Msgf("Processing downstream values: %s", downstreamPath)

	upstreamContent, err := ioutil.ReadFile(upstreamPath)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to read upstream values file: %s", upstreamPath)
	}

	downstreamContent, err := ioutil.ReadFile(downstreamPath)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to read downstream values file: %s", downstreamPath)
	}

	var upstreamValues map[string]interface{}
	var downstreamValues map[string]interface{}

	err = yaml.Unmarshal(upstreamContent, &upstreamValues)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse upstream values YAML")
	}

	err = yaml.Unmarshal(downstreamContent, &downstreamValues)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse downstream values YAML")
	}

	// Analyze the values
	valueStatus := analyzeValues(upstreamValues, downstreamValues)

	// Always generate the generated-values.yaml file for compatibility
	generatedValues := make(map[string]interface{})
	for k, v := range downstreamValues {
		if _, exists := upstreamValues[k]; !exists {
			generatedValues[k] = v
		}
	}

	generatedContent, err := yaml.Marshal(generatedValues)
	if err != nil {
		log.Error().Err(err).Msg("error while marshaling generated values")
	} else {
		err = CreateOutputFile(generatedContent, outputFilePath)
		if err != nil {
			log.Error().Err(err).Msg("unable to write generated values file")
		} else {
			log.Info().Msgf("Generated values written to: %s", outputFilePath)
		}
	}

	// Generate optimized values (downstream without redundant values)
	if optimize {
		log.Info().Msg("Generating optimized values.yaml")
		optimizedContent, err := yaml.Marshal(valueStatus.Optimized)
		if err != nil {
			log.Error().Err(err).Msg("error while marshaling optimized values")
		} else {
			err = CreateOutputFile(optimizedContent, optimizedValuesPath)
			if err != nil {
				log.Error().Err(err).Msg("unable to write optimized values file")
			} else {
				log.Info().Msgf("Optimized values written to: %s", optimizedValuesPath)
			}
		}
	}

	// Output unsupported values
	if len(valueStatus.Unsupported) > 0 {
		count := countNestedKeys(valueStatus.Unsupported)
		log.Info().Msgf("Found %d unsupported values", count)
		unsupportedContent, err := yaml.Marshal(valueStatus.Unsupported)
		if err != nil {
			log.Error().Err(err).Msg("error while marshaling unsupported values")
		} else {
			err = CreateOutputFile(unsupportedContent, unsupportedValuesPath)
			if err != nil {
				log.Error().Err(err).Msg("unable to write unsupported values file")
			} else {
				log.Info().Msgf("Unsupported values written to: %s", unsupportedValuesPath)
			}
		}
	} else {
		log.Info().Msg("No unsupported values found")
	}

	// Output redundant values
	if len(valueStatus.Redundant) > 0 {
		count := countNestedKeys(valueStatus.Redundant)
		log.Info().Msgf("Found %d redundant values", count)
		redundantContent, err := yaml.Marshal(valueStatus.Redundant)
		if err != nil {
			log.Error().Err(err).Msg("error while marshaling redundant values")
		} else {
			err = CreateOutputFile(redundantContent, redundantValuesPath)
			if err != nil {
				log.Error().Err(err).Msg("unable to write redundant values file")
			} else {
				log.Info().Msgf("Redundant values written to: %s", redundantValuesPath)
				if output == "stdout" {
					fmt.Println("\nRedundant values:")
					fmt.Println(string(redundantContent))
				}
			}
		}
	} else {
		log.Info().Msg("No redundant values found")
	}

	log.Info().Msg("Values analysis completed successfully")
}

// countNestedKeys counts all nested keys in a map (including nested maps)
func countNestedKeys(m map[string]interface{}) int {
	count := 0
	for _, v := range m {
		count++
		if nestedMap, isMap := v.(map[string]interface{}); isMap {
			count += countNestedKeys(nestedMap)
		}
	}
	return count
}

// analyzeValues compares upstream and downstream values to detect unsupported, redundant, and optimized values
func analyzeValues(upstream, downstream map[string]interface{}) ValueStatus {
	valueStatus := ValueStatus{
		Redundant:   make(map[string]interface{}),
		Unsupported: make(map[string]interface{}),
		Optimized:   make(map[string]interface{}),
	}

	// First, create a deep copy of the downstream values for optimized output
	for k, v := range downstream {
		valueStatus.Optimized[k] = deepCopy(v)
	}

	// Process the values
	detectValuesStatus("", upstream, downstream, &valueStatus)

	// Special handling for service section which often has complex structures
	handleServiceValues(upstream, downstream, &valueStatus)

	return valueStatus
}

// handleServiceValues gives special attention to service configurations
func handleServiceValues(upstream, downstream map[string]interface{}, status *ValueStatus) {
	upService, hasUpService := upstream["service"].(map[string]interface{})
	downService, hasDownService := downstream["service"].(map[string]interface{})

	if !hasUpService || !hasDownService {
		return
	}

	// Extract service from optimized values for modification
	optimizedService, hasOptimizedService := status.Optimized["service"].(map[string]interface{})
	if !hasOptimizedService {
		return
	}

	for key, downVal := range downService {
		upVal, exists := upService[key]
		if !exists {
			// This key is unsupported
			if _, hasUnsupportedService := status.Unsupported["service"].(map[string]interface{}); !hasUnsupportedService {
				status.Unsupported["service"] = make(map[string]interface{})
			}
			status.Unsupported["service"].(map[string]interface{})[key] = downVal
		} else if equalValues(downVal, upVal) {
			// This key is redundant
			if _, hasRedundantService := status.Redundant["service"].(map[string]interface{}); !hasRedundantService {
				status.Redundant["service"] = make(map[string]interface{})
			}
			status.Redundant["service"].(map[string]interface{})[key] = downVal

			// Remove from optimized
			delete(optimizedService, key)
		}
	}

	// Update optimized service section
	if len(optimizedService) > 0 {
		status.Optimized["service"] = optimizedService
	} else {
		delete(status.Optimized, "service")
	}
}

// deepCopy creates a deep copy of an interface{}
func deepCopy(src interface{}) interface{} {
	if src == nil {
		return nil
	}

	// Handle maps (nested structures)
	if srcMap, isMap := src.(map[string]interface{}); isMap {
		dstMap := make(map[string]interface{})
		for k, v := range srcMap {
			dstMap[k] = deepCopy(v)
		}
		return dstMap
	}

	// Handle slices/arrays
	if srcSlice, isSlice := src.([]interface{}); isSlice {
		dstSlice := make([]interface{}, len(srcSlice))
		for i, v := range srcSlice {
			dstSlice[i] = deepCopy(v)
		}
		return dstSlice
	}

	// For basic types (ints, strings, bools, etc.), just return as is
	return src
}

// detectValuesStatus recursively compares upstream and downstream values
func detectValuesStatus(path string, upstream, downstream map[string]interface{}, status *ValueStatus) {
	// First pass: identify unsupported keys
	for key, downVal := range downstream {
		currentPath := joinPath(path, key)

		// Check if the key exists in upstream
		_, exists := upstream[key]
		if !exists {
			// Key in downstream doesn't exist in upstream, it's unsupported
			setNestedValue(status.Unsupported, currentPath, downVal)
		}
	}

	// Second pass: identify redundant keys and process nested maps
	for key, downVal := range downstream {
		currentPath := joinPath(path, key)

		// Skip if already identified as unsupported
		if isPathInMap(status.Unsupported, currentPath) {
			continue
		}

		// Check if the key exists in upstream
		upVal, exists := upstream[key]
		if !exists {
			continue // Already handled in first pass
		}

		// If both are maps, recurse
		downMap, downIsMap := downVal.(map[string]interface{})
		upMap, upIsMap := upVal.(map[string]interface{})

		if downIsMap && upIsMap {
			// Recursively process nested maps
			detectValuesStatus(currentPath, upMap, downMap, status)
		} else if equalValues(downVal, upVal) {
			// Values are the same, this is redundant
			setNestedValue(status.Redundant, currentPath, downVal)

			// Remove redundant value from optimized map
			removeNestedValue(status.Optimized, currentPath)
		}
	}
}

// isPathInMap checks if a given path exists in a nested map
func isPathInMap(m map[string]interface{}, path string) bool {
	parts := strings.Split(path, ".")
	current := m

	for i := 0; i < len(parts); i++ {
		if i == len(parts)-1 {
			_, exists := current[parts[i]]
			return exists
		}

		next, exists := current[parts[i]]
		if !exists {
			return false
		}

		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return false
		}

		current = nextMap
	}

	return false
}

// joinPath creates a dot-notation path
func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

// equalValues checks if two values are equal
func equalValues(a, b interface{}) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Type-specific comparisons
	switch aTyped := a.(type) {
	case map[string]interface{}:
		// Compare maps
		bMap, ok := b.(map[string]interface{})
		if !ok || len(aTyped) != len(bMap) {
			return false
		}

		for k, v := range aTyped {
			bVal, exists := bMap[k]
			if !exists || !equalValues(v, bVal) {
				return false
			}
		}
		return true

	case []interface{}:
		// Compare slices
		bSlice, ok := b.([]interface{})
		if !ok || len(aTyped) != len(bSlice) {
			return false
		}

		for i, v := range aTyped {
			if !equalValues(v, bSlice[i]) {
				return false
			}
		}
		return true

	default:
		// For basic types, use string representation comparison
		aStr := fmt.Sprintf("%v", a)
		bStr := fmt.Sprintf("%v", b)
		return aStr == bStr
	}
}

// setNestedValue sets a value in a nested map based on dot notation path
func setNestedValue(m map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	lastIndex := len(parts) - 1

	// Navigate to the correct nested level
	current := m
	for i := 0; i < lastIndex; i++ {
		part := parts[i]

		// Create nested map if it doesn't exist
		if _, exists := current[part]; !exists {
			current[part] = make(map[string]interface{})
		}

		// Cast to map for next iteration
		current = current[part].(map[string]interface{})
	}

	// Set the value at the final level
	current[parts[lastIndex]] = value
}

// removeNestedValue removes a value from a nested map based on dot notation path
func removeNestedValue(m map[string]interface{}, path string) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return
	}

	// If there's only one level, just delete it from the map
	if len(parts) == 1 {
		delete(m, parts[0])
		return
	}

	// For nested paths, we need to traverse the structure
	current := m
	var parent map[string]interface{}
	var lastKey string

	// Navigate through the nested levels until the second-to-last element
	for i := 0; i < len(parts)-1; i++ {
		key := parts[i]

		// If this level doesn't exist, nothing to remove
		nextLevel, exists := current[key]
		if !exists {
			return
		}

		// Cast to map to continue traversal
		var ok bool
		nextMap, ok := nextLevel.(map[string]interface{})
		if !ok {
			// If it's not a map, we can't go deeper
			return
		}

		// Save parent for later cleanup
		parent = current
		lastKey = key
		current = nextMap
	}

	// Delete the leaf key
	lastPart := parts[len(parts)-1]
	delete(current, lastPart)

	// Clean up empty parent maps
	if len(current) == 0 && parent != nil {
		delete(parent, lastKey)
	}
}

// DetectChangedValues extracts the changed values from a diff
func DetectChangedValues(diff dyff.Diff, changes objx.Map) objx.Map {
	var keyPath []string
	for _, e := range diff.Path.PathElements {
		keyPath = append(keyPath, e.Name)
	}
	keys := strings.Join(keyPath, ".")
	if diff.Details[0].From != nil {
		changes.Set(keys, diff.Details[0].From.Value)
	}
	return changes
}

// file loads an input file for analysis
func file(input string) ytbx.InputFile {
	inputfile, err := ytbx.LoadFile(input)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load input file")
	}
	return inputfile
}

// findKubeConfig finds the kubeconfig file path
func findKubeConfig() (string, error) {
	env := os.Getenv("KUBECONFIG")
	if env != "" {
		return env, nil
	}
	path, err := homedir.Expand("~/.kube/config")
	if err != nil {
		return "", err
	}
	return path, nil
}

// CreateOutputFile creates a YAML output file at the specified path
func CreateOutputFile(yamlOutput []byte, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Log a message that a file is being created at the specified path
	log.Info().Msgf("creating file: %s", path)

	// Write the YAML output to the specified path
	err := ioutil.WriteFile(path, yamlOutput, 0644)
	if err != nil {
		return err
	}

	return nil
}

// NewHelmClient creates a new Helm client with the specified configuration
func NewHelmClient() (*action.Configuration, error) {
	actionConfig := new(action.Configuration)

	// Configure the Helm client with the specified settings
	settings.KubeContext = context
	settings.KubeConfig = kubeConfigFile

	// If no namespace is specified, use the default namespace
	if namespace == "" {
		namespace = settings.Namespace()
	}

	// Initialize the Helm client with the specified configuration
	err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		log.Info().Msgf(format, v)
	})
	if err != nil {
		return nil, err
	}

	return actionConfig, nil
}

// HelmFetch fetches the release values for the specified repository and revision
func HelmFetch(h *action.Configuration) (map[string]interface{}, error) {
	// Create a new Helm Get action with the specified configuration
	c := action.NewGet(h)

	// Fetch the latest release from the specified repository
	rel, err := c.Run(repo)
	if err != nil {
		return nil, err
	}

	// Determine the release revision based on the specified revision
	var releaseRevision int
	if revision == 0 {
		// Use current version for upstream values
		releaseRevision = rel.Version
	} else {
		releaseRevision = revision
	}

	// Create a new Helm GetValues action with the specified configuration
	val := action.NewGetValues(h)
	val.Version = releaseRevision
	val.AllValues = true

	// Fetch the values for the selected release
	relVal, err := val.Run(rel.Name)
	if err != nil {
		return nil, err
	}

	return relVal, nil
}
