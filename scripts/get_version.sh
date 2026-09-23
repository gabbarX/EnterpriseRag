#!/bin/bash
# Single source of version information
# Works for both local and CI builds

# Defaults
VERSION="unknown"
EDITION="${EDITION:-standard}"
COMMIT_ID="unknown"
BUILD_TIME="unknown"
GO_VERSION="unknown"

# Version number
if [ -f "VERSION" ]; then
    VERSION=$(cat VERSION | tr -d '\n\r')
fi

# Commit ID
if [ -n "$GITHUB_SHA" ]; then
    # GitHub Actions environment
    COMMIT_ID="${GITHUB_SHA:0:7}"
elif command -v git >/dev/null 2>&1; then
    # Local environment
    COMMIT_ID=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
fi

# Build time
if [ -n "$GITHUB_ACTIONS" ]; then
    # GitHub Actions environment; use the standard time format
    BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
else
    # Local environment
    BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
fi

# Go version
if command -v go >/dev/null 2>&1; then
    GO_VERSION=$(go version 2>/dev/null || echo "unknown")
fi

# Print a different format depending on the argument
case "${1:-env}" in
    "env")
        # Environment variable format; values containing spaces are quoted
        echo "VERSION=$VERSION"
        echo "EDITION=$EDITION"
        echo "COMMIT_ID=$COMMIT_ID"
        echo "BUILD_TIME=\"$BUILD_TIME\""
        echo "GO_VERSION=\"$GO_VERSION\""
        ;;
    "json")
        # JSON format
        cat << EOF
{
  "version": "$VERSION",
  "edition": "$EDITION",
  "commit_id": "$COMMIT_ID",
  "build_time": "$BUILD_TIME",
  "go_version": "$GO_VERSION"
}
EOF
        ;;
    "docker-args")
        # Docker build argument format
        echo "--build-arg VERSION_ARG=$VERSION"
        echo "--build-arg COMMIT_ID_ARG=$COMMIT_ID"
        echo "--build-arg BUILD_TIME_ARG=$BUILD_TIME"
        echo "--build-arg GO_VERSION_ARG=$GO_VERSION"
        ;;
    "ldflags")
        # Go ldflags format
        echo "-X 'github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/handler.Version=$VERSION' -X 'github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/handler.Edition=$EDITION' -X 'github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/handler.CommitID=$COMMIT_ID' -X 'github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/handler.BuildTime=$BUILD_TIME' -X 'github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/handler.GoVersion=$GO_VERSION'"
        ;;
    "info")
        # Human-readable format
        echo "Version: $VERSION"
        echo "Edition: $EDITION"
        echo "Commit ID: $COMMIT_ID"
        echo "Build time: $BUILD_TIME"
        echo "Go version: $GO_VERSION"
        ;;
    *)
        echo "Usage: $0 [env|json|docker-args|ldflags|info]"
        echo "  env        - environment variable format (default)"
        echo "  json       - JSON format"
        echo "  docker-args - Docker build argument format"
        echo "  ldflags    - Go ldflags format"
        echo "  info       - human-readable format"
        exit 1
        ;;
esac
