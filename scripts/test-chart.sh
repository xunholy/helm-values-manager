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

# Clean up any previous results
rm -rf values-analysis

# Create temp directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Download chart values
echo "Downloading original chart values for better analysis..."
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

# Copy to a permanent location for reference
mkdir -p chart-values
CHART_NAME=$(echo "$CHART" | sed 's/\//\-/g')
if [ -z "$VERSION" ]; then
    ORIGINAL_VALUES="chart-values/${CHART_NAME}-values.yaml"
else
    ORIGINAL_VALUES="chart-values/${CHART_NAME}-${VERSION}-values.yaml"
fi
cp "$TEMP_DIR/upstream-values.yaml" "$ORIGINAL_VALUES"
echo "Original chart values saved to $ORIGINAL_VALUES for reference"

# Function to check for commented values
check_commented_values() {
    if [ -f "values-analysis/commented-values.yaml" ]; then
        echo -e "\nDetected commented values: (these exist in upstream but are commented out)"
        cat values-analysis/commented-values.yaml
    fi
}

echo "Running analysis..."
# Run the tool with upstream values
echo -e "\n=== Method 1: Using downloaded values file (RECOMMENDED for charts with lots of comments) ==="
echo "This method preserves all comments in the chart and gives best results for charts like cilium/cilium"
./bin/helm-values-manager --upstream "$ORIGINAL_VALUES" --downstream "$DOWNSTREAM" --optimize
check_commented_values

# Clean up before next run
rm -rf values-analysis

# Test with the --chart option directly
echo -e "\n=== Method 2: Using --chart option directly (easier but may miss some comments) ==="
if [ -z "$VERSION" ]; then
    echo "Running helm-values-manager with chart (latest version)..."
    ./bin/helm-values-manager --chart "$CHART" --downstream "$DOWNSTREAM" --optimize
else
    echo "Running helm-values-manager with chart version $VERSION..."
    ./bin/helm-values-manager --chart "$CHART" --version "$VERSION" --downstream "$DOWNSTREAM" --optimize
fi
check_commented_values

echo -e "\nDone! Results are in the values-analysis directory."
echo "Original chart values are preserved in $ORIGINAL_VALUES for reference."
echo -e "\nTIP: For charts with lots of commented options (like cilium/cilium), Method 1 is RECOMMENDED"
echo "     as it better preserves and detects all commented fields in the chart."
