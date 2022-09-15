package main

import (
	"flag"
	"os"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var (
	repo           string
	kubeConfigFile string
	context        string
	namespace      string
	version        int
	settings       = cli.New()
)

type KubeConfigSetup struct {
	Context        string
	KubeConfigFile string
	Namespace      string
}

func (e *KubeConfigSetup) init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	// TODO: Derive namespace and context from kubeconfig
	defaultKubeConfigPath, err := findKubeConfig()
	if err != nil {
		log.Warn().AnErr("kubeConfigPath", err).Msg("Unable to determine default kubeconfig path")
	}

	flag.StringVar(&repo, "repo", "", "chart repository url where to locate the requested chart")
	flag.IntVar(&version, "version", 0, "specify a version constraint for the chart version to use. This constraint can be a specific tag (e.g. 1.1.1) or it may reference a valid range (e.g. ^2.0.0). If this is not specified, the latest version is used")
	flag.StringVar(&kubeConfigFile, "kubeconfig", defaultKubeConfigPath, "path to the kubeconfig file")
	flag.StringVar(&context, "kube-context", "", "name of the kubeconfig context to use")
	flag.StringVar(&namespace, "namespace", "", "namespace scope for this request")
}

func main() {
	flag.Parse()
	if repo == "" {
		log.Error().Msg("missing -repo flag")
		flag.Usage()
		os.Exit(2)
	}

	if version == 0 {
		log.Info().Msg("version not specified. default: 0")
	}
}

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

// TODO: Refactor - Currently pulled from OSS
// NewClient return a new helm client with provided config
func NewClient(namespace string, kfcg KubeConfigSetup) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)

	return func(ac *action.Configuration) (*action.Configuration, error) {
		settings.KubeContext = kfcg.Context
		settings.KubeConfig = kfcg.KubeConfigFile
		if namespace == "" {
			namespace = settings.Namespace()
		} else {
			kfcg.Namespace = settings.Namespace()
		}
		err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
			// log.Debug(fmt.Sprintf(format, v))
		})
		if err != nil {
			return nil, err
		}
		return ac, nil

	}(actionConfig)
}

// getLastRelease fetch the latest release
func getLastRelease(release string, ac *action.Get) (*release.Release, error) {
	rel, err := ac.Run(release)
	if err != nil {
		return nil, err
	}
	return rel, nil
}

// getHelmClient returns action configuration based on Helm env
func getHelmClient() (*action.Configuration, error) {
	kubeConfig := KubeConfigSetup{
		Context:        context,
		KubeConfigFile: kubeConfigFile,
		Namespace:      namespace,
	}
	ac, err := NewClient(kubeConfig.Namespace, kubeConfig)
	if err != nil {
		return nil, err
	}
	return ac, nil
}

// runFetch retrieve values for a given old release
func runFetch(args []string) (map[string]interface{}, error) {
	repo := args[0]

	ac, err := getHelmClient()
	if err != nil {
		return nil, err
	}
	p := action.NewGet(ac)
	rel, err := getLastRelease(repo, p)
	if err != nil {
		return nil, err
	}

	var previousRelease int
	if version == 0 {
		previousRelease = rel.Version - 1
	} else {
		previousRelease = version
	}
	gVal := action.NewGetValues(ac)
	gVal.Version = previousRelease
	gVal.AllValues = true

	relVal, err := gVal.Run(rel.Name)
	if err != nil {
		return nil, err
	}

	return relVal, nil
}
