#!/bin/bash
# Start or stop the Ollama and docker-compose services on demand.

# Colours
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No colour

# Project root (the parent of the directory holding this script)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Version information
VERSION="1.0.1" # Bumped on each update
SCRIPT_NAME=$(basename "$0")

# Print the help text
show_help() {
    printf "%b\n" "${GREEN}EnterpriseRag start-up script v${VERSION}${NC}"
    printf "%b\n" "${GREEN}Usage:${NC} $0 [options]"
    echo "Options:"
    echo "  -h, --help     Show this help text"
    echo "  -o, --ollama   Start the Ollama service"
    echo "  -d, --docker   Start the Docker container services"
    echo "  -a, --all      Start all services (default)"
    echo "  -s, --stop     Stop all services"
    echo "  -c, --check    Check the environment and diagnose problems"
    echo "  -r, --restart  Rebuild and restart the named container"
    echo "  -l, --list     List all running containers"
    echo "  -p, --pull     Pull the latest Docker images"
    echo "  --no-pull      Do not pull images on start-up (pulling is the default)"
    echo "  -v, --version  Show the version"
    exit 0
}

# Print the version
show_version() {
    printf "%b\n" "${GREEN}EnterpriseRag start-up script v${VERSION}${NC}"
    exit 0
}

# Logging helpers
log_info() {
    printf "%b\n" "${BLUE}[INFO]${NC} $1"
}

log_warning() {
    printf "%b\n" "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    printf "%b\n" "${RED}[ERROR]${NC} $1"
}

log_success() {
    printf "%b\n" "${GREEN}[SUCCESS]${NC} $1"
}

# Inject git short hash into the frontend image when composing from source.
# docker-compose.yml interpolates VITE_FRONTEND_COMMIT; the build context is
# frontend/ (no .git), so Vite cannot discover the commit on its own.
export_frontend_build_args() {
    if [ -n "${VITE_FRONTEND_COMMIT:-}" ]; then
        export VITE_FRONTEND_COMMIT
        return 0
    fi
    # shellcheck source=/dev/null
    eval "$("$PROJECT_ROOT/scripts/get_version.sh" env)"
    export VITE_FRONTEND_COMMIT="${COMMIT_ID:-unknown}"
    log_info "VITE_FRONTEND_COMMIT=${VITE_FRONTEND_COMMIT}"
}

# Pick an available Docker Compose command (prefer 'docker compose', then 'docker-compose')
DOCKER_COMPOSE_BIN=""
DOCKER_COMPOSE_SUBCMD=""

detect_compose_cmd() {
	# Prefer the Docker Compose plugin
	if docker compose version &> /dev/null; then
		DOCKER_COMPOSE_BIN="docker"
		DOCKER_COMPOSE_SUBCMD="compose"
		return 0
	fi

	# Fall back to docker-compose (v1)
	if command -v docker-compose &> /dev/null; then
		if docker-compose version &> /dev/null; then
			DOCKER_COMPOSE_BIN="docker-compose"
			DOCKER_COMPOSE_SUBCMD=""
			return 0
		fi
	fi

	# Neither is available
	return 1
}

# Check for a .env file and create one if missing
check_env_file() {
    log_info "Checking the environment variable configuration..."
    if [ ! -f "$PROJECT_ROOT/.env" ]; then
        log_warning ".env not found; creating it from the template"
        if [ -f "$PROJECT_ROOT/.env.example" ]; then
            cp "$PROJECT_ROOT/.env.example" "$PROJECT_ROOT/.env"
            log_success "Created .env from .env.example"
        else
            log_error "Template .env.example not found; cannot create .env"
            return 1
        fi
    else
        log_info ".env already exists"
    fi
    
    # Check that the required environment variables are set
    source "$PROJECT_ROOT/.env"
    local missing_vars=()
    
    # Check the base variables
    if [ -z "$DB_DRIVER" ]; then missing_vars+=("DB_DRIVER"); fi
    if [ -z "$STORAGE_TYPE" ]; then missing_vars+=("STORAGE_TYPE"); fi
    
    return 0
}

# Install Ollama (the method varies by platform)
install_ollama() {
    # Check whether a remote service is configured
    get_ollama_base_url
    
    if [ $IS_REMOTE -eq 1 ]; then
        log_info "Remote Ollama service configured; no local installation needed"
        return 0
    fi

    log_info "Ollama is not installed locally; installing now..."
    
    OS=$(uname)
    if [ "$OS" = "Darwin" ]; then
        # macOS installation
        log_info "macOS detected; installing Ollama with brew..."
        if ! command -v brew &> /dev/null; then
            # Install from the downloaded package
            log_info "Homebrew not installed; falling back to a direct download..."
            curl -fsSL https://ollama.com/download/Ollama-darwin.zip -o ollama.zip
            unzip ollama.zip
            mv ollama /usr/local/bin
            rm ollama.zip
        else
            brew install ollama
        fi
    else
        # Linux installation
        log_info "Linux detected; using the install script..."
        curl -fsSL https://ollama.com/install.sh | sh
    fi
    
    if [ $? -eq 0 ]; then
        log_success "Local Ollama installation complete"
        return 0
    else
        log_error "Local Ollama installation failed"
        return 1
    fi
}

# Resolve the Ollama base URL and determine whether it is a remote service
get_ollama_base_url() {

    check_env_file

    # Read the Ollama base URL from the environment
    OLLAMA_URL=${OLLAMA_BASE_URL:-"http://host.docker.internal:11434"}
    # Extract the host
    OLLAMA_HOST=$(echo "$OLLAMA_URL" | sed -E 's|^https?://||' | sed -E 's|:[0-9]+$||' | sed -E 's|/.*$||')
    # Extract the port
    OLLAMA_PORT=$(echo "$OLLAMA_URL" | grep -oE ':[0-9]+' | grep -oE '[0-9]+' || echo "11434")
    # Check for localhost or 127.0.0.1
    IS_REMOTE=0
    if [ "$OLLAMA_HOST" = "localhost" ] || [ "$OLLAMA_HOST" = "127.0.0.1" ] || [ "$OLLAMA_HOST" = "host.docker.internal" ]; then
        IS_REMOTE=0  # Local service
    else
        IS_REMOTE=1  # Remote service
    fi
}

# Start the Ollama service
start_ollama() {
    log_info "Checking the Ollama service..."
    # Extract the host and port
    get_ollama_base_url
    log_info "Ollama service address: $OLLAMA_URL"
    
    if [ $IS_REMOTE -eq 1 ]; then
        log_info "Remote Ollama service detected; using it directly, skipping local installation and start-up"
        # Check that the remote service is reachable
        if curl -s "$OLLAMA_URL/api/tags" &> /dev/null; then
            log_success "Remote Ollama service is reachable"
            return 0
        else
            log_warning "Remote Ollama service is unreachable; check that the address is correct and the service is running"
            return 1
        fi
    fi
    
    # What follows handles the local service
    # Check whether Ollama is installed
    if ! command -v ollama &> /dev/null; then
        install_ollama
        if [ $? -ne 0 ]; then
            return 1
        fi
    fi

    # Check whether the Ollama service is already running
    if curl -s "http://localhost:$OLLAMA_PORT/api/tags" &> /dev/null; then
        log_success "Local Ollama service is already running on port $OLLAMA_PORT"
    else
        log_info "Starting the local Ollama service..."
        # Note: upstream recommends managing the service via systemctl or launchctl; running it in the background directly is for ad-hoc use only
        systemctl restart ollama || (ollama serve > /dev/null 2>&1 < /dev/null &)
        
        # Wait for the service to come up
        MAX_RETRIES=30
        COUNT=0
        while [ $COUNT -lt $MAX_RETRIES ]; do
            if curl -s "http://localhost:$OLLAMA_PORT/api/tags" &> /dev/null; then
                log_success "Local Ollama service started on port $OLLAMA_PORT"
                break
            fi
            echo -ne "Waiting for the Ollama service to start... ($COUNT/$MAX_RETRIES)\r"
            sleep 1
            COUNT=$((COUNT + 1))
        done
        echo "" # Newline
        
        if [ $COUNT -eq $MAX_RETRIES ]; then
            log_error "Failed to start the local Ollama service"
            return 1
        fi
    fi

    log_success "Local Ollama service address: http://localhost:$OLLAMA_PORT"
    return 0
}

# Stop the Ollama service
stop_ollama() {
    log_info "Stopping the Ollama service..."
    
    # Check whether a remote service is configured
    get_ollama_base_url
    
    if [ $IS_REMOTE -eq 1 ]; then
        log_info "Remote Ollama service detected; nothing to stop locally"
        return 0
    fi
    
    # Check whether Ollama is installed
    if ! command -v ollama &> /dev/null; then
        log_info "Ollama is not installed locally; nothing to stop"
        return 0
    fi
    
    # Find and terminate the Ollama process
    if pgrep -x "ollama" > /dev/null; then
        # Prefer systemctl
        if command -v systemctl &> /dev/null; then
            sudo systemctl stop ollama
        else
            pkill -f "ollama serve"
        fi
        log_success "Local Ollama service stopped"
    else
        log_info "Local Ollama service is not running"
    fi
    
    return 0
}

# Check whether Docker is installed
check_docker() {
    log_info "Checking the Docker environment..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed; please install Docker first"
        return 1
    fi
    
	# Detect and select an available Docker Compose command
	if detect_compose_cmd; then
		if [ "$DOCKER_COMPOSE_BIN" = "docker" ]; then
			log_info "Detected the Docker Compose plugin (docker compose)"
		else
			log_info "Detected docker-compose (v1)"
		fi
	else
		log_error "No Docker Compose found (neither 'docker compose' nor 'docker-compose'). Please install one of them."
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

check_platform() {
     # Detect the current platform
    log_info "Detecting platform information..."
    if [ "$(uname -m)" = "x86_64" ]; then
        export PLATFORM="linux/amd64"
    elif [ "$(uname -m)" = "aarch64" ] || [ "$(uname -m)" = "arm64" ]; then
        export PLATFORM="linux/arm64"
    else
        log_warning "Unrecognised platform: $(uname -m); falling back to linux/amd64"
        export PLATFORM="linux/amd64"
    fi
    log_info "Current platform: $PLATFORM"
}

# Pre-pull the sandbox image (required to run Agent Skills; pulled but not started)
ensure_sandbox_image() {
    local sandbox_image="ORG_PLACEHOLDER/enterpriserag-sandbox:${ENTERPRISERAG_VERSION:-latest}"

    # Check whether the sandbox image is already present locally
    if docker image inspect "$sandbox_image" &> /dev/null; then
        log_success "Sandbox image ready: $sandbox_image"
        return 0
    fi

    log_info "Sandbox image ($sandbox_image) not found; pulling in the background..."
    log_info "Agent Skills depend on this image; the pull must finish before the first run"

    # Pull in the background so the main flow is not blocked
    (
        if PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD --profile sandbox pull sandbox 2>/dev/null; then
            log_success "Sandbox image pull complete: $sandbox_image"
        else
            log_warning "Sandbox image pull failed; Agent Skills may be unavailable"
            log_warning "You can pull it manually later: $DOCKER_COMPOSE_BIN $DOCKER_COMPOSE_SUBCMD --profile sandbox pull sandbox"
        fi
    ) &

    return 0
}

# Start the Docker containers
start_docker() {
    log_info "Starting the Docker containers..."
    
    # Check the Docker environment
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    # Check the .env file
    check_env_file
    
    # Read the .env file
    source "$PROJECT_ROOT/.env"
    storage_type=${STORAGE_TYPE:-local}
    
    check_platform
    
	# Change into the project root before running docker-compose
    cd "$PROJECT_ROOT"

    export_frontend_build_args
    
    # Start the core services
    log_info "Starting the core service containers..."
	# Always start via the Compose command we detected
	if [ "$NO_PULL" = true ]; then
		# Do not pull; use the local images
		log_info "Skipping the image pull; using local images..."
		PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD up --build -d
	else
		# Pull the latest images
		log_info "Pulling the latest images..."
		PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD up --pull always -d
	fi
    if [ $? -ne 0 ]; then
        log_error "Failed to start the Docker containers"
        return 1
    fi
    
    log_success "All Docker containers started successfully"

    # Show the container status
    log_info "Current container status:"
	"$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD ps

    # Pre-pull the sandbox image (required to run Agent Skills; pulled but not started)
    ensure_sandbox_image

    return 0
}

# Stop the Docker containers
stop_docker() {
    log_info "Stopping the Docker containers..."
    
    # Check the Docker environment
    check_docker
    if [ $? -ne 0 ]; then
        # Try to stop the containers even if the check failed, just in case
        log_warning "Docker environment check failed; attempting to stop the containers anyway..."
    fi
    
    # Change into the project root before running docker-compose
    cd "$PROJECT_ROOT"
    
    # Stop all containers
	"$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD down --remove-orphans
    if [ $? -ne 0 ]; then
        log_error "Failed to stop the Docker containers"
        return 1
    fi
    
    log_success "All Docker containers stopped"
    return 0
}

# List all running containers
list_containers() {
    log_info "Listing all running containers..."
    
    # Check the Docker environment
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    # Change into the project root before running docker-compose
    cd "$PROJECT_ROOT"
    
    # List all containers
    printf "%b\n" "${BLUE}Currently running containers:${NC}"
	"$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD ps --services | sort
    
    return 0
}

# Pull the latest Docker images
pull_images() {
    log_info "Pulling the latest Docker images..."
    
    # Check the Docker environment
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    # Check the .env file
    check_env_file
    
    # Read the .env file
    source "$PROJECT_ROOT/.env"
    storage_type=${STORAGE_TYPE:-local}
    
    check_platform
    
    # Change into the project root before running docker-compose
    cd "$PROJECT_ROOT"
    
    # Pull all images
    log_info "Pulling the latest image for every service..."
	PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD pull
    if [ $? -ne 0 ]; then
        log_error "Image pull failed"
        return 1
    fi

    # Pull the sandbox image (it sits behind a profile, so it must be pulled separately)
    log_info "Pulling the sandbox image..."
    PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD --profile sandbox pull sandbox 2>/dev/null || \
        log_warning "Sandbox image pull failed (optional, skipping)"

    log_success "All images pulled to the latest version"
    
    # Show the images that were pulled
    log_info "Pulled images:"
    docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.CreatedAt}}\t{{.Size}}" | head -10
    
    return 0
}

# Restart the named container
restart_container() {
    local container_name="$1"
    
    if [ -z "$container_name" ]; then
        log_error "No container name given"
        echo "Available containers:"
        list_containers
        return 1
    fi
    
    log_info "Rebuilding and restarting container: $container_name"
    
    # Check the Docker environment
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    check_platform
    
    # Change into the project root before running docker-compose
    cd "$PROJECT_ROOT"

    export_frontend_build_args
    
    # Check whether the container exists
	if ! "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD ps --services | grep -q "^$container_name$"; then
        log_error "Container '$container_name' does not exist or is not running"
        echo "Available containers:"
        list_containers
        return 1
    fi
    
    # Build and restart the container
    log_info "Rebuilding container '$container_name'..."
	PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD build "$container_name"
    if [ $? -ne 0 ]; then
        log_error "Failed to build container '$container_name'"
        return 1
    fi
    
    log_info "Restarting container '$container_name'..."
	PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD up -d --no-deps "$container_name"
    if [ $? -ne 0 ]; then
        log_error "Failed to restart container '$container_name'"
        return 1
    fi
    
    log_success "Container '$container_name' rebuilt and restarted successfully"
    return 0
}

# Check the system environment
check_environment() {
    log_info "Starting the environment check..."
    
    # Check the operating system
    OS=$(uname)
    log_info "Operating system: $OS"
    
    # Check Docker
    check_docker
    
    # Check the .env file
    check_env_file
    
    get_ollama_base_url
    
    if [ $IS_REMOTE -eq 1 ]; then
        log_info "Remote Ollama service configuration detected"
        if curl -s "$OLLAMA_URL/api/tags" &> /dev/null; then
            version=$(curl -s "$OLLAMA_URL/api/tags" | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
            log_success "Remote Ollama service is reachable, version: $version"
        else
            log_warning "Remote Ollama service is unreachable; check that the address is correct and the service is running"
        fi
    else
        if command -v ollama &> /dev/null; then
            log_success "Ollama is installed locally"
            if curl -s "http://localhost:$OLLAMA_PORT/api/tags" &> /dev/null; then
                version=$(curl -s "http://localhost:$OLLAMA_PORT/api/tags" | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
                log_success "Local Ollama service is running, version: $version"
            else
                log_warning "Ollama is installed locally but the service is not running"
            fi
        else
            log_warning "Ollama is not installed locally"
        fi
    fi
    
    # Check the sandbox image
    log_info "Checking the sandbox image..."
    local sandbox_image="ORG_PLACEHOLDER/enterpriserag-sandbox:${ENTERPRISERAG_VERSION:-latest}"
    if docker image inspect "$sandbox_image" &> /dev/null; then
        log_success "Sandbox image ready: $sandbox_image"
    else
        log_warning "Sandbox image not found: $sandbox_image (Agent Skills require this image)"
        log_info "Pull it with: $0 -p or docker pull $sandbox_image"
    fi

    # Check the disk space
    log_info "Checking disk space..."
    df -h | grep -E "(Filesystem|/$)"
    
    # Check the memory
    log_info "Checking memory usage..."
    if [ "$OS" = "Darwin" ]; then
        vm_stat | perl -ne '/page size of (\d+)/ and $size=$1; /Pages free:\s*(\d+)/ and print "Free Memory: ", $1 * $size / 1048576, " MB\n"'
    else
        free -h | grep -E "(total|Mem:)"
    fi
    
    # Check the CPU
    log_info "CPU information:"
    if [ "$OS" = "Darwin" ]; then
        sysctl -n machdep.cpu.brand_string
        echo "CPU cores: $(sysctl -n hw.ncpu)"
    else
        grep "model name" /proc/cpuinfo | head -1
        echo "CPU cores: $(nproc)"
    fi
    
    # Check the container status
    log_info "Checking the container status..."
    if docker info &> /dev/null; then
        docker ps -a
    else
        log_warning "Could not read the container status; Docker may not be running"
    fi
    
    log_success "Environment check complete"
    return 0
}

# Parse the command line arguments
START_OLLAMA=false
START_DOCKER=false
STOP_SERVICES=false
CHECK_ENVIRONMENT=false
LIST_CONTAINERS=false
RESTART_CONTAINER=false
PULL_IMAGES=false
NO_PULL=false
CONTAINER_NAME=""

# With no arguments, start every service by default
if [ $# -eq 0 ]; then
    START_OLLAMA=true
    START_DOCKER=true
fi

while [ "$1" != "" ]; do
    case $1 in
        -h | --help )       show_help
                            ;;
        -o | --ollama )     START_OLLAMA=true
                            ;;
        -d | --docker )     START_DOCKER=true
                            ;;
        -a | --all )        START_OLLAMA=true
                            START_DOCKER=true
                            ;;
        -s | --stop )       STOP_SERVICES=true
                            ;;
        -c | --check )      CHECK_ENVIRONMENT=true
                            ;;
        -l | --list )       LIST_CONTAINERS=true
                            ;;
        -p | --pull )       PULL_IMAGES=true
                            ;;
        --no-pull )         NO_PULL=true
                            START_OLLAMA=true
                            START_DOCKER=true
                            ;;
        -r | --restart )    RESTART_CONTAINER=true
                            CONTAINER_NAME="$2"
                            shift
                            ;;
        -v | --version )    show_version
                            ;;
        * )                 log_error "Unknown option: $1"
                            show_help
                            ;;
    esac
    shift
done

# Run the environment check
if [ "$CHECK_ENVIRONMENT" = true ]; then
    check_environment
    exit $?
fi

# List all containers
if [ "$LIST_CONTAINERS" = true ]; then
    list_containers
    exit $?
fi

# Pull the latest images
if [ "$PULL_IMAGES" = true ]; then
    pull_images
    exit $?
fi

# Restart the named container
if [ "$RESTART_CONTAINER" = true ]; then
    restart_container "$CONTAINER_NAME"
    exit $?
fi

# Carry out the requested service operation
if [ "$STOP_SERVICES" = true ]; then
    # Stop the services
    stop_ollama
    OLLAMA_RESULT=$?
    
    stop_docker
    DOCKER_RESULT=$?
    
    # Print the summary
    echo ""
    log_info "=== Stop results ==="
    if [ $OLLAMA_RESULT -eq 0 ]; then
        log_success "\u2713 Ollama service stopped"
    else
        log_error "\u2717 Failed to stop the Ollama service"
    fi
    
    if [ $DOCKER_RESULT -eq 0 ]; then
        log_success "\u2713 Docker containers stopped"
    else
        log_error "\u2717 Failed to stop the Docker containers"
    fi
    
    log_success "Services stopped."
else
    # Start the services
    OLLAMA_RESULT=1
    DOCKER_RESULT=1
    if [ "$START_OLLAMA" = true ]; then
        start_ollama
        OLLAMA_RESULT=$?
    fi
    
    if [ "$START_DOCKER" = true ]; then
        start_docker
        DOCKER_RESULT=$?
    fi
    
    # Print the summary
    echo ""
    log_info "=== Start-up results ==="
    if [ "$START_OLLAMA" = true ]; then
        if [ $OLLAMA_RESULT -eq 0 ]; then
            log_success "\u2713 Ollama service started"
        else
            log_error "\u2717 Failed to start the Ollama service"
        fi
    fi
    
    if [ "$START_DOCKER" = true ]; then
        if [ $DOCKER_RESULT -eq 0 ]; then
            log_success "\u2713 Docker containers started"
        else
            log_error "\u2717 Failed to start the Docker containers"
        fi
    fi
    
    if [ "$START_OLLAMA" = true ] && [ "$START_DOCKER" = true ]; then
        if [ $OLLAMA_RESULT -eq 0 ] && [ $DOCKER_RESULT -eq 0 ]; then
            log_success "All services started. Available at:"
            printf "%b\n" "${GREEN}  - Frontend: http://localhost:${FRONTEND_PORT:-80}${NC}"
            printf "%b\n" "${GREEN}  - API: http://localhost:${APP_PORT:-8080}${NC}"
            echo ""
            log_info "Streaming container logs (press Ctrl+C to stop following; the containers keep running)..."
            "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD logs app docreader postgres --since=10s -f
        else
            log_error "Some services failed to start; check the logs and fix the problems"
        fi
    elif [ "$START_OLLAMA" = true ] && [ $OLLAMA_RESULT -eq 0 ]; then
        log_success "Ollama service started. Available at:"
        printf "%b\n" "${GREEN}  - Ollama API: http://localhost:$OLLAMA_PORT${NC}"
    elif [ "$START_DOCKER" = true ] && [ $DOCKER_RESULT -eq 0 ]; then
        log_success "Docker containers started. Available at:"
        printf "%b\n" "${GREEN}  - Frontend: http://localhost:${FRONTEND_PORT:-80}${NC}"
        printf "%b\n" "${GREEN}  - API: http://localhost:${APP_PORT:-8080}${NC}"
        echo ""
        log_info "Streaming container logs (press Ctrl+C to stop following; the containers keep running)..."
        "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD logs app docreader postgres --since=10s -f
    fi
fi

exit 0