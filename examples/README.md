# Helm Values Manager Examples and Testing Guide

This directory contains example values files and usage guides for the Helm Values Manager tool.

## Example Files

- `test-upstream.yaml` - Example chart defaults based on Bitnami Nginx chart
- `test-downstream.yaml` - Example user configuration with redundant, modified, and unsupported values

## Testing with Helm Charts

This section explains how to test the Helm Values Manager with Helm charts, both locally and using repositories.

### Prerequisites

1. **Install Helm**:
   ```bash
   curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
   ```

2. **Set up Helm repositories**:
   ```bash
   # Add Bitnami repository (widely used)
   helm repo add bitnami https://charts.bitnami.com/bitnami

   # Update repositories
   helm repo update
   ```

### Testing Options

#### 1. Test with Built-in Example Files (Easiest)

The simplest approach is to use the provided example files:

```bash
# From the project root
./bin/helm-values-manager --upstream examples/test-upstream.yaml --downstream examples/test-downstream.yaml
```

This will analyze the differences between the example files and generate reports in the `values-analysis` directory.

#### 2. Test with Downloaded Chart Values (Reliable)

Download a chart's default values and use them as the upstream source:

```bash
# Download chart's values.yaml
helm show values bitnami/nginx > nginx-values.yaml

# Create a custom values file with your changes
echo "replicaCount: 3" > my-values.yaml

# Run the analysis
./bin/helm-values-manager --upstream nginx-values.yaml --downstream my-values.yaml
```

#### 3. Using the --chart Option with Version Support

You can directly reference a chart to extract default values, with optional version specification:

```bash
# Latest version
./bin/helm-values-manager --chart bitnami/nginx --downstream examples/test-downstream.yaml

# Specific version
./bin/helm-values-manager --chart bitnami/nginx --version 15.0.0 --downstream examples/test-downstream.yaml
```

#### 4. Test with an Installed Release (Requires Kubernetes)

If you have a Kubernetes cluster configured:

```bash
# Install a chart
helm install my-nginx bitnami/nginx

# Run analysis against this release
./bin/helm-values-manager --repo my-nginx --downstream examples/test-downstream.yaml

# Clean up
helm uninstall my-nginx
```

### Debugging Tips

If you encounter issues with Helm repositories:

1. **Check your Helm repositories**:
   ```bash
   helm repo list
   ```

2. **Verify repository connection**:
   ```bash
   helm search repo bitnami/nginx
   ```

3. **Manually download values** if automatic fetching fails:
   ```bash
   helm show values bitnami/nginx > nginx-values.yaml
   ```

4. **Set environment variables** if needed:
   ```bash
   export HELM_REPOSITORY_CONFIG="$HOME/.config/helm/repositories.yaml"
   export HELM_REPOSITORY_CACHE="$HOME/.cache/helm/repository"
   ```

### Helper Scripts

The project includes two helper scripts for testing:

1. **test.sh** - Tests basic functionality using the example files:
   ```bash
   ./scripts/test.sh
   ```

2. **test-chart.sh** - Tests with any Helm chart:
   ```bash
   # Make it executable
   chmod +x scripts/test-chart.sh

   # Run with latest version
   ./scripts/test-chart.sh bitnami/nginx examples/test-downstream.yaml

   # Run with specific version
   ./scripts/test-chart.sh bitnami/nginx 15.0.0 examples/test-downstream.yaml
   ```

## Understanding Output Files

Helm Values Manager generates three output files in the target directory (default: `values-analysis/`):

- **optimized-values.yaml**: A cleaned version of your values file without redundant values (values that exactly match the upstream defaults)
- **unsupported-values.yaml**: Values in your file that don't have a corresponding key in the upstream chart
- **redundant-values.yaml**: Values in your file that match the upstream defaults (can be safely removed)

These files help you understand how your custom values relate to the chart defaults and help you maintain cleaner configurations.

## Tutorial: Optimizing Nginx Values

This tutorial demonstrates how to use Helm Values Manager to optimize your values for Nginx.

### 1. Get the default values from the chart

```bash
# Add the Bitnami repository if you haven't already
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Download the default values to a file
helm show values bitnami/nginx > nginx-upstream-values.yaml
```

### 2. Create your custom values file

Create a file called `my-nginx-values.yaml` with your customizations:

```yaml
replicaCount: 3
image:
  tag: 1.25.0
service:
  type: NodePort
  nodePorts:
    http: 30080
resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 50m
    memory: 64Mi
metrics:
  enabled: true

# The following value doesn't exist in the upstream chart
customSetting: "this will be detected as unsupported"
```

### 3. Analyze with Helm Values Manager

```bash
helm values-manager --upstream nginx-upstream-values.yaml --downstream my-nginx-values.yaml
```

### 4. Generate optimized values

```bash
helm values-manager --upstream nginx-upstream-values.yaml --downstream my-nginx-values.yaml --optimize
```

### 5. Reviewing the output

```bash
# Check unsupported values
cat values-analysis/unsupported-values.yaml

# Check redundant values (if any exist in your custom file)
cat values-analysis/redundant-values.yaml

# Review optimized values
cat values-analysis/optimized-values.yaml
```

## Using in CI/CD

Here's an example of how you might use Helm Values Manager in a CI/CD pipeline:

```bash
#!/bin/bash
# Example CI script

# Run the values manager
helm values-manager --upstream nginx-upstream-values.yaml --downstream my-nginx-values.yaml --outdir ci-results

# Check if unsupported values exist
if [ -s "ci-results/unsupported-values.yaml" ]; then
  echo "Error: Unsupported values detected in configuration!"
  cat ci-results/unsupported-values.yaml
  exit 1
fi

# Deploy using optimized values if required
if [ "$OPTIMIZE" == "true" ]; then
  helm values-manager --upstream nginx-upstream-values.yaml --downstream my-nginx-values.yaml --outdir ci-results --optimize
  helm upgrade --install nginx bitnami/nginx -f ci-results/optimized-values.yaml
else
  helm upgrade --install nginx bitnami/nginx -f my-nginx-values.yaml
fi
```
