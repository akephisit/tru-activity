#!/bin/bash

# TRU Activity - Development Environment Setup
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

# Check if Docker is running
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi
    print_success "Docker is running"
}

# Check if Docker Compose is available
check_docker_compose() {
    if ! command -v docker-compose >/dev/null 2>&1 && ! docker compose version >/dev/null 2>&1; then
        print_error "Docker Compose not found. Please install Docker Compose."
        exit 1
    fi
    print_success "Docker Compose is available"
}

# Function to start development environment
start_dev() {
    print_status "Starting TRU Activity development environment..."
    
    check_docker
    check_docker_compose
    
    # Create .env files if they don't exist
    if [ ! -f backend/.env ]; then
        print_warning "Creating backend/.env from template..."
        cp backend/.env.example backend/.env 2>/dev/null || echo "# Backend environment variables will be loaded from docker-compose.dev.yml" > backend/.env
    fi
    
    if [ ! -f frontend/.env ]; then
        print_warning "Creating frontend/.env from template..."
        cp frontend/.env.example frontend/.env 2>/dev/null || echo "# Frontend environment variables will be loaded from docker-compose.dev.yml" > frontend/.env
    fi
    
    # Start services
    print_status "Building and starting services..."
    
    # Use docker compose or docker-compose based on availability
    if docker compose version >/dev/null 2>&1; then
        COMPOSE_CMD="docker compose"
    else
        COMPOSE_CMD="docker-compose"
    fi
    
    $COMPOSE_CMD -f docker-compose.dev.yml up --build -d
    
    print_success "Development environment started!"
    print_status "Services:"
    echo "  📱 Frontend: http://localhost:5173 (Vite dev server)"
    echo "  🔗 Backend API: http://localhost:8080"
    echo "  🎮 GraphQL Playground: http://localhost:8080/"
    echo "  🗄️  PostgreSQL: localhost:5432"
    echo "  📦 Redis: localhost:6379"
    echo ""
    print_status "Logs: $COMPOSE_CMD -f docker-compose.dev.yml logs -f"
    print_status "Stop: $COMPOSE_CMD -f docker-compose.dev.yml down"
}

# Function to stop development environment
stop_dev() {
    print_status "Stopping TRU Activity development environment..."
    
    if docker compose version >/dev/null 2>&1; then
        docker compose -f docker-compose.dev.yml down
    else
        docker-compose -f docker-compose.dev.yml down
    fi
    
    print_success "Development environment stopped!"
}

# Function to restart development environment
restart_dev() {
    print_status "Restarting TRU Activity development environment..."
    stop_dev
    start_dev
}

# Function to show logs
logs_dev() {
    if docker compose version >/dev/null 2>&1; then
        docker compose -f docker-compose.dev.yml logs -f
    else
        docker-compose -f docker-compose.dev.yml logs -f
    fi
}

# Function to reset database
reset_db() {
    print_warning "This will delete all development data. Are you sure? (y/N)"
    read -r response
    case "$response" in
        [yY][eE][sS]|[yY]) 
            print_status "Resetting database..."
            if docker compose version >/dev/null 2>&1; then
                docker compose -f docker-compose.dev.yml down -v
                docker compose -f docker-compose.dev.yml up postgres redis -d
            else
                docker-compose -f docker-compose.dev.yml down -v
                docker-compose -f docker-compose.dev.yml up postgres redis -d
            fi
            print_success "Database reset complete!"
            ;;
        *)
            print_status "Database reset cancelled."
            ;;
    esac
}

# Function to run backend only
backend_only() {
    print_status "Starting backend and dependencies only..."
    
    if docker compose version >/dev/null 2>&1; then
        docker compose -f docker-compose.dev.yml up postgres redis backend --build -d
    else
        docker-compose -f docker-compose.dev.yml up postgres redis backend --build -d
    fi
    
    print_success "Backend services started!"
    echo "  🔗 Backend API: http://localhost:8080"
    echo "  🎮 GraphQL Playground: http://localhost:8080/"
}

# Function to run frontend only (requires backend to be running)
frontend_only() {
    print_status "Starting frontend only..."
    
    if docker compose version >/dev/null 2>&1; then
        docker compose -f docker-compose.dev.yml up frontend --build -d
    else
        docker-compose -f docker-compose.dev.yml up frontend --build -d
    fi
    
    print_success "Frontend service started!"
    echo "  📱 Frontend: http://localhost:5173"
}

# Main script logic
case "${1:-start}" in
    start)
        start_dev
        ;;
    stop)
        stop_dev
        ;;
    restart)
        restart_dev
        ;;
    logs)
        logs_dev
        ;;
    reset-db)
        reset_db
        ;;
    backend)
        backend_only
        ;;
    frontend)
        frontend_only
        ;;
    status)
        print_status "Development environment status:"
        if docker compose version >/dev/null 2>&1; then
            docker compose -f docker-compose.dev.yml ps
        else
            docker-compose -f docker-compose.dev.yml ps
        fi
        ;;
    *)
        echo "TRU Activity Development Environment"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  start      Start all services (default)"
        echo "  stop       Stop all services"
        echo "  restart    Restart all services"
        echo "  logs       Show logs from all services"
        echo "  backend    Start backend and dependencies only"
        echo "  frontend   Start frontend only"
        echo "  status     Show service status"
        echo "  reset-db   Reset database (deletes all data)"
        echo ""
        echo "Examples:"
        echo "  $0                 # Start development environment"
        echo "  $0 backend         # Start only backend services"
        echo "  $0 logs            # View logs"
        echo "  $0 stop            # Stop everything"
        ;;
esac