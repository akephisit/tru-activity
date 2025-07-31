#!/bin/bash

# TRU Activity Deployment Script
# This script automates the deployment process to Google Cloud Platform

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ID=""
REGION="asia-southeast1"
TERRAFORM_DIR="infrastructure/terraform"
BACKEND_SERVICE_NAME="tru-activity-backend"
FRONTEND_PROJECT_ID=""

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

# Check if required tools are installed
check_dependencies() {
    log_info "Checking dependencies..."
    
    local deps=("gcloud" "terraform" "docker")
    local missing_deps=()
    
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            missing_deps+=("$dep")
        fi
    done
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing dependencies: ${missing_deps[*]}"
        log_info "Please install the missing dependencies and try again."
        exit 1
    fi
    
    log_success "All dependencies are installed"
}

# Load configuration from file or environment
load_config() {
    if [ -f ".env.deploy" ]; then
        log_info "Loading configuration from .env.deploy"
        source .env.deploy
    fi
    
    if [ -z "$PROJECT_ID" ]; then
        read -p "Enter your GCP Project ID: " PROJECT_ID
    fi
    
    if [ -z "$FRONTEND_PROJECT_ID" ]; then
        FRONTEND_PROJECT_ID=$PROJECT_ID
    fi
    
    log_info "Using Project ID: $PROJECT_ID"
    log_info "Using Region: $REGION"
}

# Authenticate with Google Cloud
auth_gcloud() {
    log_info "Checking Google Cloud authentication..."
    
    if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q "@"; then
        log_warning "Not authenticated with Google Cloud"
        gcloud auth login
    fi
    
    gcloud config set project "$PROJECT_ID"
    log_success "Google Cloud authentication verified"
}

# Initialize Terraform
init_terraform() {
    log_info "Initializing Terraform..."
    
    cd "$TERRAFORM_DIR"
    
    if [ ! -f "terraform.tfvars" ]; then
        log_warning "terraform.tfvars not found. Creating template..."
        cat > terraform.tfvars << EOF
project_id = "$PROJECT_ID"
region = "$REGION"

# Set these values before running terraform apply
db_password = "CHANGE_ME_SECURE_PASSWORD"
jwt_secret = "CHANGE_ME_JWT_SECRET"
email_from = "noreply@yourdomain.com"
sendgrid_api_key = "CHANGE_ME_SENDGRID_API_KEY"
qr_secret = "CHANGE_ME_QR_SECRET"
EOF
        log_warning "Please update terraform.tfvars with your actual values before continuing"
        return 1
    fi
    
    terraform init
    cd - > /dev/null
    log_success "Terraform initialized"
}

# Deploy infrastructure
deploy_infrastructure() {
    log_info "Deploying infrastructure with Terraform..."
    
    cd "$TERRAFORM_DIR"
    
    terraform plan -out=tfplan
    
    read -p "Do you want to apply these changes? (y/N): " confirm
    if [[ $confirm =~ ^[Yy]$ ]]; then
        terraform apply tfplan
        log_success "Infrastructure deployed successfully"
    else
        log_warning "Infrastructure deployment cancelled"
        cd - > /dev/null
        return 1
    fi
    
    cd - > /dev/null
}

# Enable required APIs
enable_apis() {
    log_info "Enabling required Google Cloud APIs..."
    
    local apis=(
        "run.googleapis.com"
        "sqladmin.googleapis.com"
        "redis.googleapis.com"
        "cloudbuild.googleapis.com"
        "secretmanager.googleapis.com"
        "monitoring.googleapis.com"
        "logging.googleapis.com"
        "cloudtrace.googleapis.com"
        "vpcaccess.googleapis.com"
        "servicenetworking.googleapis.com"
        "compute.googleapis.com"
    )
    
    for api in "${apis[@]}"; do
        gcloud services enable "$api" --project="$PROJECT_ID"
    done
    
    log_success "APIs enabled successfully"
}

# Build and deploy backend
deploy_backend() {
    log_info "Deploying backend with Cloud Build..."
    
    # Trigger Cloud Build
    gcloud builds submit \
        --config=cloudbuild.yaml \
        --substitutions=_REGION="$REGION" \
        --project="$PROJECT_ID" \
        .
    
    log_success "Backend deployed successfully"
}

# Deploy frontend to Cloud Run
deploy_frontend() {
    log_info "Deploying frontend to Cloud Run..."
    
    # Build and deploy using Cloud Build
    gcloud builds submit \
        --config=frontend/cloudbuild.yaml \
        --substitutions=_REGION="$REGION" \
        --project="$PROJECT_ID" \
        ./frontend
    
    log_success "Frontend deployed successfully"
}

# Run health checks
health_check() {
    log_info "Running health checks..."
    
    local backend_url="https://$BACKEND_SERVICE_NAME-$(echo $PROJECT_ID | sed 's/-//g')-$REGION.a.run.app"
    
    # Wait for service to be ready
    log_info "Waiting for backend service to be ready..."
    sleep 30
    
    # Check backend health
    if curl -f "$backend_url/health" > /dev/null 2>&1; then
        log_success "Backend health check passed"
    else
        log_error "Backend health check failed"
        return 1
    fi
    
    # Check database connectivity
    if curl -f "$backend_url/ready" > /dev/null 2>&1; then
        log_success "Database connectivity check passed"
    else
        log_error "Database connectivity check failed"
        return 1
    fi
    
    log_success "All health checks passed"
}

# Setup monitoring
setup_monitoring() {
    log_info "Setting up monitoring and alerting..."
    
    # Apply monitoring configuration
    cd "$TERRAFORM_DIR"
    terraform apply -target=google_monitoring_alert_policy.backend_down -auto-approve
    terraform apply -target=google_monitoring_alert_policy.high_error_rate -auto-approve
    terraform apply -target=google_monitoring_alert_policy.high_memory_usage -auto-approve
    terraform apply -target=google_monitoring_alert_policy.database_connections -auto-approve
    terraform apply -target=google_monitoring_dashboard.tru_activity_dashboard -auto-approve
    cd - > /dev/null
    
    log_success "Monitoring and alerting configured"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up temporary files..."
    rm -f "$TERRAFORM_DIR/tfplan"
    log_success "Cleanup completed"
}

# Main deployment function
main() {
    log_info "Starting TRU Activity deployment..."
    
    # Setup trap for cleanup
    trap cleanup EXIT
    
    check_dependencies
    load_config
    auth_gcloud
    enable_apis
    
    if ! init_terraform; then
        log_error "Please update terraform.tfvars and run the script again"
        exit 1
    fi
    
    deploy_infrastructure
    deploy_backend
    deploy_frontend
    
    sleep 10  # Give services time to start
    health_check
    setup_monitoring
    
    log_success "🎉 TRU Activity deployed successfully!"
    log_info "Backend URL: https://$BACKEND_SERVICE_NAME-$(echo $PROJECT_ID | sed 's/-//g')-$REGION.a.run.app"
    log_info "Frontend URL: https://tru-activity-frontend-$(echo $PROJECT_ID | sed 's/-//g')-$REGION.a.run.app"
    log_info "Monitoring Dashboard: https://console.cloud.google.com/monitoring/dashboards"
}

# Handle script arguments
case "${1:-deploy}" in
    "deploy")
        main
        ;;
    "infrastructure")
        check_dependencies
        load_config
        auth_gcloud
        enable_apis
        init_terraform
        deploy_infrastructure
        ;;
    "backend")
        check_dependencies
        load_config
        auth_gcloud
        deploy_backend
        ;;
    "frontend")
        check_dependencies
        load_config
        deploy_frontend
        ;;
    "health")
        check_dependencies
        load_config
        health_check
        ;;
    "monitoring")
        check_dependencies
        load_config
        auth_gcloud
        setup_monitoring
        ;;
    "cleanup")
        cleanup
        ;;
    *)
        echo "Usage: $0 [deploy|infrastructure|backend|frontend|health|monitoring|cleanup]"
        echo ""
        echo "Commands:"
        echo "  deploy        - Full deployment (default)"
        echo "  infrastructure - Deploy infrastructure only"
        echo "  backend       - Deploy backend only"
        echo "  frontend      - Deploy frontend only"
        echo "  health        - Run health checks only"
        echo "  monitoring    - Setup monitoring only"
        echo "  cleanup       - Cleanup temporary files"
        exit 1
        ;;
esac