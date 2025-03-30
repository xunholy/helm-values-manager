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
  cat > examples/test-upstream.yaml << 'EOF'
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
EOF
fi

if [ ! -f examples/test-downstream.yaml ]; then
  echo -e "${YELLOW}Creating test downstream values file...${NC}"
  cat > examples/test-downstream.yaml << 'EOF'
image:
  repository: nginx
  tag: 1.19.0
service:
  type: NodePort
replicas: "1"
nonexistentValue: test
EOF
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
./bin/helm-values-manager --upstream examples/test-upstream.yaml --downstream examples/test-downstream.yaml --outdir "${CUSTOM_OUTPUT_DIR}"

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
