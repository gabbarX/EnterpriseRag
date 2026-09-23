#!/bin/bash
# Build all of EnterpriseRag's Docker images from source.

# Colours
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No colour

# Project root (the parent of the directory holding this script)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Enable BuildKit
export DOCKER_BUILDKIT=1

# Version information
VERSION="1.0.0"
SCRIPT_NAME=$(basename "$0")

# Print the help text
show_help() {
    echo -e "${GREEN}EnterpriseRag image build script v${VERSION}${NC}"
    echo -e "${GREEN}Usage:${NC} $0 [options]"
    echo "Options:"
    echo "  -h, --help     Show this help text"
    echo "  -a, --all      Build all images (default)"
    echo "  -p, --app      Build the application image only"
    echo "  -d, --docreader Build the document reader image only"
    echo "  -f, --frontend Build the frontend image only"
    echo "  -s, --sandbox  Build the sandbox image only"
    echo "  -c, --clean    Remove all local images"
    echo "  -v, --version  Show the version"
    exit 0
}

# Print the version
show_version() {
    echo -e "${GREEN}EnterpriseRag image build script v${VERSION}${NC}"
    exit 0
}

# Logging helpers
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Check whether Docker is installed
check_docker() {
    log_info "Checking the Docker environment..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed; please install Docker first"
        return 1
    fi
    
    # Check that the Docker daemon is running
    if ! docker info &> /dev/null; then
        log_error "The Docker service is not running; please start Docker"
        return 1
    fi
    
    log_success "Docker environment check passed"
    return 0
}

# Detect the platform
check_platform() {
    log_info "Detecting platform information..."
    if [ "$(uname -m)" = "x86_64" ]; then
        export PLATFORM="linux/amd64"
        export TARGETARCH="amd64"
    elif [ "$(uname -m)" = "aarch64" ] || [ "$(uname -m)" = "arm64" ]; then
        export PLATFORM="linux/arm64"
        export TARGETARCH="arm64"
    else
        log_warning "Unrecognised platform: $(uname -m); falling back to linux/amd64"
        export PLATFORM="linux/amd64"
        export TARGETARCH="amd64"
    fi
    log_info "Current platform: $PLATFORM"
    log_info "Current architecture: $TARGETARCH"
}

# Collect the version information
get_version_info() {
    # Read the version number from the VERSION file
    if [ -f "VERSION" ]; then
        VERSION=$(cat VERSION | tr -d '\n\r')
    else
        VERSION="unknown"
    fi
    
    # Read the commit ID
    if command -v git >/dev/null 2>&1; then
        COMMIT_ID=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    else
        COMMIT_ID="unknown"
    fi
    
    # Record the build time
    BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
    
    # Read the Go version
    if command -v go >/dev/null 2>&1; then
        GO_VERSION=$(go version 2>/dev/null || echo "unknown")
    else
        GO_VERSION="unknown"
    fi
    
    log_info "Version: $VERSION"
    log_info "Commit ID: $COMMIT_ID"
    log_info "Build time: $BUILD_TIME"
    log_info "Go version: $GO_VERSION"
}

# Build the application image
build_app_image() {
    log_info "Building the application image (enterpriserag-app)..."
    
    cd "$PROJECT_ROOT"
    
    # Collect the version information
    get_version_info
    
    docker build \
        --platform $PLATFORM \
        --build-arg GOPRIVATE_ARG=${GOPRIVATE:-""} \
        --build-arg GOPROXY_ARG=${GOPROXY:-"https://goproxy.cn,direct"} \
        --build-arg GOSUMDB_ARG=${GOSUMDB:-"off"} \
        --build-arg VERSION_ARG="$VERSION" \
        --build-arg COMMIT_ID_ARG="$COMMIT_ID" \
        --build-arg BUILD_TIME_ARG="$BUILD_TIME" \
        --build-arg GO_VERSION_ARG="$GO_VERSION" \
        --build-arg WITH_ANYDOC=${WITH_ANYDOC:-1} \
        -f docker/Dockerfile.app \
        -t ORG_PLACEHOLDER/enterpriserag-app:latest \
        .
    
    if [ $? -eq 0 ]; then
        log_success "Application image built successfully"
        return 0
    else
        log_error "Application image build failed"
        return 1
    fi
}

# Build the document reader image
build_docreader_image() {
    log_info "Building the document reader image (enterpriserag-docreader)..."
    
    cd "$PROJECT_ROOT"
    
    docker build \
        --platform $PLATFORM \
        --build-arg PLATFORM=$PLATFORM \
        --build-arg TARGETARCH=$TARGETARCH \
        --build-arg APT_MIRROR=${APT_MIRROR:-} \
        -f docker/Dockerfile.docreader \
        -t ORG_PLACEHOLDER/enterpriserag-docreader:latest \
        .
    
    if [ $? -eq 0 ]; then
        log_success "Document reader image built successfully"
        return 0
    else
        log_error "Document reader image build failed"
        return 1
    fi
}

# Build the frontend image (multi-stage: npm runs inside the builder, so no host-side dist build is needed)
build_frontend_image() {
    log_info "Building the frontend image (enterpriserag-ui)..."
    
    cd "$PROJECT_ROOT"
    
    # Collect the version information (used to inject the frontend commit hash)
    get_version_info

    docker build \
        --platform $PLATFORM \
        --build-arg VITE_FRONTEND_COMMIT="$COMMIT_ID" \
        ${NPM_REGISTRY:+--build-arg NPM_REGISTRY="$NPM_REGISTRY"} \
        ${NODE_MAX_OLD_SPACE_SIZE:+--build-arg NODE_MAX_OLD_SPACE_SIZE="$NODE_MAX_OLD_SPACE_SIZE"} \
        -f frontend/Dockerfile \
        -t ORG_PLACEHOLDER/enterpriserag-ui:latest \
        frontend/
    
    if [ $? -eq 0 ]; then
        log_success "Frontend image built successfully"
        return 0
    else
        log_error "Frontend image build failed"
        return 1
    fi
}

# Build the sandbox image
build_sandbox_image() {
    log_info "Building the sandbox image (enterpriserag-sandbox)..."

    cd "$PROJECT_ROOT"

    # Also tag it as main: the default image tracks main (see DefaultDockerImage in sandbox.go),
    # and the Docker backend only pulls when the image is missing locally, so skipping this tag wastes the build.
    docker build \
        --platform $PLATFORM \
        --build-arg TARGETPLATFORM=$PLATFORM \
        -f docker/Dockerfile.sandbox \
        --target sandbox \
        -t ORG_PLACEHOLDER/enterpriserag-sandbox:latest \
        -t ORG_PLACEHOLDER/enterpriserag-sandbox:main \
        .

    if [ $? -ne 0 ]; then
        log_error "Sandbox image build failed"
        return 1
    fi

    # Cube builds its template straight from the image and probes :49983/health, so a
    # missing envd always fails; Cube therefore uses a variant image with envd baked in.
    # Pinned to linux/amd64: cubesandbox-base, the source image for envd, has no arm64 build.
    log_info "Building the Cube variant of the sandbox image (enterpriserag-sandbox:main-cube)..."

    docker build \
        --platform linux/amd64 \
        --build-arg TARGETPLATFORM=linux/amd64 \
        --build-arg TARGETARCH=amd64 \
        -f docker/Dockerfile.sandbox \
        --target cube \
        -t ORG_PLACEHOLDER/enterpriserag-sandbox:latest-cube \
        -t ORG_PLACEHOLDER/enterpriserag-sandbox:main-cube \
        .

    if [ $? -ne 0 ]; then
        log_error "Sandbox image Cube variant build failed"
        return 1
    fi

    # Desktop variant: XFCE + x11vnc + websockify. Tagged for E2B template
    # builds; the Docker backend does not consume this image yet.
    log_info "Building the desktop variant of the sandbox image (enterpriserag-sandbox:main-desktop)..."

    docker build \
        --platform $PLATFORM \
        --build-arg TARGETPLATFORM=$PLATFORM \
        -f docker/Dockerfile.sandbox \
        --target desktop \
        -t ORG_PLACEHOLDER/enterpriserag-sandbox:latest-desktop \
        -t ORG_PLACEHOLDER/enterpriserag-sandbox:main-desktop \
        .

    if [ $? -ne 0 ]; then
        log_error "Sandbox image desktop variant build failed"
        return 1
    fi

    log_info "Building the desktop Cube variant of the sandbox image (enterpriserag-sandbox:main-desktop-cube)..."

    docker build \
        --platform linux/amd64 \
        --build-arg TARGETPLATFORM=linux/amd64 \
        --build-arg TARGETARCH=amd64 \
        -f docker/Dockerfile.sandbox \
        --target desktop-cube \
        -t ORG_PLACEHOLDER/enterpriserag-sandbox:latest-desktop-cube \
        -t ORG_PLACEHOLDER/enterpriserag-sandbox:main-desktop-cube \
        .

    if [ $? -eq 0 ]; then
        log_success "Sandbox image built successfully"
        return 0
    else
        log_error "Sandbox image desktop Cube variant build failed"
        return 1
    fi
}

# Build all images
build_all_images() {
    log_info "Building all images..."

    local app_result=0
    local docreader_result=0
    local frontend_result=0
    local sandbox_result=0

    # Build the application image
    build_app_image
    app_result=$?

    # Build the document reader image
    build_docreader_image
    docreader_result=$?

    # Build the frontend image
    build_frontend_image
    frontend_result=$?

    # Build the sandbox image
    build_sandbox_image
    sandbox_result=$?

    # Show the build results
    echo ""
    log_info "=== Build results ==="
    if [ $app_result -eq 0 ]; then
        log_success "\u2713 Application image built successfully"
    else
        log_error "\u2717 Application image build failed"
    fi

    if [ $docreader_result -eq 0 ]; then
        log_success "\u2713 Document reader image built successfully"
    else
        log_error "\u2717 Document reader image build failed"
    fi

    if [ $frontend_result -eq 0 ]; then
        log_success "\u2713 Frontend image built successfully"
    else
        log_error "\u2717 Frontend image build failed"
    fi

    if [ $sandbox_result -eq 0 ]; then
        log_success "\u2713 Sandbox image built successfully"
    else
        log_error "\u2717 Sandbox image build failed"
    fi

    if [ $app_result -eq 0 ] && [ $docreader_result -eq 0 ] && [ $frontend_result -eq 0 ] && [ $sandbox_result -eq 0 ]; then
        log_success "All images built successfully."
        return 0
    else
        log_error "Some images failed to build"
        return 1
    fi
}

# Remove the local images
clean_images() {
    log_info "Removing the local EnterpriseRag images..."
    
    # Stop the related containers
    log_info "Stopping the related containers..."
    docker stop $(docker ps -q --filter "ancestor=ORG_PLACEHOLDER/enterpriserag-app:latest" 2>/dev/null) 2>/dev/null || true
    docker stop $(docker ps -q --filter "ancestor=ORG_PLACEHOLDER/enterpriserag-docreader:latest" 2>/dev/null) 2>/dev/null || true
    docker stop $(docker ps -q --filter "ancestor=ORG_PLACEHOLDER/enterpriserag-ui:latest" 2>/dev/null) 2>/dev/null || true
    
    # Remove the related containers
    log_info "Removing the related containers..."
    docker rm $(docker ps -aq --filter "ancestor=ORG_PLACEHOLDER/enterpriserag-app:latest" 2>/dev/null) 2>/dev/null || true
    docker rm $(docker ps -aq --filter "ancestor=ORG_PLACEHOLDER/enterpriserag-docreader:latest" 2>/dev/null) 2>/dev/null || true
    docker rm $(docker ps -aq --filter "ancestor=ORG_PLACEHOLDER/enterpriserag-ui:latest" 2>/dev/null) 2>/dev/null || true
    
    # Remove the images
    log_info "Removing the local images..."
    docker rmi ORG_PLACEHOLDER/enterpriserag-app:latest 2>/dev/null || true
    docker rmi ORG_PLACEHOLDER/enterpriserag-docreader:latest 2>/dev/null || true
    docker rmi ORG_PLACEHOLDER/enterpriserag-ui:latest 2>/dev/null || true
    docker rmi ORG_PLACEHOLDER/enterpriserag-sandbox:latest 2>/dev/null || true
    docker rmi ORG_PLACEHOLDER/enterpriserag-sandbox:latest-cube 2>/dev/null || true
    docker rmi ORG_PLACEHOLDER/enterpriserag-sandbox:latest-desktop 2>/dev/null || true
    docker rmi ORG_PLACEHOLDER/enterpriserag-sandbox:latest-desktop-cube 2>/dev/null || true
    docker rmi ORG_PLACEHOLDER/enterpriserag-sandbox:main 2>/dev/null || true
    docker rmi ORG_PLACEHOLDER/enterpriserag-sandbox:main-cube 2>/dev/null || true
    docker rmi ORG_PLACEHOLDER/enterpriserag-sandbox:main-desktop 2>/dev/null || true
    docker rmi ORG_PLACEHOLDER/enterpriserag-sandbox:main-desktop-cube 2>/dev/null || true
    
    docker image prune -f
    
    log_success "Image clean-up complete"
    return 0
}

# Parse the command line arguments
BUILD_ALL=false
BUILD_APP=false
BUILD_DOCREADER=false
BUILD_FRONTEND=false
BUILD_SANDBOX=false
CLEAN_IMAGES=false

# With no arguments, build every image by default
if [ $# -eq 0 ]; then
    BUILD_ALL=true
fi

while [ "$1" != "" ]; do
    case $1 in
        -h | --help )       show_help
                            ;;
        -a | --all )        BUILD_ALL=true
                            ;;
        -p | --app )        BUILD_APP=true
                            ;;
        -d | --docreader )  BUILD_DOCREADER=true
                            ;;
        -f | --frontend )   BUILD_FRONTEND=true
                            ;;
        -s | --sandbox )    BUILD_SANDBOX=true
                            ;;
        -c | --clean )      CLEAN_IMAGES=true
                            ;;
        -v | --version )    show_version
                            ;;
        * )                 log_error "Unknown option: $1"
                            show_help
                            ;;
    esac
    shift
done

# Check the Docker environment
check_docker
if [ $? -ne 0 ]; then
    exit 1
fi

# Detect the platform
check_platform

# Run the clean-up
if [ "$CLEAN_IMAGES" = true ]; then
    clean_images
    exit $?
fi

# Run the build
if [ "$BUILD_ALL" = true ]; then
    build_all_images
    exit $?
fi

if [ "$BUILD_APP" = true ]; then
    build_app_image
    exit $?
fi

if [ "$BUILD_DOCREADER" = true ]; then
    build_docreader_image
    exit $?
fi

if [ "$BUILD_FRONTEND" = true ]; then
    build_frontend_image
    exit $?
fi

if [ "$BUILD_SANDBOX" = true ]; then
    build_sandbox_image
    exit $?
fi

exit 0
