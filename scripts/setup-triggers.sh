#!/bin/bash

# Setup Cloud Build Triggers for GitHub Integration
# This script creates triggers that automatically build from GitHub without uploading files

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Load configuration
load_config() {
    if [ -f ".env.deploy" ]; then
        log_info "Loading configuration from .env.deploy"
        source .env.deploy
    else
        log_error ".env.deploy not found"
        exit 1
    fi
    
    export PROJECT_ID
    export REGION
    
    log_info "Using Project ID: $PROJECT_ID"
    log_info "Using Region: $REGION"
}

# Check if GitHub connection exists
check_github_connection() {
    log_info "Checking GitHub connection..."
    
    # List GitHub connections
    local connections=$(gcloud builds connections list --region=$REGION --format="value(name)" 2>/dev/null || echo "")
    
    if [ -z "$connections" ]; then
        log_warning "No GitHub connection found. Creating one..."
        create_github_connection
    else
        log_success "GitHub connection exists: $connections"
        GITHUB_CONNECTION=$(echo "$connections" | head -1)
    fi
}

# Create GitHub connection
create_github_connection() {
    log_info "Creating GitHub connection..."
    log_info "Follow the instructions to connect your GitHub account"
    
    gcloud builds connections create github "github-connection" \
        --region=$REGION
    
    GITHUB_CONNECTION="github-connection"
    log_success "GitHub connection created: $GITHUB_CONNECTION"
}

# Create repository link
create_repository_link() {
    log_info "Creating repository link..."
    
    # Get GitHub username and repo name
    read -p "Enter your GitHub username: " GITHUB_USERNAME
    read -p "Enter your repository name: " GITHUB_REPO
    
    local repo_link_name="tru-activity-repo"
    
    # Check if repo link exists
    local existing_link=$(gcloud builds repositories list \
        --connection=$GITHUB_CONNECTION \
        --region=$REGION \
        --filter="name:$repo_link_name" \
        --format="value(name)" 2>/dev/null || echo "")
    
    if [ -z "$existing_link" ]; then
        gcloud builds repositories create $repo_link_name \
            --connection=$GITHUB_CONNECTION \
            --region=$REGION \
            --remote-uri="https://github.com/$GITHUB_USERNAME/$GITHUB_REPO"
        
        log_success "Repository link created: $repo_link_name"
    else
        log_success "Repository link already exists: $repo_link_name"
    fi
    
    REPO_LINK="projects/$PROJECT_ID/locations/$REGION/connections/$GITHUB_CONNECTION/repositories/$repo_link_name"
}

# Create Cloud Build Triggers
create_triggers() {
    log_info "Creating Cloud Build Triggers..."
    
    # Main trigger for full deployment
    create_main_trigger
    
    # Frontend-only trigger
    create_frontend_trigger
    
    # Backend-only trigger  
    create_backend_trigger
}

# Create main deployment trigger
create_main_trigger() {
    log_info "Creating main deployment trigger..."
    
    cat > /tmp/main-trigger.yaml << EOF
name: tru-activity-main-deploy
description: "Full stack deployment trigger"
github:
  owner: $GITHUB_USERNAME
  name: $GITHUB_REPO
  push:
    branch: ^main$
filename: cloudbuild.yaml
substitutions:
  _REGION: $REGION
EOF

    gcloud builds triggers create github \
        --repo-name=$GITHUB_REPO \
        --repo-owner=$GITHUB_USERNAME \
        --branch-pattern="^main$" \
        --build-config=cloudbuild.yaml \
        --name="tru-activity-main-deploy" \
        --description="Full stack deployment on main branch" \
        --substitutions="_REGION=$REGION" || log_warning "Main trigger might already exist"
    
    log_success "Main deployment trigger created"
}

# Create frontend-only trigger
create_frontend_trigger() {
    log_info "Creating frontend-only trigger..."
    
    gcloud builds triggers create github \
        --repo-name=$GITHUB_REPO \
        --repo-owner=$GITHUB_USERNAME \
        --branch-pattern="^frontend-.*$" \
        --build-config=frontend/cloudbuild.yaml \
        --name="tru-activity-frontend-deploy" \
        --description="Frontend-only deployment" \
        --included-files="frontend/**" \
        --substitutions="_REGION=$REGION" || log_warning "Frontend trigger might already exist"
    
    log_success "Frontend-only trigger created"
}

# Create backend-only trigger
create_backend_trigger() {
    log_info "Creating backend-only trigger..."
    
    # Create backend-only cloudbuild.yaml
    cat > /tmp/backend-only-cloudbuild.yaml << 'EOF'
steps:
  # Build backend Docker image
  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'build'
      - '--no-cache'
      - '-t'
      - 'gcr.io/$PROJECT_ID/tru-activity-backend:$BUILD_ID'
      - '-t'
      - 'gcr.io/$PROJECT_ID/tru-activity-backend:latest'
      - './backend'
    id: 'build-backend'

  # Push backend image
  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'push'
      - 'gcr.io/$PROJECT_ID/tru-activity-backend:$BUILD_ID'
    id: 'push-backend'
    waitFor: ['build-backend']

  # Deploy backend
  - name: 'gcr.io/cloud-builders/gcloud'
    args:
      - 'run'
      - 'services'
      - 'replace'
      - 'backend/service.yaml'
      - '--region=$_REGION'
    id: 'deploy-backend'
    waitFor: ['push-backend']

  # Update backend image
  - name: 'gcr.io/cloud-builders/gcloud'
    entrypoint: 'bash'
    args:
      - '-c'
      - |
        gcloud run services update tru-activity-backend \
          --image=gcr.io/$PROJECT_ID/tru-activity-backend:$BUILD_ID \
          --region=$_REGION
    waitFor: ['deploy-backend']

substitutions:
  _REGION: 'asia-southeast1'

timeout: '1200s'
EOF

    # Copy to backend directory
    cp /tmp/backend-only-cloudbuild.yaml backend/cloudbuild-backend-only.yaml
    
    gcloud builds triggers create github \
        --repo-name=$GITHUB_REPO \
        --repo-owner=$GITHUB_USERNAME \
        --branch-pattern="^backend-.*$" \
        --build-config=backend/cloudbuild-backend-only.yaml \
        --name="tru-activity-backend-deploy" \
        --description="Backend-only deployment" \
        --included-files="backend/**" \
        --substitutions="_REGION=$REGION" || log_warning "Backend trigger might already exist"
    
    log_success "Backend-only trigger created"
}

# List created triggers
list_triggers() {
    log_info "Created Cloud Build Triggers:"
    gcloud builds triggers list --format="table(name,description,github.owner,github.name,github.push.branch)"
}

# Main function
main() {
    log_info "Setting up Cloud Build Triggers for faster deployment..."
    
    load_config
    check_github_connection
    create_repository_link
    create_triggers
    list_triggers
    
    echo ""
    log_success "🎉 Cloud Build Triggers setup completed!"
    log_info "Now you can deploy by pushing to GitHub branches:"
    echo ""
    log_info "📦 Full deployment: Push to 'main' branch"
    log_info "🎨 Frontend only: Push to 'frontend-*' branch"  
    log_info "⚙️  Backend only: Push to 'backend-*' branch"
    echo ""
    log_info "No more file uploads! Builds will pull directly from GitHub."
}

# Run main function
main "$@"