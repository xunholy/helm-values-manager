package helm

import (
	"fmt"
	"os"
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
