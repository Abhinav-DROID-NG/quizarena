#!/bin/bash

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"
BACKEND_LOG="$SCRIPT_DIR/backend.log"
FRONTEND_LOG="$SCRIPT_DIR/frontend.log"
BACKEND_PID="$SCRIPT_DIR/backend.pid"
FRONTEND_PID="$SCRIPT_DIR/frontend.pid"

# Default ports
DB_PORT="${DB_PORT:-5432}"
BACKEND_PORT="${BACKEND_PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-5500}"

# Functions
print_header() {
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC} $1"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker not found. Please install Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi
    print_success "Docker found"
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose not found. Please install: https://docs.docker.com/compose/install/"
        exit 1
    fi
    print_success "Docker Compose found"
    
    # Check Go
    if ! command -v go &> /dev/null; then
        print_error "Go not found. Please install Go 1.25+: https://golang.org/dl/"
        exit 1
    fi
    GO_VERSION=$(go version | awk '{print $3}')
    print_success "Go found ($GO_VERSION)"
    
    # Check Python or Node for frontend server
    if ! command -v python3 &> /dev/null && ! command -v node &> /dev/null; then
        print_error "Python3 or Node.js not found. Please install one of them."
        exit 1
    fi
    
    if command -v python3 &> /dev/null; then
        print_success "Python3 found (for frontend server)"
        FRONTEND_SERVER="python3"
    else
        print_success "Node.js found (for frontend server)"
        FRONTEND_SERVER="node"
    fi
}

# Create .env file if not exists
create_env_file() {
    if [ ! -f "$ENV_FILE" ]; then
        print_info "Creating .env file..."
        cat > "$ENV_FILE" <<EOF
# QuizArena Configuration
PORT=8080
FRONTEND_ORIGIN=http://localhost:8080

# Database
DATABASE_URL=postgres://postgres:postgres@localhost:5432/quizarena?sslmode=disable
DB_MAX_CONNS=30

# JWT
JWT_SECRET=change-me-in-production-$(date +%s)
JWT_EXP_HOURS=24

# Google OAuth (Get from https://console.cloud.google.com/)
GOOGLE_CLIENT_ID=

# Shutdown
SHUTDOWN_TIMEOUT_SEC=10
EOF
        print_success "Created .env file"
        print_warning "Update GOOGLE_CLIENT_ID in .env file with your credentials"
    else
        print_info ".env already exists"
    fi
}

load_env_file() {
    if [ -f "$ENV_FILE" ]; then
        print_info "Loading environment from .env"
        set -a
        # shellcheck disable=SC1090
        source "$ENV_FILE"
        set +a
    fi
}

# Start database
start_database() {
    print_info "Starting PostgreSQL database..."
    
    # Check if container is already running
    if docker-compose ps -q postgres >/dev/null 2>&1 && [ -n "$(docker-compose ps -q postgres)" ] && docker-compose ps postgres | grep -q Up; then
        print_success "PostgreSQL already running"
        return 0
    fi
    
    # Check if container exists but stopped
    if docker-compose ps -q postgres >/dev/null 2>&1 && [ -n "$(docker-compose ps -q postgres)" ]; then
        print_info "Restarting existing PostgreSQL container..."
        docker-compose up -d postgres > /dev/null 2>&1
    else
        print_info "Creating new PostgreSQL container..."
        docker-compose up -d postgres
    fi
    
    # Wait for database to be ready
    print_info "Waiting for database to be ready..."
    for i in {1..30}; do
        if docker-compose exec -T postgres pg_isready -U postgres > /dev/null 2>&1; then
            print_success "PostgreSQL is ready"
            return 0
        fi
        echo -n "."
        sleep 1
    done
    
    print_error "Database failed to start"
    exit 1
}

# Stop database
stop_database() {
    print_info "Stopping PostgreSQL database..."
    docker-compose down > /dev/null 2>&1 || true
    print_success "Database stopped"
}

# Start backend
start_backend() {
    print_info "Starting backend server (port $BACKEND_PORT)..."
    
    if [ -f "$BACKEND_PID" ]; then
        OLD_PID=$(cat "$BACKEND_PID")
        if kill -0 "$OLD_PID" 2>/dev/null; then
            print_success "Backend already running (PID: $OLD_PID)"
            return 0
        fi
    fi
    
    cd "$SCRIPT_DIR"
    
    # Download dependencies
    print_info "Downloading Go dependencies..."
    go mod download > /dev/null 2>&1
    
    # Run backend in background
    PORT="${PORT:-$BACKEND_PORT}" go run ./... > "$BACKEND_LOG" 2>&1 &
    BACKEND_PID_NEW=$!
    echo $BACKEND_PID_NEW > "$BACKEND_PID"
    
    # Wait for backend to start
    print_info "Waiting for backend to be ready..."
    for i in {1..30}; do
        if curl -s http://localhost:$BACKEND_PORT/health > /dev/null 2>&1; then
            print_success "Backend is running (PID: $BACKEND_PID_NEW, Port: $BACKEND_PORT)"
            return 0
        fi
        echo -n "."
        sleep 1
    done
    
    print_error "Backend failed to start. Check logs: tail -f $BACKEND_LOG"
    exit 1
}

# Stop backend
stop_backend() {
    print_info "Stopping backend server..."
    if [ -f "$BACKEND_PID" ]; then
        PID=$(cat "$BACKEND_PID")
        if kill -0 "$PID" 2>/dev/null; then
            kill $PID || true
            rm -f "$BACKEND_PID"
        fi
    fi
    print_success "Backend stopped"
}

# Start frontend
start_frontend() {
    print_info "Frontend is served by the backend at http://localhost:$BACKEND_PORT"
    print_info "Skipping separate frontend server startup"
}

# Stop frontend
stop_frontend() {
    print_info "Stopping frontend server..."
    if [ -f "$FRONTEND_PID" ]; then
        PID=$(cat "$FRONTEND_PID")
        if kill -0 "$PID" 2>/dev/null; then
            kill $PID || true
            rm -f "$FRONTEND_PID"
        fi
    fi
    print_success "Frontend stopped"
}

# Show logs
show_logs() {
    case "$1" in
        backend)
            tail -f "$BACKEND_LOG"
            ;;
        frontend)
            tail -f "$FRONTEND_LOG"
            ;;
        database)
            docker-compose logs -f postgres
            ;;
        *)
            echo "Available logs:"
            echo "  bash setup.sh logs backend"
            echo "  bash setup.sh logs frontend"
            echo "  bash setup.sh logs database"
            ;;
    esac
}

# Show status
show_status() {
    print_header "Service Status"
    
    # Database
    if docker-compose ps -q postgres >/dev/null 2>&1 && [ -n "$(docker-compose ps -q postgres)" ] && docker-compose ps postgres | grep -q Up; then
        print_success "PostgreSQL: Running (port $DB_PORT)"
    else
        print_error "PostgreSQL: Stopped"
    fi
    
    # Backend
    if [ -f "$BACKEND_PID" ] && kill -0 $(cat "$BACKEND_PID") 2>/dev/null; then
        print_success "Backend: Running (port $BACKEND_PORT, PID: $(cat $BACKEND_PID))"
    else
        print_error "Backend: Stopped"
    fi
    
    # Frontend
    if [ -f "$FRONTEND_PID" ] && kill -0 $(cat "$FRONTEND_PID") 2>/dev/null; then
        print_success "Frontend: Running (port $FRONTEND_PORT, PID: $(cat $FRONTEND_PID))"
    else
        print_error "Frontend: Stopped"
    fi
}

# Cleanup
cleanup() {
    print_header "Cleanup"
    stop_backend
    stop_frontend
    stop_database
    
    read -p "Remove .env, logs, and PIDs? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -f "$ENV_FILE" "$BACKEND_LOG" "$FRONTEND_LOG" "$BACKEND_PID" "$FRONTEND_PID"
        print_success "Cleanup complete"
    fi
}

# Show help
show_help() {
    cat << 'EOF'
╔════════════════════════════════════════════════════════════════╗
║           QuizArena - Setup Script                            ║
╚════════════════════════════════════════════════════════════════╝

USAGE:
  bash setup.sh [COMMAND] [OPTIONS]

COMMANDS:
  (no command)  Start all services (default)
  start         Start all services
  stop          Stop all services
  restart       Restart all services
  status        Show service status
  logs          Show available logs
  logs [TYPE]   Show specific logs (backend, frontend, database)
  clean         Stop services and cleanup files
  help          Show this help message

ENVIRONMENT VARIABLES:
  DB_PORT          PostgreSQL port (default: 5432)
  BACKEND_PORT     Backend port (default: 8080)

EXAMPLES:
  bash setup.sh                          # Start everything
  bash setup.sh stop                     # Stop everything
  bash setup.sh logs backend             # View backend logs
  DB_PORT=5433 bash setup.sh             # Use custom DB port
  BACKEND_PORT=8081 bash setup.sh

URLS (after startup):
  App:       http://localhost:8080
  Backend:   http://localhost:8080
  Database:  localhost:5432

EOF
}

# Main
main() {
    case "${1:-start}" in
        start)
            check_prerequisites
            create_env_file
            load_env_file
            start_database
            start_backend
            start_frontend
            
            print_header "✓ QuizArena is ready!"
            echo ""
            echo -e "${GREEN}App:${NC}       http://localhost:$BACKEND_PORT"
            echo -e "${GREEN}Backend:${NC}   http://localhost:$BACKEND_PORT"
            echo -e "${GREEN}Database:${NC}  localhost:$DB_PORT"
            echo ""
            echo "Next steps:"
            echo "  1. Open http://localhost:$BACKEND_PORT in your browser"
            echo "  2. Update GOOGLE_CLIENT_ID in .env file"
            echo "     Get credentials: https://console.cloud.google.com/"
            echo "  3. Reload the page and click 'Login'"
            echo ""
            echo "Commands:"
            echo "  View logs:    bash setup.sh logs [backend|frontend|database]"
            echo "  Show status:  bash setup.sh status"
            echo "  Stop all:     bash setup.sh stop"
            echo ""
            ;;
        stop)
            stop_backend
            stop_frontend
            stop_database
            print_header "All services stopped"
            ;;
        restart)
            stop_backend
            stop_frontend
            stop_database
            sleep 2
            start_database
            start_backend
            start_frontend
            print_header "All services restarted"
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs "$2"
            ;;
        clean)
            cleanup
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo "Unknown command: $1"
            show_help
            exit 1
            ;;
    esac
}

# Run main
main "$@"
