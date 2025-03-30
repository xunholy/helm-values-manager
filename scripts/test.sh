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
# We're going to use the existing test files if they exist
# but we'll check to make sure they're there
if [ ! -f examples/test-upstream.yaml ] || [ ! -f examples/test-downstream.yaml ]; then
  echo -e "${RED}Required test files not found. Please ensure both examples/test-upstream.yaml and examples/test-downstream.yaml exist.${NC}"
  exit 1
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

echo -e "${YELLOW}Testing with a custom output directory...${NC}"
CUSTOM_OUTPUT_DIR="custom-output"
./bin/helm-values-manager --upstream examples/test-upstream.yaml --downstream examples/test-downstream.yaml --outdir "${CUSTOM_OUTPUT_DIR}"

# Check if files were created in the custom directory
echo -e "${YELLOW}Checking output files in custom directory...${NC}"
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
