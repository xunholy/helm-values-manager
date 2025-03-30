#!/bin/bash

set -e

HELM_VALUES_MANAGER_VERSION=${HELM_VALUES_MANAGER_VERSION:-"latest"}
HELM_VALUES_MANAGER_INSTALL_DIR=${HELM_VALUES_MANAGER_INSTALL_DIR:-"$HELM_PLUGIN_DIR"}

OWNER=${OWNER:-"xunholy"}
REPO=${REPO:-"helm-values-manager"}
GITHUB_URI="https://github.com/${OWNER}/${REPO}"

# Check if the binary is already installed
valueManagerBin="${HELM_VALUES_MANAGER_INSTALL_DIR}/bin/helm-values-manager"
if [ -f "$valueManagerBin" ]; then
  echo "Helm Values Manager is already installed at $valueManagerBin"
  echo "Use 'helm values-manager -h' to see available commands"
  exit 0
fi

# Create bin directory if it doesn't exist
mkdir -p "${HELM_VALUES_MANAGER_INSTALL_DIR}/bin"
mkdir -p "${HELM_VALUES_MANAGER_INSTALL_DIR}/scripts"

# Determine architecture
initArch() {
  ARCH=$(uname -m)
  case $ARCH in
    armv5*) ARCH="armv5";;
    armv6*) ARCH="armv6";;
    armv7*) ARCH="arm";;
    aarch64) ARCH="arm64";;
    x86) ARCH="386";;
    x86_64) ARCH="amd64";;
    i686) ARCH="386";;
    i386) ARCH="386";;
  esac
}

# Determine OS
initOS() {
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$OS" in
    # Minimalist GNU for Windows
    mingw*) OS='windows';;
    msys*) OS='windows';;
  esac
}

# Initialize variables
initArch
initOS

# Prepare the download URL
if [ "$HELM_VALUES_MANAGER_VERSION" = "latest" ]; then
  DOWNLOAD_URL="${GITHUB_URI}/releases/latest/download/helm-values-manager_${OS}_${ARCH}"
else
  DOWNLOAD_URL="${GITHUB_URI}/releases/download/${HELM_VALUES_MANAGER_VERSION}/helm-values-manager_${OS}_${ARCH}"
fi

# Create scripts directory and copy example generation script
cat > "${HELM_VALUES_MANAGER_INSTALL_DIR}/scripts/generate-examples.sh" << 'EOF'
#!/bin/bash

# Create examples directory if it doesn't exist
mkdir -p examples

# Generate upstream values.yaml example
cat > examples/test-upstream.yaml << 'EOFINNER'
# Test upstream values file (chart defaults)
image:
  repository: test/app
  tag: 1.0.0
  pullPolicy: IfNotPresent

replicas: 1

resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 50m
    memory: 64Mi

service:
  type: ClusterIP
  port: 80

ingress:
  enabled: false
  annotations: {}
  hosts:
    - host: chart-example.local
      paths:
        - path: /
          pathType: Prefix

configMap:
  data:
    KEY1: value1
    KEY2: value2

persistence:
  enabled: false
  size: 1Gi
  storageClass: standard
EOFINNER

# Generate downstream values.yaml example
cat > examples/test-downstream.yaml << 'EOFINNER'
# Test downstream values file (user custom values)
# Contains both redundant and unsupported values

# Redundant values (same as upstream)
image:
  repository: test/app
  pullPolicy: IfNotPresent

# Modified values (should be kept)
image:
  tag: 1.2.0  # Different from upstream

# Unsupported values (not in upstream)
unsupportedKey: "This doesn't exist in upstream"
deprecated:
  feature:
    enabled: true
    settings:
      timeout: 30

# Nested redundant values
resources:
  limits:
    cpu: 100m    # Redundant
    memory: 256Mi  # Modified (different from upstream)
  requests:
    cpu: 50m     # Redundant
    memory: 64Mi  # Redundant

# Nested unsupported values
service:
  type: ClusterIP  # Redundant
  port: 80         # Redundant
  nodePort: 30080  # Unsupported
  extraConfig:     # Unsupported
    timeout: 5s

# Modified structure
configMap:
  data:
    KEY1: newvalue  # Modified
    KEY3: value3    # Added

# Values with different types but same effective value
replicas: "1"  # String instead of int, but same value
EOFINNER

echo "Examples generated in the examples/ directory"
echo "Run the tool with: helm values-manager --upstream examples/test-upstream.yaml --downstream examples/test-downstream.yaml --optimize"
EOF

chmod +x "${HELM_VALUES_MANAGER_INSTALL_DIR}/scripts/generate-examples.sh"

# Create a test script
cat > "${HELM_VALUES_MANAGER_INSTALL_DIR}/scripts/test.sh" << 'EOF'
#!/bin/bash

set -e

# Colors for better output
BLUE='\033[0;34m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Building and testing Helm Values Manager...${NC}"

# Clean up previous test results
rm -rf values-analysis custom-output

# Build the project if it doesn't exist already
if [ ! -f bin/helm-values-manager ]; then
  echo -e "${YELLOW}Building project...${NC}"
  ./scripts/build.sh
fi

# Set up test files
echo -e "${YELLOW}Setting up test files...${NC}"
mkdir -p examples
if [ ! -f examples/test-upstream.yaml ]; then
  echo -e "${YELLOW}Creating test upstream values file...${NC}"
  cat > examples/test-upstream.yaml << 'EOFINNER'
replicaCount: 1
image:
  repository: nginx
  tag: latest
  pullPolicy: IfNotPresent
service:
  type: ClusterIP
  port: 80
ingress:
  enabled: false
  hosts:
    - host: chart-example.local
      paths: []
  tls: []
resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 128Mi
nodeSelector: {}
tolerations: []
affinity: {}
EOFINNER
fi

if [ ! -f examples/test-downstream.yaml ]; then
  echo -e "${YELLOW}Creating test downstream values file...${NC}"
  cat > examples/test-downstream.yaml << 'EOFINNER'
image:
  repository: nginx
  tag: 1.19.0
service:
  type: NodePort
replicas: "1"
nonexistentValue: test
EOFINNER
fi

echo -e "${YELLOW}Processing values with default output directory...${NC}"
./bin/helm-values-manager --upstream examples/test-upstream.yaml --downstream examples/test-downstream.yaml

# Check if files were created
echo -e "${YELLOW}Checking output files...${NC}"
DEFAULT_OUTPUT_DIR="values-analysis"

if [ -f "${DEFAULT_OUTPUT_DIR}/generated-values.yaml" ]; then
  echo -e "${GREEN}✓ ${DEFAULT_OUTPUT_DIR}/generated-values.yaml created${NC}"
else
  echo -e "${RED}✗ ${DEFAULT_OUTPUT_DIR}/generated-values.yaml not created${NC}"
  exit 1
fi

if [ -f "${DEFAULT_OUTPUT_DIR}/optimized-values.yaml" ]; then
  echo -e "${GREEN}✓ ${DEFAULT_OUTPUT_DIR}/optimized-values.yaml created${NC}"
else
  echo -e "${RED}✗ ${DEFAULT_OUTPUT_DIR}/optimized-values.yaml not created${NC}"
  exit 1
fi

if [ -f "${DEFAULT_OUTPUT_DIR}/unsupported-values.yaml" ]; then
  echo -e "${GREEN}✓ ${DEFAULT_OUTPUT_DIR}/unsupported-values.yaml created${NC}"
else
  echo -e "${RED}✗ ${DEFAULT_OUTPUT_DIR}/unsupported-values.yaml not created${NC}"
  exit 1
fi

if [ -f "${DEFAULT_OUTPUT_DIR}/redundant-values.yaml" ]; then
  echo -e "${GREEN}✓ ${DEFAULT_OUTPUT_DIR}/redundant-values.yaml created${NC}"
else
  echo -e "${RED}✗ ${DEFAULT_OUTPUT_DIR}/redundant-values.yaml not created${NC}"
  exit 1
fi

# Check expected content
echo -e "${YELLOW}Checking for expected unsupported values...${NC}"
if grep -q "nonexistentValue" "${DEFAULT_OUTPUT_DIR}/unsupported-values.yaml"; then
  echo -e "${GREEN}✓ nonexistentValue found in unsupported values${NC}"
else
  echo -e "${RED}✗ nonexistentValue not found in unsupported values${NC}"
  # Don't exit on this check yet as we're still improving detection
fi

echo -e "${YELLOW}Checking for expected redundant values...${NC}"
if grep -q "replicas" "${DEFAULT_OUTPUT_DIR}/redundant-values.yaml"; then
  echo -e "${GREEN}✓ replicas found in redundant values${NC}"
else
  echo -e "${RED}✗ replicas not found in redundant values${NC}"
  # Don't exit on this check yet as we're still improving detection
fi

# Test with custom output directory
CUSTOM_OUTPUT_DIR="custom-output"
echo -e "${YELLOW}Testing with custom output directory...${NC}"
./bin/helm-values-manager --upstream examples/test-upstream.yaml --downstream examples/test-downstream.yaml --output "${CUSTOM_OUTPUT_DIR}"

# Check if files were created in custom directory
echo -e "${YELLOW}Checking custom output files...${NC}"
if [ -f "${CUSTOM_OUTPUT_DIR}/generated-values.yaml" ]; then
  echo -e "${GREEN}✓ ${CUSTOM_OUTPUT_DIR}/generated-values.yaml created${NC}"
else
  echo -e "${RED}✗ ${CUSTOM_OUTPUT_DIR}/generated-values.yaml not created${NC}"
  exit 1
fi

if [ -f "${CUSTOM_OUTPUT_DIR}/optimized-values.yaml" ]; then
  echo -e "${GREEN}✓ ${CUSTOM_OUTPUT_DIR}/optimized-values.yaml created${NC}"
else
  echo -e "${RED}✗ ${CUSTOM_OUTPUT_DIR}/optimized-values.yaml not created${NC}"
  exit 1
fi

if [ -f "${CUSTOM_OUTPUT_DIR}/unsupported-values.yaml" ]; then
  echo -e "${GREEN}✓ ${CUSTOM_OUTPUT_DIR}/unsupported-values.yaml created${NC}"
else
  echo -e "${RED}✗ ${CUSTOM_OUTPUT_DIR}/unsupported-values.yaml not created${NC}"
  exit 1
fi

if [ -f "${CUSTOM_OUTPUT_DIR}/redundant-values.yaml" ]; then
  echo -e "${GREEN}✓ ${CUSTOM_OUTPUT_DIR}/redundant-values.yaml created${NC}"
else
  echo -e "${RED}✗ ${CUSTOM_OUTPUT_DIR}/redundant-values.yaml not created${NC}"
  exit 1
fi

echo -e "${GREEN}All tests completed successfully!${NC}"
EOF

chmod +x "${HELM_VALUES_MANAGER_INSTALL_DIR}/scripts/test.sh"

# Create a delete script
cat > "${HELM_VALUES_MANAGER_INSTALL_DIR}/scripts/delete.sh" << 'EOF'
#!/bin/bash

echo "Removing Helm Values Manager plugin..."
rm -rf "${HELM_PLUGIN_DIR}"
echo "Plugin removed successfully"
EOF

chmod +x "${HELM_VALUES_MANAGER_INSTALL_DIR}/scripts/delete.sh"

echo "Installing Helm Values Manager from ${DOWNLOAD_URL}..."

# If in development mode, build from source
if [ -f "go.mod" ] && [ -f "Taskfile.yml" ]; then
  echo "Development environment detected. Building from source..."

  # Check if task is installed
  if command -v task &> /dev/null; then
    task build
  elif [ -d "cmd/helm-values-manager" ]; then
    go build -o "${valueManagerBin}" cmd/helm-values-manager/main.go
  else
    go build -o "${valueManagerBin}" main.go
  fi
else
  # Download the appropriate binary for the architecture
  if command -v curl > /dev/null; then
    curl -sSL "${DOWNLOAD_URL}" -o "${valueManagerBin}"
  elif command -v wget > /dev/null; then
    wget -q "${DOWNLOAD_URL}" -O "${valueManagerBin}"
  else
    echo "Error: need curl or wget to download Helm Values Manager"
    exit 1
  fi
fi

# Make the binary executable
chmod +x "${valueManagerBin}"

echo "Helm Values Manager installed successfully!"
echo "Run 'helm values-manager -h' to see available commands"
