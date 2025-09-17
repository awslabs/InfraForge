#!/bin/bash

# InfraForge Build Script
# Usage: ./scripts/build.sh

set -e

# Download Go modules from project root
echo "Downloading Go modules..."
go mod download
echo "✓ Go modules downloaded"
echo

cd cmd/infraforge

# Get version information
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")

echo "Building InfraForge ${VERSION}..."
echo "Build Date: ${BUILD_DATE}"
echo "Git Commit: ${GIT_COMMIT}"
echo

# Static build with version information
CGO_ENABLED=0 go build -ldflags="-s -w -extldflags=-static -X main.Version=${VERSION} -X main.BuildDate=${BUILD_DATE} -X main.GitCommit=${GIT_COMMIT}" -o infraforge

# Create version file
echo "${VERSION}" > infraforge-version.txt

echo "✓ Build completed: infraforge ${VERSION}"
echo "✓ Version file created: infraforge-version.txt"
echo
echo "Test version info:"
./infraforge --version
