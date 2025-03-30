#!/bin/bash

set -e

# Colors for output
GREEN="\033[0;32m"
RED="\033[0;31m"
YELLOW="\033[0;33m"
RESET="\033[0m"

# Determine version
if [ -d .git ]; then
    VERSION=$(git describe --tags --always --dirty)
else
    VERSION="dev"
fi

echo -e "${GREEN}Building helm-values-manager version ${VERSION}...${RESET}"

# Create output directory
mkdir -p bin

# Check if we're using the new module structure
if [ -d "cmd/helm-values-manager" ]; then
    echo -e "${YELLOW}Using new module structure${RESET}"
    go build -ldflags "-X main.version=${VERSION}" -o bin/helm-values-manager cmd/helm-values-manager/main.go

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Build successful!${RESET}"
        echo -e "Binary created at: bin/helm-values-manager"
        echo ""

        # Check if docker is installed
        if command -v docker &> /dev/null; then
            echo -e "To build a Docker image run:"
            echo -e "docker build -t helm-values-manager:${VERSION} ."
        fi

        chmod +x bin/helm-values-manager
    else
        echo -e "${RED}Build failed!${RESET}"
        exit 1
    fi
else
    echo -e "${YELLOW}Using legacy structure${RESET}"
    go build -ldflags "-X main.version=${VERSION}" -o bin/helm-values-manager main.go

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Build successful!${RESET}"
        echo -e "Binary created at: bin/helm-values-manager"
        echo ""

        # Check if docker is installed
        if command -v docker &> /dev/null; then
            echo -e "To build a Docker image run:"
            echo -e "docker build -t helm-values-manager:${VERSION} ."
        fi

        chmod +x bin/helm-values-manager
    else
        echo -e "${RED}Build failed!${RESET}"
        exit 1
    fi
fi
