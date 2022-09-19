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
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/gonvenience/ytbx"
	"github.com/homeport/dyff/pkg/dyff"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
	"github.com/stretchr/objx"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var (
	repo           string
	kubeConfigFile string
	context        string
	namespace      string
	version        int
	output         string
)

type KubeConfigSetup struct {
	Context        string
	KubeConfigFile string
	Namespace      string
}

type ChangeRequest struct {
	Path    string
	Content reflect.Value
}

type Changes struct {
	Items []*ChangeRequest
}

func init() {
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
	flag.StringVar(&output, "output", "stdout", "output format. One of: (yaml,stdout)")
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
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	downstreamFile := file(filepath.Join(cwd, "examples", "upstream-values.yaml"))
	upstreamFile := file(filepath.Join(cwd, "examples", "downstream-values.yaml"))
	diff, err := dyff.CompareInputFiles(upstreamFile, downstreamFile)
	if err != nil {
		panic(err)
	}
	changes := objx.Map{}
	for _, v := range diff.Diffs {
		changes = DetectChangedValues(v, changes)
	}
	yamlOutput, err := yaml.Marshal(&changes)
	if err != nil {
		fmt.Printf("Error while Marshaling. %v", err)
	}
	switch output {
	case "yaml":
		CreateOutputFile(yamlOutput)
	default:
		fmt.Println(string(yamlOutput))
	}
}

func DetectChangedValues(diff dyff.Diff, changes objx.Map) objx.Map {
	var keyPath []string
	for _, e := range diff.Path.PathElements {
		keyPath = append(keyPath, e.Name)
	}
	keys := strings.Join(keyPath, ".")
	changes.Set(keys, diff.Details[0].From.Value)
	return changes
}

func file(input string) ytbx.InputFile {
	inputfile, err := ytbx.LoadFile(input)
	if err != nil {
		fmt.Sprintf("Failed to load input file from %s: %s", input, err.Error())
	}

	return inputfile
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

func CreateOutputFile(yamlOutput []byte) {
	fileName := "examples/generated-values.yaml"
	err := ioutil.WriteFile(fileName, yamlOutput, 0644)
	if err != nil {
		panic("Unable to write data into the file")
	}
}
