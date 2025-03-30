#!/bin/bash
# Helper script to test Helm Values Manager with any Helm chart

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
    echo "Run: helm repo add bitnami https://charts.bitnami.com/bitnami && helm repo update"
    exit 1
fi

echo "Running analysis..."
# Run the tool with upstream values
echo "Method 1: Using downloaded values file:"
./bin/helm-values-manager --upstream "$TEMP_DIR/upstream-values.yaml" --downstream "$DOWNSTREAM" --optimize

# Test with the --chart option directly
echo -e "\nMethod 2: Using --chart option directly:"
if [ -z "$VERSION" ]; then
    echo "Running helm-values-manager with chart (latest version)..."
    ./bin/helm-values-manager --chart "$CHART" --downstream "$DOWNSTREAM" --optimize
else
    echo "Running helm-values-manager with chart version $VERSION..."
    ./bin/helm-values-manager --chart "$CHART" --version "$VERSION" --downstream "$DOWNSTREAM" --optimize
fi

echo -e "\nDone! Results are in the values-analysis directory."
