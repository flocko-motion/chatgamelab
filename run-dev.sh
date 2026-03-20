#!/bin/bash
cd "$(dirname "$0")"

# Function to show usage
show_usage() {
    echo "Usage: $0 [MODE] [OPTIONS]"
    echo ""
    echo "Modes:"
    echo "  frontend    Develop frontend locally (starts db + backend in Docker)"
    echo "  backend     Develop backend locally (starts db + web in Docker)"
    echo "  all         Start only database"
    echo "  (no args)   Start all services in Docker (db + backend + web)"
    echo ""
    echo "Options:"
    echo "  --reset-db        Reset database before starting"
    echo "  --no-build        Skip Docker rebuild (use cached images)"
    echo "  --port-frontend   Override frontend port (default: 80)"
    echo "  --port-backend    Override backend port (default: 7001)"
    echo "  --port-db         Override database port (default: 7003)"
    echo ""
    echo "Examples:"
    echo "  $0 frontend                    # Develop frontend (run npm run dev locally)"
    echo "  $0 backend                     # Develop backend (run go run . locally)"
    echo "  $0 all                         # Start only database"
    echo "  $0                             # Run everything in Docker"
    echo "  $0 frontend --reset-db         # Reset DB and start frontend dev"
    echo "  $0 frontend --no-build          # Skip rebuild for faster startup"
    echo "  $0 frontend --port-backend 8080  # Use custom backend port"
}

# Function to check if a port is available, and offer to free it if not
check_port() {
    local port=$1
    local label=$2

    # Check if anything is listening on the port
    local pid_info
    pid_info=$(ss -tlnp "sport = :$port" 2>/dev/null | tail -n +2)
    if [ -z "$pid_info" ]; then
        return 0  # port is free
    fi

    echo -e "\033[1;31m❌ Port $port ($label) is already in use:\033[0m"
    echo ""

    # Show docker containers using this port
    local docker_match
    docker_match=$(docker ps --format '{{.ID}}\t{{.Names}}\t{{.Ports}}' 2>/dev/null | grep ":$port->" || true)
    if [ -n "$docker_match" ]; then
        echo -e "\033[1;33m  Docker container(s):\033[0m"
        echo "$docker_match" | while read -r line; do
            echo "    $line"
        done
    else
        echo -e "\033[1;33m  Process:\033[0m"
        echo "    $pid_info"
    fi
    echo ""

    read -r -p "  Stop the blocking process and continue? [y/N] " answer
    if [[ "$answer" =~ ^[Yy]$ ]]; then
        # Try stopping docker container first
        if [ -n "$docker_match" ]; then
            local container_id
            container_id=$(echo "$docker_match" | awk '{print $1}' | head -1)
            echo "  Stopping docker container $container_id..."
            docker stop "$container_id" >/dev/null 2>&1
        else
            # Kill the process by extracting PID from ss output
            local pid
            pid=$(echo "$pid_info" | grep -oP 'pid=\K[0-9]+' | head -1)
            if [ -n "$pid" ]; then
                echo "  Killing process $pid..."
                kill "$pid" 2>/dev/null || kill -9 "$pid" 2>/dev/null
            fi
        fi
        # Brief wait for port to free up
        sleep 1
        # Verify port is now free
        if ss -tlnp "sport = :$port" 2>/dev/null | tail -n +2 | grep -q .; then
            echo -e "\033[1;31m  Failed to free port $port. Please free it manually.\033[0m"
            exit 1
        fi
        echo -e "\033[1;32m  Port $port is now free.\033[0m"
        echo ""
    else
        echo "  Aborting."
        exit 1
    fi
}

# Function to check all required ports for the current mode
check_ports() {
    local mode=$1
    case $mode in
        frontend)
            check_port "$PORT_BACKEND" "backend"
            check_port "$PORT_POSTGRES" "database"
            ;;
        backend)
            check_port "$PORT_EXPOSED" "frontend"
            check_port "$PORT_POSTGRES" "database"
            ;;
        all)
            check_port "$PORT_POSTGRES" "database"
            ;;
        all_services)
            check_port "$PORT_EXPOSED" "frontend"
            check_port "$PORT_BACKEND" "backend"
            check_port "$PORT_POSTGRES" "database"
            ;;
    esac
}

# Function to reset database
reset_database() {
    echo "Resetting database..."
    echo "Stopping all containers first..."
    docker compose -f docker-compose.dev.yml down
    ./reset-dev-db.sh
    if [ $? -ne 0 ]; then
        echo "Database reset failed"
        exit 1
    fi
}

# Check if .env exists
if [ ! -f .env ]; then
    echo "❌ ERROR: .env file not found in root directory"
    echo ""
    echo "To fix this issue:"
    echo "  1. Copy the example file: cp .env.example .env"
    echo "  2. Edit .env with your configuration values"
    echo "  3. Run this script again"
    echo ""
    echo "The .env file is required for development environment configuration."
    exit 1
fi

# Source .env to get default port values
source .env

# Parse arguments
MODE=""
RESET_DB=false
BUILD_CONTAINERS=true
PORT_EXPOSED_OVERRIDE=""
PORT_BACKEND_OVERRIDE=""
PORT_POSTGRES_OVERRIDE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        frontend|backend|all)
            MODE="$1"
            shift
            ;;
        --reset-db)
            RESET_DB=true
            shift
            ;;
        --no-build)
            BUILD_CONTAINERS=false
            shift
            ;;
        --port-frontend)
            PORT_EXPOSED_OVERRIDE="$2"
            shift 2
            ;;
        --port-backend)
            PORT_BACKEND_OVERRIDE="$2"
            shift 2
            ;;
        --port-db)
            PORT_POSTGRES_OVERRIDE="$2"
            shift 2
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# If no mode specified, default to all services
if [ -z "$MODE" ]; then
    MODE="all_services"
fi

# Install npm dependencies if needed
if [ ! -d "web/node_modules" ]; then
    echo "📦 Installing frontend dependencies..."
    (cd web && npm install)
    echo ""
fi

# Reset database if requested
if [ "$RESET_DB" = true ]; then
    reset_database
fi

# Apply port overrides (use override if set, otherwise use .env value)
PORT_EXPOSED=${PORT_EXPOSED_OVERRIDE:-$PORT_EXPOSED}
PORT_BACKEND=${PORT_BACKEND_OVERRIDE:-$PORT_BACKEND}
PORT_POSTGRES=${PORT_POSTGRES_OVERRIDE:-$PORT_POSTGRES}

# Set build flag
if [ "$BUILD_CONTAINERS" = true ]; then
    BUILD_FLAG=" --build"
    echo -e "\033[1;33m🔨 Building containers (use --no-build to skip)\033[0m"
else
    BUILD_FLAG=""
    echo -e "\033[1;36m⚡ Using cached containers (faster startup)\033[0m"
fi
echo ""

# Check for port conflicts before starting
check_ports "$MODE"

# Start services based on mode
echo "Starting development environment in $MODE mode..."
echo ""

case $MODE in
    frontend)
        echo -e "\033[1;32m Frontend Development Mode\033[0m"
        echo -e "\033[1;33m Backend (Docker):  \033[1;34mhttp://localhost:${PORT_BACKEND}\033[0m"
        echo -e "\033[1;35m Database (Docker): \033[1;34mlocalhost:${PORT_POSTGRES}\033[0m"
        echo ""
        echo -e "\033[1;32m Frontend (Vite Dev): \033[1;34mhttp://localhost:${PORT_EXPOSED}\033[0m"
        echo ""
        docker compose -f docker-compose.dev.yml --profile frontend up${BUILD_FLAG}
        ;;
    backend)
        echo -e "\033[1;33m Backend Development Mode\033[0m"
        echo -e "\033[1;36m Frontend (Docker): \033[1;34mhttp://localhost:${PORT_EXPOSED}\033[0m"
        echo -e "\033[1;35m Database (Docker): \033[1;34mlocalhost:${PORT_POSTGRES}\033[0m"
        echo ""
        echo -e "\033[1;32m Now run the backend locally:\033[0m"
        echo -e "   \033[1;34mcd server && go run . server\033[0m"
        echo ""
        # Skip build for backend mode - running locally
        docker compose -f docker-compose.dev.yml --profile backend up
        ;;
    all)
        echo -e "\033[1;35m🗄️  Starting Database Only\033[0m"
        echo -e "\033[1;34m📍 Database:  localhost:${PORT_POSTGRES}\033[0m"
        echo ""
        docker compose -f docker-compose.dev.yml --profile db up${BUILD_FLAG}
        ;;
    all_services)
        echo -e "\033[1;34m All Services Mode\033[0m"
        echo -e "\033[1;36m Frontend (Docker): \033[1;34mhttp://localhost:${PORT_EXPOSED}\033[0m"
        echo -e "\033[1;33m Backend (Docker):  \033[1;34mhttp://localhost:${PORT_BACKEND}\033[0m"
        echo -e "\033[1;35m Database (Docker): \033[1;34mlocalhost:${PORT_POSTGRES}\033[0m"
        echo ""
        echo -e "\033[1;32m✨ All services running in Docker containers\033[0m"
        echo ""
        docker compose -f docker-compose.dev.yml --profile all_services up${BUILD_FLAG}
        ;;
esac
