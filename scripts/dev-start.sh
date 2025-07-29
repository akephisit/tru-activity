#!/bin/bash

# TRU Activity Development Start Script

echo "🚀 Starting TRU Activity Development Environment"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose is not installed. Please install it first."
    exit 1
fi

echo "📋 Checking environment files..."

# Check backend .env file
if [ ! -f "backend/.env" ]; then
    echo "⚠️  Backend .env file not found. Copying from .env.example..."
    cp backend/.env.example backend/.env
fi

echo "🐳 Starting Docker containers..."
docker-compose up -d

echo "⏳ Waiting for services to be ready..."
sleep 10

# Check if services are running
echo "🔍 Checking service health..."

# Check PostgreSQL
if docker-compose exec -T postgres pg_isready -U postgres > /dev/null 2>&1; then
    echo "✅ PostgreSQL is ready"
else
    echo "❌ PostgreSQL is not ready"
fi

# Check Redis
if docker-compose exec -T redis redis-cli ping > /dev/null 2>&1; then
    echo "✅ Redis is ready"
else
    echo "❌ Redis is not ready"
fi

# Check Backend
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "✅ Backend is ready"
else
    echo "⚠️  Backend might still be starting up..."
fi

echo ""
echo "🎉 Development environment is starting up!"
echo ""
echo "📱 Access points:"
echo "   Frontend:  http://localhost:5173"
echo "   Backend:   http://localhost:8080"
echo "   GraphQL:   http://localhost:8080/query"
echo "   Health:    http://localhost:8080/health"
echo ""
echo "🔧 Management commands:"
echo "   View logs:     docker-compose logs -f"
echo "   Stop all:      docker-compose down"
echo "   Restart:       docker-compose restart"
echo ""
echo "📊 Database access:"
echo "   PostgreSQL:    docker exec -it tru-activity-db psql -U postgres -d tru_activity"
echo "   Redis CLI:     docker exec -it tru-activity-redis redis-cli"
echo ""

# Show container status
echo "📋 Container Status:"
docker-compose ps