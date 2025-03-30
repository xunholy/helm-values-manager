package helm

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
)

// Client represents a Helm client with configuration
type Client struct {
	Settings   *cli.EnvSettings
	Config     *action.Configuration
	Context    string
	Namespace  string
	KubeConfig string
}

// NewClient creates a new Helm client with the specified configuration
func NewClient(context, namespace, kubeConfig string) (*Client, error) {
	client := &Client{
		Settings:   cli.New(),
		Context:    context,
		Namespace:  namespace,
		KubeConfig: kubeConfig,
	}

	// Configure the Helm client with the specified settings
	client.Settings.KubeContext = context
	client.Settings.KubeConfig = kubeConfig

	// Initialize action configuration
	actionConfig := new(action.Configuration)

	// If no namespace is specified, use the default namespace
	if namespace == "" {
		namespace = client.Settings.Namespace()
	}

	// Initialize the Helm client with the specified configuration
	err := actionConfig.Init(client.Settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		log.Info().Msgf(format, v)
	})
	if err != nil {
		return nil, err
	}

	client.Config = actionConfig
	return client, nil
}

// FetchReleaseValues fetches the values from a Helm release
func (c *Client) FetchReleaseValues(releaseName string, revision int) (map[string]interface{}, error) {
	// Create a new Helm Get action with the specified configuration
	get := action.NewGet(c.Config)

	// Fetch the latest release from the specified repository
	rel, err := get.Run(releaseName)
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
	val := action.NewGetValues(c.Config)
	val.Version = releaseRevision
	val.AllValues = true

	// Fetch the values for the selected release
	relVal, err := val.Run(rel.Name)
	if err != nil {
		return nil, err
	}

	return relVal, nil
}

// FetchChartValues gets values from a Helm chart repository or local file
func FetchChartValues(chartName, version string) (map[string]interface{}, error) {
	log.Info().Msgf("Fetching values from Helm chart: %s (version: %s)", chartName, version)

	// First check if it's a local file and try to load it as a values file
	if _, err := os.Stat(chartName); err == nil {
		// If it's a YAML file, just load it as values
		if strings.HasSuffix(chartName, ".yaml") || strings.HasSuffix(chartName, ".yml") {
			log.Info().Msgf("Loading values from YAML file: %s", chartName)
			yamlContent, err := os.ReadFile(chartName)
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

	// We'll use the 'helm show values' command as a reliable way to get chart values
	log.Info().Msgf("Using helm command to download chart values")

	// Create a temporary file to store the values
	tempDir, err := os.MkdirTemp("", "helm-values-manager-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	tempFile := fmt.Sprintf("%s/chart-values.yaml", tempDir)

	// Construct the helm command
	var cmd string
	if version == "" {
		cmd = fmt.Sprintf("helm show values %s > %s", chartName, tempFile)
		log.Info().Msgf("Using latest chart version")
	} else {
		cmd = fmt.Sprintf("helm show values %s --version %s > %s", chartName, version, tempFile)
		log.Info().Msgf("Using chart version: %s", version)
	}

	// Execute the command
	cmdExec := exec.Command("bash", "-c", cmd)
	cmdOutput, err := cmdExec.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Str("output", string(cmdOutput)).Msg("Failed to run helm show values command")
		return nil, fmt.Errorf("failed to fetch chart values: %s (output: %s)", err, string(cmdOutput))
	}

	// Check if the file exists and has content
	if _, err := os.Stat(tempFile); err != nil || isFileEmpty(tempFile) {
		log.Error().Msg("No values retrieved from helm command or chart not found")
		return nil, fmt.Errorf("chart values not found or empty for: %s version: %s", chartName, version)
	}

	// Load the values from the temporary file
	yamlContent, err := os.ReadFile(tempFile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read values from temporary file")
		return nil, fmt.Errorf("failed to read chart values: %w", err)
	}

	var values map[string]interface{}
	if err := yaml.Unmarshal(yamlContent, &values); err != nil {
		log.Error().Err(err).Msg("Failed to parse values from temporary file")
		return nil, fmt.Errorf("failed to parse chart values: %w", err)
	}

	if len(values) == 0 {
		log.Error().Msg("Empty values map retrieved from chart")
		return nil, fmt.Errorf("chart %s (version: %s) has empty values", chartName, version)
	}

	log.Info().Msg("Successfully retrieved chart values")
	return values, nil
}

// Helper function to check if a file is empty
func isFileEmpty(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return true
	}
	return info.Size() == 0
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

// FetchChartValuesRaw gets the raw YAML values from a Helm chart
func FetchChartValuesRaw(chartName, version string) ([]byte, error) {
	log.Info().Msgf("Fetching values from Helm chart: %s (version: %s)", chartName, version)
	log.Info().Msg("Using helm command to download chart values with comments preserved")

	// Construct the command to fetch raw values
	args := []string{"show", "values"}

	// Add version if specified
	if version != "" {
		args = append(args, "--version", version)
		log.Info().Msgf("Using chart version: %s", version)
	} else {
		log.Info().Msg("Using latest chart version")
	}

	// Add chart name
	args = append(args, chartName)

	// Execute the command
	cmd := exec.Command("helm", args...)

	// Capture the output
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("helm command failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute helm command: %w", err)
	}

	// Explicitly check if we have any commented fields by searching for common patterns
	if len(output) > 0 {
		yamlStr := string(output)
		if strings.Contains(yamlStr, "#") {
			log.Info().Msg("Successfully retrieved raw chart values with comments preserved")
		} else {
			log.Warn().Msg("Raw chart values fetched, but no comments detected. Some commented values may not be detected correctly.")
		}
	}

	return output, nil
}
