#!/bin/bash

# TRU Activity - Local Development (No Docker)
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if Go is installed
check_go() {
    if ! command -v go >/dev/null 2>&1; then
        print_error "Go is not installed. Please install Go 1.22+ first."
        echo "Visit: https://golang.org/dl/"
        return 1
    fi
    print_success "Go is installed ($(go version | awk '{print $3}'))"
}

# Check if Node.js is installed
check_node() {
    if ! command -v node >/dev/null 2>&1; then
        print_error "Node.js is not installed. Please install Node.js 20+ first."
        echo "Visit: https://nodejs.org/"
        return 1
    fi
    print_success "Node.js is installed ($(node --version))"
}

# Check if PostgreSQL is running
check_postgres() {
    if ! command -v psql >/dev/null 2>&1; then
        print_warning "PostgreSQL client not found. You'll need PostgreSQL for the database."
        print_status "Install PostgreSQL or use external database service"
        return 1
    fi
    
    # Check if PostgreSQL is running locally
    if pg_isready -h localhost -p 5432 >/dev/null 2>&1; then
        print_success "PostgreSQL is running on localhost:5432"
        return 0
    else
        print_warning "PostgreSQL is not running on localhost:5432"
        print_status "Start PostgreSQL or update .env with your database connection"
        return 1
    fi
}

# Check if Redis is running
check_redis() {
    if ! command -v redis-cli >/dev/null 2>&1; then
        print_warning "Redis client not found. You'll need Redis for caching."
        print_status "Install Redis or use external Redis service"
        return 1
    fi
    
    # Check if Redis is running locally
    if redis-cli -h localhost -p 6379 ping >/dev/null 2>&1; then
        print_success "Redis is running on localhost:6379"
        return 0
    else
        print_warning "Redis is not running on localhost:6379"
        print_status "Start Redis or update .env with your Redis connection"
        return 1
    fi
}

# Setup backend environment
setup_backend() {
    print_status "Setting up backend..."
    
    cd backend
    
    # Create .env if it doesn't exist
    if [ ! -f .env ]; then
        print_status "Creating backend/.env from template..."
        cp .env.example .env
        print_warning "Please update backend/.env with your database and Redis settings"
    fi
    
    # Install dependencies
    print_status "Installing Go dependencies..."
    go mod download
    
    # Install air for hot reloading if not installed
    if ! command -v air >/dev/null 2>&1; then
        print_status "Installing air for hot reloading..."
        go install github.com/air-verse/air@latest
    fi
    
    cd ..
    print_success "Backend setup completed"
}

# Setup frontend environment
setup_frontend() {
    print_status "Setting up frontend..."
    
    cd frontend
    
    # Create .env if it doesn't exist
    if [ ! -f .env ]; then
        print_status "Creating frontend/.env from template..."
        cp .env.example .env
    fi
    
    # Install dependencies
    print_status "Installing Node.js dependencies..."
    npm install
    
    cd ..
    print_success "Frontend setup completed"
}

# Start backend
start_backend() {
    print_status "Starting backend with hot reload..."
    
    cd backend
    
    # Check if air is available
    if command -v air >/dev/null 2>&1; then
        print_status "Starting backend with air (hot reload)..."
        air &
        BACKEND_PID=$!
        echo $BACKEND_PID > /tmp/tru-backend.pid
    else
        print_status "Starting backend with go run..."
        go run cmd/server/main.go &
        BACKEND_PID=$!
        echo $BACKEND_PID > /tmp/tru-backend.pid
    fi
    
    cd ..
    print_success "Backend started on http://localhost:8080 (PID: $BACKEND_PID)"
}

# Start frontend
start_frontend() {
    print_status "Starting frontend development server..."
    
    cd frontend
    npm run dev -- --host 0.0.0.0 --port 5173 &
    FRONTEND_PID=$!
    echo $FRONTEND_PID > /tmp/tru-frontend.pid
    cd ..
    
    print_success "Frontend started on http://localhost:5173 (PID: $FRONTEND_PID)"
}

# Stop services
stop_services() {
    print_status "Stopping TRU Activity services..."
    
    # Stop backend
    if [ -f /tmp/tru-backend.pid ]; then
        BACKEND_PID=$(cat /tmp/tru-backend.pid)
        if ps -p $BACKEND_PID > /dev/null; then
            kill $BACKEND_PID
            print_success "Backend stopped"
        fi
        rm -f /tmp/tru-backend.pid
    fi
    
    # Stop frontend
    if [ -f /tmp/tru-frontend.pid ]; then
        FRONTEND_PID=$(cat /tmp/tru-frontend.pid)
        if ps -p $FRONTEND_PID > /dev/null; then
            kill $FRONTEND_PID
            print_success "Frontend stopped"
        fi
        rm -f /tmp/tru-frontend.pid
    fi
    
    # Kill any remaining processes on ports
    print_status "Cleaning up processes on ports 8080 and 5173..."
    pkill -f "air" 2>/dev/null || true
    lsof -ti:8080 | xargs kill -9 2>/dev/null || true
    lsof -ti:5173 | xargs kill -9 2>/dev/null || true
    
    print_success "All services stopped"
}

# Check service status
check_status() {
    print_status "TRU Activity service status:"
    
    # Check backend
    if curl -s http://localhost:8080/health >/dev/null 2>&1; then
        print_success "Backend: Running (http://localhost:8080)"
    else
        print_warning "Backend: Not running"
    fi
    
    # Check frontend
    if curl -s http://localhost:5173 >/dev/null 2>&1; then
        print_success "Frontend: Running (http://localhost:5173)"
    else
        print_warning "Frontend: Not running"
    fi
    
    # Check processes
    if [ -f /tmp/tru-backend.pid ]; then
        BACKEND_PID=$(cat /tmp/tru-backend.pid)
        if ps -p $BACKEND_PID > /dev/null; then
            echo "  Backend PID: $BACKEND_PID"
        fi
    fi
    
    if [ -f /tmp/tru-frontend.pid ]; then
        FRONTEND_PID=$(cat /tmp/tru-frontend.pid)
        if ps -p $FRONTEND_PID > /dev/null; then
            echo "  Frontend PID: $FRONTEND_PID"
        fi
    fi
}

# Main script logic
case "${1:-start}" in
    setup)
        print_status "Setting up TRU Activity local development environment..."
        check_go || exit 1
        check_node || exit 1
        print_warning "Database and Redis checks (optional):"
        check_postgres || true
        check_redis || true
        setup_backend
        setup_frontend
        print_success "Setup completed! Run '$0 start' to start development servers"
        ;;
    start)
        print_status "Starting TRU Activity local development..."
        check_go || exit 1
        check_node || exit 1
        
        # Setup if not done
        if [ ! -f backend/.env ] || [ ! -d frontend/node_modules ]; then
            print_status "Running setup first..."
            setup_backend
            setup_frontend
        fi
        
        start_backend
        sleep 2
        start_frontend
        
        print_success "Development environment started!"
        echo ""
        print_status "Services:"
        echo "  📱 Frontend: http://localhost:5173"
        echo "  🔗 Backend API: http://localhost:8080"
        echo "  🎮 GraphQL Playground: http://localhost:8080/"
        echo ""
        print_status "Commands:"
        echo "  $0 stop       - Stop all services"
        echo "  $0 status     - Check service status"
        echo "  $0 logs       - Show logs"
        ;;
    stop)
        stop_services
        ;;
    status)
        check_status
        ;;
    logs)
        print_status "Showing recent logs..."
        echo "Backend logs:"
        tail -f backend/logs/*.log 2>/dev/null || echo "No backend logs found"
        ;;
    restart)
        stop_services
        sleep 2
        start_backend
        sleep 2
        start_frontend
        print_success "Services restarted!"
        ;;
    backend)
        print_status "Starting backend only..."
        check_go || exit 1
        if [ ! -f backend/.env ]; then
            setup_backend
        fi
        start_backend
        ;;
    frontend)
        print_status "Starting frontend only..."
        check_node || exit 1
        if [ ! -d frontend/node_modules ]; then
            setup_frontend
        fi
        start_frontend
        ;;
    *)
        echo "TRU Activity Local Development (No Docker)"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  setup      Setup development environment"
        echo "  start      Start all services (default)"
        echo "  stop       Stop all services"
        echo "  restart    Restart all services"
        echo "  backend    Start backend only"
        echo "  frontend   Start frontend only"
        echo "  status     Show service status"
        echo "  logs       Show logs"
        echo ""
        echo "Prerequisites:"
        echo "  - Go 1.22+ (https://golang.org/dl/)"
        echo "  - Node.js 20+ (https://nodejs.org/)"
        echo "  - PostgreSQL (optional, can use external DB)"
        echo "  - Redis (optional, can use external Redis)"
        echo ""
        echo "Examples:"
        echo "  $0 setup             # Setup environment"
        echo "  $0 start             # Start both services"
        echo "  $0 backend           # Start only backend"
        echo "  $0 stop              # Stop everything"
        ;;
esac