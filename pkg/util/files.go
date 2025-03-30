package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
)

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
	err := os.WriteFile(path, yamlOutput, 0644)
	if err != nil {
		return err
	}

	return nil
}

// FindKubeConfig finds the kubeconfig file path
func FindKubeConfig() (string, error) {
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

// EnsureDirectory creates a directory if it doesn't exist
func EnsureDirectory(dir string) error {
	return os.MkdirAll(dir, 0755)
}
