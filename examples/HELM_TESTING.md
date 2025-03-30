# Testing with Helm Charts

This guide explains how to test the Helm Values Manager with Helm charts, both locally and using repositories.

## Prerequisites

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

## Testing Options

### 1. Test with Built-in Example Files (Easiest)

The simplest approach is to use the provided example files:

```bash
# From the project root
./bin/helm-values-manager --upstream examples/test-upstream.yaml --downstream examples/test-downstream.yaml
```

This will analyze the differences between the example files and generate reports in the `values-analysis` directory.

### 2. Test with Downloaded Chart Values (Reliable)

Download a chart's default values and use them as the upstream source:

```bash
# Download chart's values.yaml
helm show values bitnami/nginx > nginx-values.yaml

# Create a custom values file with your changes
echo "replicaCount: 3" > my-values.yaml

# Run the analysis
./bin/helm-values-manager --upstream nginx-values.yaml --downstream my-values.yaml
```

### 3. Using the --chart Option (Note: May Require Additional Setup)

Note: The direct chart fetching feature may require additional setup due to how Helm loads plugins and repositories.

```bash
./bin/helm-values-manager --chart bitnami/nginx --downstream examples/test-downstream.yaml
```

**If you encounter errors with this method**, use the downloaded values approach (option 2) instead.

### 4. Test with an Installed Release (Requires Kubernetes)

If you have a Kubernetes cluster configured:

```bash
# Install a chart
helm install my-nginx bitnami/nginx

# Run analysis against this release
./bin/helm-values-manager --repo my-nginx --downstream examples/test-downstream.yaml

# Clean up
helm uninstall my-nginx
```

## Debugging Tips

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

## Example Workflow for Local Testing

```bash
# From project root:
# 1. Build the binary
task build

# 2. Download values from a chart
helm show values bitnami/nginx > examples/nginx-values.yaml

# 3. Create a custom values file
cat > examples/my-nginx-values.yaml << EOF
replicaCount: 3
service:
  type: NodePort
image:
  repository: nginx
  tag: 1.21.0
EOF

# 4. Run the analysis
./bin/helm-values-manager --upstream examples/nginx-values.yaml --downstream examples/my-nginx-values.yaml --optimize

# 5. Check results
cat values-analysis/optimized-values.yaml
cat values-analysis/redundant-values.yaml
cat values-analysis/unsupported-values.yaml
```

# Recommended Approach: Helper Script

For the most reliable testing experience, you can create a helper script to download chart values automatically:

```bash
#!/bin/bash
# save this as test-chart.sh and make it executable (chmod +x test-chart.sh)

# Check arguments
if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <chart> [chart-version] <custom-values-file>"
    echo "Example: $0 bitnami/nginx my-values.yaml"
    echo "Example with version: $0 bitnami/nginx 15.0.0 my-values.yaml"
    exit 1
fi

# Parse arguments
CHART="$1"
DOWNSTREAM=""
VERSION=""

if [ "$#" -eq 2 ]; then
    # No version specified
    DOWNSTREAM="$2"
else
    # Version specified
    VERSION="$2"
    DOWNSTREAM="$3"
fi

# Create temp directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Download chart values
if [ -z "$VERSION" ]; then
    echo "Downloading values from $CHART (latest version)..."
    helm show values "$CHART" > "$TEMP_DIR/upstream-values.yaml"
else
    echo "Downloading values from $CHART version $VERSION..."
    helm show values "$CHART" --version "$VERSION" > "$TEMP_DIR/upstream-values.yaml"
fi

# Check if download was successful
if [ ! -s "$TEMP_DIR/upstream-values.yaml" ]; then
    echo "Failed to download values from $CHART. Make sure the chart exists and the repo is added."
    exit 1
fi

echo "Running analysis..."
# Run the tool
./bin/helm-values-manager --upstream "$TEMP_DIR/upstream-values.yaml" --downstream "$DOWNSTREAM" --optimize

echo "Done! Results are in the values-analysis directory."
```

Usage:

```bash
# Make it executable
chmod +x test-chart.sh

# Run with latest version
./test-chart.sh bitnami/nginx examples/test-downstream.yaml

# Run with specific version
./test-chart.sh bitnami/nginx 15.0.0 examples/test-downstream.yaml
```

This script provides a convenient way to test the tool with any Helm chart without encountering the plugin loading issues.
