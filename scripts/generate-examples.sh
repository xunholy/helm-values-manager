#!/bin/bash

set -e

# Colors for better output
BLUE='\033[0;34m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Default values
CHART_NAME=${1:-"nginx"}
OUTPUT_DIR=${2:-"examples"}

echo -e "${BLUE}Generating example values for chart: ${CHART_NAME}${NC}"

# Create output directory if it doesn't exist
mkdir -p "${OUTPUT_DIR}"

# Check if Helm is installed
if ! command -v helm &> /dev/null; then
    echo -e "${RED}Error: Helm is not installed. Please install Helm first.${NC}"
    exit 1
fi

# Add common repositories if they don't exist
if ! helm repo list | grep -q "bitnami"; then
    echo -e "${BLUE}Adding Bitnami repository...${NC}"
    helm repo add bitnami https://charts.bitnami.com/bitnami
fi

# Update repositories
echo -e "${BLUE}Updating Helm repositories...${NC}"
helm repo update

# Determine full chart name
if [[ "$CHART_NAME" != */* ]]; then
    FULL_CHART_NAME="bitnami/$CHART_NAME"
else
    FULL_CHART_NAME="$CHART_NAME"
fi

# Generate upstream values file
UPSTREAM_FILE="${OUTPUT_DIR}/${CHART_NAME}-upstream-values.yaml"
echo -e "${BLUE}Generating upstream values from chart ${FULL_CHART_NAME}...${NC}"
if ! helm show values "$FULL_CHART_NAME" > "$UPSTREAM_FILE"; then
    echo -e "${RED}Error: Failed to fetch values from chart ${FULL_CHART_NAME}${NC}"
    echo -e "${YELLOW}Try specifying a different chart or adding the required repository.${NC}"
    exit 1
fi

echo -e "${GREEN}Successfully generated upstream values file: ${UPSTREAM_FILE}${NC}"

# Generate example custom values file with some common modifications
CUSTOM_FILE="${OUTPUT_DIR}/${CHART_NAME}-custom-values.yaml"
echo -e "${BLUE}Generating example custom values file...${NC}"

# Function to extract a section from the upstream values
extract_section() {
    local section=$1
    local file=$2
    grep -A50 "^$section:" "$file" | grep -B50 -m2 "^[a-z]" | head -n -1 || true
}

# Create custom values file with some common modifications and comments
cat > "$CUSTOM_FILE" << EOF
# Custom values for ${CHART_NAME}
# This is an example custom values file generated for testing with Helm Values Manager

# Example of values that match upstream defaults (redundant)
$(extract_section "image" "$UPSTREAM_FILE" || echo "image: {}")

# Example of modified values (these should be kept)
replicaCount: 3  # Modified from default

# Example of unsupported value (not in upstream chart)
unsupportedCustomSetting: "This doesn't exist in the upstream chart"

EOF

echo -e "${GREEN}Successfully generated custom values file: ${CUSTOM_FILE}${NC}"
echo -e "\n${BLUE}Usage:${NC}"
echo -e "  helm value-manager -upstream ${UPSTREAM_FILE} -downstream ${CUSTOM_FILE}"
echo -e "  helm value-manager -upstream ${UPSTREAM_FILE} -downstream ${CUSTOM_FILE} -optimize\n"
echo -e "${YELLOW}Note: Edit ${CUSTOM_FILE} to add your own customizations.${NC}"
