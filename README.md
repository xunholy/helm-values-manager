[![Licence](https://img.shields.io/badge/licence-Apache%202.0-green)]()

# Helm Values Manager

A Helm plugin that helps you manage your values.yaml files efficiently by analyzing, comparing, and optimizing them.

## Key Features

- **Value Comparison**: Compare downstream values to upstream chart defaults
- **Value Analysis**: Identify unsupported values in your custom values
- **Redundancy Detection**: Find and eliminate redundant values that match upstream defaults
- **Optimization**: Generate optimized values.yaml files without redundant entries
- **Multiple Input Methods**: Load upstream values from files, Helm charts, or releases

## Installation

### From Helm Plugin Repository

```bash
helm plugin install https://github.com/xunholy/helm-values-manager
```

### From Source

```bash
git clone https://github.com/xunholy/helm-values-manager.git
cd helm-values-manager
task build

# Manually install as a Helm plugin
mkdir -p $HELM_PLUGINS/helm-values-manager
cp -r bin plugin.yaml scripts $HELM_PLUGINS/helm-values-manager/
```

## Usage Examples

Here are the common usage patterns for Helm Values Manager:

### Compare with upstream values from a file

When you have a values.yaml file from a chart and your custom values:

```bash
helm values-manager --upstream chart-values.yaml --downstream my-values.yaml
```

### Load upstream values directly from a Helm chart

You can directly reference a chart to extract default values:

```bash
# Using latest version
helm values-manager --chart bitnami/nginx --downstream my-values.yaml

# Specifying a particular version
helm values-manager --chart bitnami/nginx --version 4.7.0 --downstream my-values.yaml
```

### Compare with a Helm release

If you have an existing release and want to compare with its chart defaults:

```bash
helm values-manager --repo my-release --downstream my-values.yaml
```

### Optimize your values.yaml

Remove redundant values that match the upstream defaults:

```bash
helm values-manager --upstream chart-values.yaml --downstream my-values.yaml --optimize
```

### Specify output directory

Output files to a custom directory:

```bash
helm values-manager --upstream chart-values.yaml --downstream my-values.yaml --outdir ./my-analysis
```

## Understanding Output

Helm Values Manager generates these output files in the target directory (default: `values-analysis/`):

- **optimized-values.yaml**: A cleaned version of your values file without redundant values (values that exactly match the upstream defaults)
- **unsupported-values.yaml**: Values in your file that don't have a corresponding key in the upstream chart
- **redundant-values.yaml**: Values in your file that match the upstream defaults (can be safely removed)
- **commented-values.yaml**: Values in your file that exist in the upstream chart but are commented out (only generated if such values are found)

These files help you understand how your custom values relate to the chart defaults and help you maintain cleaner configurations.

### Special Feature: Commented Values Detection

Many Helm charts (especially those with complex configurations like `cilium/cilium`) use commented-out fields to show available options. When you use these commented options in your values file, they might appear as "unsupported" in a regular analysis.

Helm Values Manager intelligently detects these commented fields and puts them in a separate `commented-values.yaml` file instead of marking them as unsupported. This helps you understand:

1. These values are actually supported by the chart
2. They are just commented out in the default values file
3. You can safely use them in your configuration

Example (using the cilium chart):
```bash
# Chart has commented options like:
# -- kubeProxyReplacement: false

# Your values has:
kubeProxyReplacement: true

# This will be classified as "commented" not "unsupported"
```

## Options

```
  -chart string
        name of the Helm chart to fetch upstream values from
  -chart-version string
        specific version of the Helm chart
  -downstream string
        path to the downstream values.yaml file (required)
  -kube-context string
        name of the kubeconfig context to use
  -kubeconfig string
        path to the kubeconfig file (default "~/.kube/config")
  -namespace string
        namespace scope for this request
  -optimize
        optimize values.yaml by removing redundant values
  -outdir string
        directory to store output files (default "values-analysis")
  -output string
        output format. One of: (yaml,stdout) (default "stdout")
  -repo string
        chart repository url where to locate the requested chart
  -revision int
        specify a revision constraint for the chart revision to use
  -upstream string
        path to the upstream values.yaml file
```

## Example Workflow

1. **Install a chart**
   ```bash
   helm install nginx-ingress nginx-stable/nginx-ingress
   ```

2. **Export your values**
   ```bash
   helm get values nginx-ingress > my-values.yaml
   ```

3. **Analyze for optimization**
   ```bash
   helm values-manager --chart nginx-stable/nginx-ingress --downstream my-values.yaml --optimize
   ```

4. **Use optimized values in your upgrade**
   ```bash
   helm upgrade nginx-ingress nginx-stable/nginx-ingress -f values-analysis/optimized-values.yaml
   ```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.

## Development

For detailed development instructions, see [DEVELOPMENT.md](DEVELOPMENT.md).

### Prerequisites

- Go 1.21 or higher
- Task (https://taskfile.dev/)
- Helm (for testing with Helm releases)

### Building from source

```bash
# Clone the repository
git clone https://github.com/xunholy/helm-values-manager.git
cd helm-values-manager

# Build the binary
task build

# Run unit tests
task test
```

### Running Tests

#### Unit Tests

Run the Go unit tests:

```bash
task test
```

#### Integration Tests

The integration tests verify that the application correctly processes YAML files and generates the expected outputs:

```bash
task integration-test
```

This will:
1. Build the application if needed
2. Set up test YAML files
3. Process the test files with default and custom output directories
4. Verify that all expected output files are created
5. Check that values are correctly identified as redundant or unsupported

#### Manual Testing with Helm Charts

For testing with Helm charts and repositories, see our [Examples and Testing Guide](examples/README.md) which includes:
- Working with chart repositories
- Using local and remote charts
- Testing with different chart versions
- Helper scripts for easy testing

#### Manual Testing with Example Files

You can test manually using the provided example files:

```bash
# From the project root directory:

# Generate example files (if they don't exist already)
./scripts/generate-examples.sh

# Run the tool on the example files
./bin/helm-values-manager --upstream examples/test-upstream.yaml --downstream examples/test-downstream.yaml

# Test with optimization
./bin/helm-values-manager --upstream examples/test-upstream.yaml --downstream examples/test-downstream.yaml --optimize

# Test with custom output directory
./bin/helm-values-manager --upstream examples/test-upstream.yaml --downstream examples/test-downstream.yaml --outdir ./custom-output

# Examine the output
ls -la values-analysis/
```

### Available Tasks

```bash
# List all available tasks
task --list

# Build the application
task build

# Run unit tests
task test

# Run integration tests
task integration-test

# Format code
task fmt

# Lint and vet code
task lint
task vet

# Build a Docker image
task docker

# Push Docker image to registry
task docker-push

# Release a new version (requires goreleaser)
task release
```
