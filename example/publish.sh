#!/bin/bash

set -e

# Check if version parameter is provided
if [ -z "$1" ]; then
    echo "Error: Version parameter is required"
    echo "Usage: $0 <version>"
    echo "Example: $0 2.0.0"
    exit 1
fi

VERSION="$1"
BINARY_NAME="testapp"
REGISTRY="ghcr.io/zeitlos/knockknock"
IMAGE_REF="${REGISTRY}/${BINARY_NAME}:${VERSION}"

echo "Building ${BINARY_NAME} v${VERSION}"
echo ""

# Build the binary with version information
go build -ldflags="-X 'main.Version=${VERSION}'" -o "${BINARY_NAME}"

echo ""
echo "Publishing ${BINARY_NAME} to ${IMAGE_REF}"
echo ""

# Push the binary using ORAS
oras push "${IMAGE_REF}" \
    "${BINARY_NAME}:application/vnd.unknown.layer.v1+binary"

rm $BINARY_NAME

echo ""
echo "Successfully published ${BINARY_NAME} v${VERSION}"
echo "Pull with: oras pull ${IMAGE_REF}"