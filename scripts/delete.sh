#!/bin/bash

set -e

# Colors for output
GREEN="\033[0;32m"
RED="\033[0;31m"
RESET="\033[0m"

echo -e "${GREEN}Removing Helm Values Manager plugin...${RESET}"

# Check if plugin directory exists
if [ -z "$HELM_PLUGIN_DIR" ]; then
    echo -e "${RED}Error: HELM_PLUGIN_DIR environment variable not set.${RESET}"
    exit 1
fi

# Remove the plugin directory
if [ -d "$HELM_PLUGIN_DIR" ]; then
    rm -rf "$HELM_PLUGIN_DIR"
    echo -e "${GREEN}Plugin removed successfully from $HELM_PLUGIN_DIR${RESET}"
else
    echo -e "${RED}Plugin directory not found at $HELM_PLUGIN_DIR${RESET}"
    exit 1
fi

echo -e "${GREEN}Helm Values Manager has been uninstalled.${RESET}"
