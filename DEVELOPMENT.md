# Development Guide

This document provides detailed instructions for developers who want to contribute to the Helm Values Manager project.

## Prerequisites

- Go 1.21 or higher
- Task (https://taskfile.dev/) - Build tool
- Helm (for testing with Helm releases)
- Docker (optional, for building container images)

## Setting Up Your Development Environment

1. Clone the repository:
   ```bash
   git clone https://github.com/xunholy/helm-values-manager.git
   cd helm-values-manager
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Install Task (if not already installed):
   ```bash
   # macOS
   brew install go-task/tap/go-task

   # Linux
   sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin

   # Windows with scoop
   scoop install task
   ```

## Project Structure

```
.
├── cmd/                 # Command-line applications
│   └── helm-values-manager/ # Main application entry point
├── pkg/                 # Reusable packages
│   ├── analyzer/        # Values analysis functionality
│   ├── helm/            # Helm client integration
│   ├── output/          # Output formatting and file handling
│   └── util/            # Utility functions
├── scripts/             # Helper scripts
│   ├── build.sh         # Build script
│   ├── test.sh          # Test script
│   └── generate-examples.sh # Creates example files
├── bin/                 # Compiled binaries (generated)
├── examples/            # Example YAML files
└── values-analysis/     # Default output directory (generated)
```

## Building the Project

Build the binary:
```bash
task build
```

The binary will be created at `bin/helm-values-manager`.

## Testing

### Running Unit Tests

```bash
task test
```

### Running Integration Tests

```bash
task integration-test
```

The integration tests run the application against example YAML files and verify that the expected output files are generated.

### Running Both Test Types

To run both unit and integration tests sequentially:

```bash
# Run all tests
task test && task integration-test

# Or use the CI task which includes tests
task ci
```

### Manual Testing

1. Generate example files:
   ```bash
   ./scripts/generate-examples.sh
   ```

2. Process the example files:
   ```bash
   ./bin/helm-values-manager --upstream examples/test-upstream.yaml --downstream examples/test-downstream.yaml
   ```

3. Examine the output:
   ```bash
   ls -la values-analysis/
   ```

4. Test with optimization:
   ```bash
   ./bin/helm-values-manager --upstream examples/test-upstream.yaml --downstream examples/test-downstream.yaml --optimize
   ```

5. Test with custom output directory:
   ```bash
   ./bin/helm-values-manager --upstream examples/test-upstream.yaml --downstream examples/test-downstream.yaml --outdir ./custom-output
   ```

## Code Quality

Format your code:
```bash
task fmt
```

Run linting and static analysis:
```bash
task lint
task vet
```

## Docker

Build a Docker image:
```bash
task docker
```

Push the Docker image to a registry:
```bash
task docker-push
```

## Release Process

1. Tag your version:
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```

2. Use GoReleaser to create a release:
   ```bash
   task release
   ```

## Plugin Installation Testing

To test the plugin installation process:

1. Build the binary:
   ```bash
   task build
   ```

2. Create a temporary Helm plugins directory:
   ```bash
   mkdir -p /tmp/helm-plugins/helm-values-manager
   ```

3. Copy the necessary files:
   ```bash
   cp -r bin plugin.yaml scripts /tmp/helm-plugins/helm-values-manager/
   ```

4. Set the HELM_PLUGINS environment variable:
   ```bash
   export HELM_PLUGIN_DIR=/tmp/helm-plugins/helm-values-manager
   ```

5. Run the plugin:
   ```bash
   cd /tmp/helm-plugins
   helm values-manager -h
   ```

## Debugging Tips

1. To increase verbosity, add logging statements using the zerolog package:
   ```go
   import "github.com/rs/zerolog/log"

   // Log an info message
   log.Info().Msg("Processing completed")

   // Log with fields
   log.Info().Str("file", filename).Int("count", count).Msg("File processed")

   // Log errors
   log.Error().Err(err).Msg("Failed to process file")
   ```

2. Examine YAML files directly:
   ```bash
   cat values-analysis/unsupported-values.yaml
   ```

3. Run with a simple test case:
   ```bash
   echo "foo: bar" > simple-upstream.yaml
   echo "foo: bar\nbaz: qux" > simple-downstream.yaml
   ./bin/helm-values-manager --upstream simple-upstream.yaml --downstream simple-downstream.yaml
   ```

## Getting Help

If you encounter any issues during development, feel free to:

- Open an issue on GitHub
- Ask for help in pull requests
- Check the existing documentation and code comments
