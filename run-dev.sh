#!/bin/bash
cd "$(dirname "$0")"

# Function to show usage
show_usage() {
    echo "Usage: $0 [MODE] [OPTIONS]"
    echo ""
    echo "Modes:"
    echo "  frontend    Develop frontend locally (starts db + backend in Docker)"
    echo "  backend     Develop backend locally (starts db + web in Docker)"
    echo "  all         Start all services in Docker (db + backend + web)"
    echo "  db          Start only database"
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
    echo "  $0 all                         # Run everything in Docker"
    echo "  $0 db                          # Start only database"
    echo "  $0 frontend --reset-db         # Reset DB and start frontend dev"
    echo "  $0 frontend --no-build          # Skip rebuild for faster startup"
    echo "  $0 frontend --port-backend 8080  # Use custom backend port"
}

# Function to reset database
reset_database() {
    echo "Resetting database..."
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
        frontend|backend|all|db)
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

# If no mode specified, default to frontend
if [ -z "$MODE" ]; then
    MODE="frontend"
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

# Start services based on mode
echo "Starting development environment in $MODE mode..."
echo ""

case $MODE in
    frontend)
        echo -e "\033[1;32m🚀 Frontend Development Mode\033[0m"
        echo -e "\033[1;33m� Backend (Docker):  \033[1;34mhttp://localhost:${PORT_BACKEND}\033[0m"
        echo -e "\033[1;35m�️  Database (Docker): \033[1;34mlocalhost:${PORT_POSTGRES}\033[0m"
        echo ""
        echo -e "\033[1;32m� Now run the frontend locally:\033[0m"
        echo -e "   \033[1;34mcd web && npm run dev\033[0m"
        echo ""
        docker compose -f docker-compose.dev.yml --profile frontend up${BUILD_FLAG}
        ;;
    backend)
        echo -e "\033[1;33m🔧 Backend Development Mode\033[0m"
        echo -e "\033[1;36m📱 Frontend (Docker): \033[1;34mhttp://localhost:${PORT_EXPOSED}\033[0m"
        echo -e "\033[1;35m🗄️  Database (Docker): \033[1;34mlocalhost:${PORT_POSTGRES}\033[0m"
        echo ""
        echo -e "\033[1;32m💡 Now run the backend locally:\033[0m"
        echo -e "   \033[1;34mcd server && go run . server\033[0m"
        echo ""
        docker compose -f docker-compose.dev.yml --profile backend up${BUILD_FLAG}
        ;;
    all)
        echo -e "\033[1;34m🚀 All Services Mode\033[0m"
        echo -e "\033[1;36m📱 Frontend (Docker): \033[1;34mhttp://localhost:${PORT_EXPOSED}\033[0m"
        echo -e "\033[1;33m🔧 Backend (Docker):  \033[1;34mhttp://localhost:${PORT_BACKEND}\033[0m"
        echo -e "\033[1;35m🗄️  Database (Docker): \033[1;34mlocalhost:${PORT_POSTGRES}\033[0m"
        echo ""
        echo -e "\033[1;32m✨ All services running in Docker containers\033[0m"
        echo ""
        docker compose -f docker-compose.dev.yml --profile all up${BUILD_FLAG}
        ;;
    db)
        echo -e "\033[1;35m🗄️  Starting Database Only\033[0m"
        echo -e "\033[1;34m📍 Database:  localhost:${PORT_POSTGRES}\033[0m"
        echo ""
        docker compose -f docker-compose.dev.yml --profile db up${BUILD_FLAG}
        ;;
esac
