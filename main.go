package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"time"

	"gopkg.in/yaml.v2"
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
	upstream, err := ReadYAML(filepath.Join(cwd, "examples", "upstream-values.yaml"))
	if err != nil {
		log.Error().Err(err).Msg("failed to read yaml file")
	}
	downstream, err := ReadYAML(filepath.Join(cwd, "examples", "downstream-values.yaml"))
	if err != nil {
		log.Error().Err(err).Msg("failed to read yaml file")
	}
	DetectChangedValues(upstream, downstream)
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

func ReadYAML(filePath string) (map[string]interface{}, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	result := make(map[string]interface{})
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func DetectChangedValues(a map[string]interface{}, b map[string]interface{}) ([]byte, error) {
	// step through each map to determine key : value equal otherwise

	if reflect.DeepEqual(a, b) {
		return nil, nil
	}

	valueA := reflect.ValueOf(a)
	valueB := reflect.ValueOf(b)

	changes := &Changes{}

	isTheSame := GetChanges(valueA, valueB, "root", changes)

	fmt.Println(">>> isTheSame", isTheSame)

	return nil, nil
}

func GetChanges(a reflect.Value, b reflect.Value, path string, changes *Changes) bool {
	kindA := a.Kind()
	kindB := b.Kind()

	fmt.Println("[ENTER] >>> path", path, ">>> kind", kindA, kindB, reflect.ValueOf(kindA).Kind())

	if kindA != kindB {
		return false
	}

	// a and be are the same type from here on....
	if kindA == kindB {
		switch kindA {
		case reflect.Map:
			var keysA, keysB []string
			for _, key := range a.MapKeys() {
				if key.Kind() != reflect.String {
					panic("Expect map keys to be string")
				}
				keysA = append(keysA, key.String())
			}
			for _, key := range b.MapKeys() {
				if key.Kind() != reflect.String {
					panic("Expect map keys to be string")
				}
				keysB = append(keysB, key.String())
			}

			sort.Strings(keysA)
			sort.Strings(keysB)

			if reflect.DeepEqual(keysA, keysB) {
				// doesn't matter if we use keysA or B as they are the same
				for _, key := range a.MapKeys() {
					itemA := a.MapIndex(key)
					itemB := b.MapIndex(key)

					isSame := GetChanges(itemA.Elem(), itemB.Elem(), path+"."+key.String(), changes)
					if !isSame {
						return false
					}
				}
			} else {
				fmt.Println(">>> keys are NOT same")
				fmt.Println(a.MapKeys())
				fmt.Println(b.MapKeys())
				return false
			}
		case reflect.Slice:
			return reflect.DeepEqual(a, b)

		case reflect.Interface:
			fmt.Println(">>> debug", a, b, kindA, kindB)

		default:
			fmt.Println(a, b, kindA, kindB)
			panic("Couldn't determine stuff")

		}
	}

	return false
}
