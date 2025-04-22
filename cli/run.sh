#!/bin/bash

# Set the SDK path for CGO
export SDKROOT=$(xcrun --sdk macosx --show-sdk-path)

# Set the configuration path
export SYNC_MANAGER_CONFIG="$(dirname "$(pwd)")/sync-manager.yaml"

# Run the CLI with Air for hot reloading
air -- "$@" 