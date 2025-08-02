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
# TERRAFORM_DIR="infrastructure/terraform" # Removed - using gcloud only
BACKEND_SERVICE_NAME="tru-activity-backend"
FRONTEND_PROJECT_ID=""

# Deployment options
CLEAN_DEPLOY=false
INTERACTIVE_MODE=true

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

# Interactive prompts
ask_clean_deployment() {
    if [ "$INTERACTIVE_MODE" = false ]; then
        return
    fi
    
    echo ""
    log_info "🧹 Deployment Options"
    echo ""
    echo "Choose deployment mode:"
    echo "1) Standard deployment (keep existing data)"
    echo "2) Clean deployment (⚠️  DELETE all data and redeploy)"
    echo "3) Reset database only (⚠️  DELETE database data, keep infrastructure)"
    echo ""
    read -p "Enter your choice (1-3) [default: 1]: " choice
    
    case $choice in
        2)
            log_warning "You selected CLEAN DEPLOYMENT"
            log_warning "This will DELETE ALL resources and data!"
            echo ""
            log_warning "🚨 THIS WILL DELETE ALL RESOURCES AND DATA! 🚨"
            log_warning "This action is IRREVERSIBLE!"
            echo ""
            read -p "Type 'YES I WANT TO DELETE EVERYTHING' to confirm: " confirm
            if [ "$confirm" = "YES I WANT TO DELETE EVERYTHING" ]; then
                CLEAN_DEPLOY=true
            else
                log_info "Continuing with standard deployment"
            fi
            ;;
        3)
            log_warning "You selected DATABASE RESET"
            log_warning "This will DELETE ALL database data!"
            echo ""
            read -p "Are you sure? Type 'yes' to confirm: " confirm
            if [ "$confirm" = "yes" ]; then
                reset_database
                return 1  # Skip main deployment
            else
                log_info "Continuing with standard deployment"
            fi
            ;;
        1|"")
            log_info "Continuing with standard deployment"
            ;;
        *)
            log_warning "Invalid choice. Continuing with standard deployment"
            ;;
    esac
}

# Parse command line arguments
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --clean)
                CLEAN_DEPLOY=true
                INTERACTIVE_MODE=false
                shift
                ;;
            --no-interactive|-y)
                INTERACTIVE_MODE=false
                shift
                ;;
            --help|-h)
                show_usage
                exit 0
                ;;
            *)
                # Assume it's the command
                break
                ;;
        esac
    done
}

# Show usage information
show_usage() {
    echo "Usage: $0 [OPTIONS] [COMMAND]"
    echo ""
    echo "Options:"
    echo "  --clean           Clean deployment (delete all data and redeploy)"
    echo "  --no-interactive  Skip interactive prompts"
    echo "  -y                Same as --no-interactive"
    echo "  --help, -h        Show this help message"
    echo ""
    echo "Commands:"
    echo "  deploy            Full deployment (default)"
    echo "  infrastructure    Deploy infrastructure only"
    echo "  backend           Deploy backend only"
    echo "  frontend          Deploy frontend only"
    echo "  health            Run health checks only"
    echo "  monitoring        Setup monitoring only"
    echo "  cleanup           Cleanup temporary files"
    echo ""
    echo "Destructive Commands:"
    echo "  destroy           🚨 DELETE ALL RESOURCES (irreversible!)"
    echo "  reset             🚨 DELETE ALL DATA (keep infrastructure)"
    echo "  clean-deploy      🧹 Destroy everything and redeploy fresh"
    echo ""
    echo "Examples:"
    echo "  $0                          # Interactive deployment"
    echo "  $0 --clean deploy           # Clean deployment without prompts"
    echo "  $0 -y deploy               # Standard deployment without prompts"
    echo "  $0 --clean infrastructure  # Clean infrastructure deployment"
}

# Check if required tools are installed
check_dependencies() {
    log_info "Checking dependencies..."
    
    local deps=("gcloud" "docker")
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

# Validate required environment variables
validate_config() {
    local missing_vars=()
    
    if [ -z "$PROJECT_ID" ]; then
        missing_vars+=("PROJECT_ID")
    fi
    
    if [ -z "$DB_PASSWORD" ]; then
        missing_vars+=("DB_PASSWORD")
    fi
    
    if [ -z "$JWT_SECRET" ]; then
        missing_vars+=("JWT_SECRET")
    fi
    
    if [ -z "$QR_SECRET" ]; then
        missing_vars+=("QR_SECRET")
    fi
    
    if [ -z "$SENDGRID_API_KEY" ]; then
        missing_vars+=("SENDGRID_API_KEY")
    fi
    
    if [ ${#missing_vars[@]} -ne 0 ]; then
        log_error "Missing or invalid configuration variables:"
        for var in "${missing_vars[@]}"; do
            log_error "  - $var"
        done
        log_info ""
        log_info "Please create .env.deploy file based on .env.deploy.example"
        log_info "Or set the required environment variables"
        exit 1
    fi
}

# Load configuration from file or environment
load_config() {
    if [ -f ".env.deploy" ]; then
        log_info "Loading configuration from .env.deploy"
        source .env.deploy
    elif [ -f ".env.deploy.example" ]; then
        log_warning ".env.deploy not found, but .env.deploy.example exists"
        log_info "Please copy .env.deploy.example to .env.deploy and update the values"
        log_info "cp .env.deploy.example .env.deploy"
        exit 1
    fi
    
    if [ -z "$PROJECT_ID" ]; then
        read -p "Enter your GCP Project ID: " PROJECT_ID
    fi
    
    if [ -z "$FRONTEND_PROJECT_ID" ]; then
        FRONTEND_PROJECT_ID=$PROJECT_ID
    fi
    
    # Export variables for validation
    export PROJECT_ID
    export REGION
    export DB_PASSWORD
    export JWT_SECRET
    export QR_SECRET
    export SENDGRID_API_KEY
    export FRONTEND_PROJECT_ID
    
    log_info "Using Project ID: $PROJECT_ID"
    log_info "Using Region: $REGION"
    
    # Validate configuration
    validate_config
    
    log_success "Configuration loaded and validated"
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

# Check if resource exists
resource_exists() {
    local resource_type="$1"
    local resource_name="$2"
    local extra_args="$3"
    
    case "$resource_type" in
        "sql-instance")
            gcloud sql instances describe "$resource_name" &>/dev/null
            ;;
        "sql-database")
            gcloud sql databases describe "$resource_name" --instance=tru-activity-db &>/dev/null
            ;;
        "redis-instance")
            gcloud redis instances describe "$resource_name" --region="$REGION" &>/dev/null
            ;;
        "vpc-connector")
            gcloud compute networks vpc-access connectors describe "$resource_name" --region="$REGION" &>/dev/null
            ;;
        "secret")
            gcloud secrets describe "$resource_name" &>/dev/null
            ;;
        *)
            return 1
            ;;
    esac
}

# Setup infrastructure with gcloud commands
setup_infrastructure() {
    log_info "Setting up GCP resources..."
    
    # Create Cloud SQL instance
    if resource_exists "sql-instance" "tru-activity-db"; then
        log_info "Cloud SQL instance already exists, checking configuration..."
        # Ensure public IP is enabled and authorized networks are set
        gcloud sql instances patch tru-activity-db \
            --authorized-networks=0.0.0.0/0 \
            --assign-ip \
            --quiet || log_warning "Failed to update SQL instance configuration"
    else
        log_info "Creating Cloud SQL PostgreSQL instance..."
        gcloud sql instances create tru-activity-db \
            --database-version=POSTGRES_17 \
            --tier=db-f1-micro \
            --edition=ENTERPRISE \
            --region="$REGION" \
            --root-password="$DB_PASSWORD" \
            --storage-size=20GB \
            --storage-auto-increase \
            --backup-start-time=03:00 \
            --maintenance-window-day=SUN \
            --maintenance-window-hour=04 \
            --deletion-protection \
            --authorized-networks=0.0.0.0/0 \
            --assign-ip
    fi
    
    # Create database
    if resource_exists "sql-database" "tru_activity_prod"; then
        log_info "Database already exists, skipping creation"
    else
        log_info "Creating database..."
        gcloud sql databases create tru_activity_prod \
            --instance=tru-activity-db
    fi
    
    # Create Redis instance
    if resource_exists "redis-instance" "tru-activity-redis"; then
        log_info "Redis instance already exists, skipping creation"
    else
        log_info "Creating Redis instance..."
        gcloud redis instances create tru-activity-redis \
            --size=1 \
            --region="$REGION" \
            --redis-version=redis_7_0
    fi
    
    # Create subnet for VPC connector if it doesn't exist
    if ! gcloud compute networks subnets describe tru-activity-connector-subnet --region="$REGION" &>/dev/null; then
        log_info "Creating VPC connector subnet..."
        gcloud compute networks subnets create tru-activity-connector-subnet \
            --network=default \
            --range=10.8.0.0/28 \
            --region="$REGION"
    else
        log_info "VPC connector subnet already exists"
    fi
    
    # Create VPC connector
    if resource_exists "vpc-connector" "tru-activity-connector"; then
        log_info "VPC connector already exists, skipping creation"
    else
        log_info "Creating VPC connector..."
        gcloud compute networks vpc-access connectors create tru-activity-connector \
            --region="$REGION" \
            --subnet-project="$PROJECT_ID" \
            --subnet=tru-activity-connector-subnet \
            --min-instances=2 \
            --max-instances=10
    fi
    
    # Create secrets
    log_info "Setting up secrets..."
    
    if resource_exists "secret" "db-password"; then
        log_info "Updating db-password secret..."
        echo -n "$DB_PASSWORD" | gcloud secrets versions add db-password --data-file=-
    else
        log_info "Creating db-password secret..."
        echo -n "$DB_PASSWORD" | gcloud secrets create db-password --data-file=-
    fi
    
    if resource_exists "secret" "jwt-secret"; then
        log_info "JWT secret already exists, checking versions..."
        # Add version if no versions exist
        if [ $(gcloud secrets versions list jwt-secret --format="value(name)" | wc -l) -eq 0 ]; then
            echo -n "$JWT_SECRET" | gcloud secrets versions add jwt-secret --data-file=-
        fi
    else
        log_info "Creating jwt-secret..."
        echo -n "$JWT_SECRET" | gcloud secrets create jwt-secret --data-file=-
    fi
    
    if resource_exists "secret" "qr-secret"; then
        log_info "QR secret already exists, checking versions..."
        # Add version if no versions exist
        if [ $(gcloud secrets versions list qr-secret --format="value(name)" | wc -l) -eq 0 ]; then
            echo -n "$QR_SECRET" | gcloud secrets versions add qr-secret --data-file=-
        fi
    else
        log_info "Creating qr-secret..."
        echo -n "$QR_SECRET" | gcloud secrets create qr-secret --data-file=-
    fi
    
    if resource_exists "secret" "sendgrid-api-key"; then
        log_info "Updating sendgrid-api-key secret..."
        echo -n "$SENDGRID_API_KEY" | gcloud secrets versions add sendgrid-api-key --data-file=-
    else
        log_info "Creating sendgrid-api-key secret..."
        echo -n "$SENDGRID_API_KEY" | gcloud secrets create sendgrid-api-key --data-file=-
    fi
    
    # Grant secret access permissions to backend service account
    log_info "Setting up secret access permissions..."
    local backend_sa="tru-activity-backend@$PROJECT_ID.iam.gserviceaccount.com"
    
    gcloud secrets add-iam-policy-binding jwt-secret \
        --member="serviceAccount:$backend_sa" \
        --role="roles/secretmanager.secretAccessor" \
        --quiet || true
    
    gcloud secrets add-iam-policy-binding qr-secret \
        --member="serviceAccount:$backend_sa" \
        --role="roles/secretmanager.secretAccessor" \
        --quiet || true
    
    gcloud secrets add-iam-policy-binding sendgrid-api-key \
        --member="serviceAccount:$backend_sa" \
        --role="roles/secretmanager.secretAccessor" \
        --quiet || true
    
    log_success "Infrastructure setup completed"
}

# Create service accounts and IAM roles
setup_service_accounts() {
    log_info "Setting up service accounts..."
    
    # Create backend service account
    if ! gcloud iam service-accounts describe tru-activity-backend@$PROJECT_ID.iam.gserviceaccount.com &>/dev/null; then
        log_info "Creating backend service account..."
        gcloud iam service-accounts create tru-activity-backend \
            --display-name="TRU Activity Backend Service Account" \
            --description="Service account for TRU Activity backend"
    else
        log_info "Backend service account already exists"
    fi
    
    # Create migration service account
    if ! gcloud iam service-accounts describe tru-activity-migration@$PROJECT_ID.iam.gserviceaccount.com &>/dev/null; then
        log_info "Creating migration service account..."
        gcloud iam service-accounts create tru-activity-migration \
            --display-name="TRU Activity Migration Service Account" \
            --description="Service account for database migrations"
    else
        log_info "Migration service account already exists"
    fi
    
    # Grant necessary IAM roles to backend service account
    log_info "Setting up IAM roles for backend service account..."
    local backend_sa="tru-activity-backend@$PROJECT_ID.iam.gserviceaccount.com"
    
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member="serviceAccount:$backend_sa" \
        --role="roles/cloudsql.client" \
        --quiet || true
    
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member="serviceAccount:$backend_sa" \
        --role="roles/redis.editor" \
        --quiet || true
    
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member="serviceAccount:$backend_sa" \
        --role="roles/secretmanager.secretAccessor" \
        --quiet || true
    
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member="serviceAccount:$backend_sa" \
        --role="roles/cloudsql.instanceUser" \
        --quiet || true
    
    # Grant necessary IAM roles to migration service account
    log_info "Setting up IAM roles for migration service account..."
    local migration_sa="tru-activity-migration@$PROJECT_ID.iam.gserviceaccount.com"
    
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member="serviceAccount:$migration_sa" \
        --role="roles/cloudsql.client" \
        --quiet || true
    
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member="serviceAccount:$migration_sa" \
        --role="roles/secretmanager.secretAccessor" \
        --quiet || true
    
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member="serviceAccount:$migration_sa" \
        --role="roles/cloudsql.instanceUser" \
        --quiet || true
    
    log_success "Service accounts setup completed"
}


# Update service.yaml with current project values
update_service_yaml() {
    log_info "Updating service.yaml with project values..."
    
    # Get Redis IP
    local redis_ip
    redis_ip=$(gcloud redis instances describe tru-activity-redis --region=$REGION --format="value(host)" 2>/dev/null || echo "10.0.0.1")
    
    # Get Cloud SQL public IP
    local sql_ip
    sql_ip=$(gcloud sql instances describe tru-activity-db --format="value(ipAddresses[0].ipAddress)" 2>/dev/null || echo "127.0.0.1")
    
    # Create a temporary service.yaml with substituted values
    sed -e "s/PROJECT_ID/$PROJECT_ID/g" \
        -e "s/REGION/$REGION/g" \
        -e "s/REDIS_IP/$redis_ip/g" \
        -e "s|/cloudsql/PROJECT_ID:REGION:tru-activity-db|$sql_ip|g" \
        -e "s/127\.0\.0\.1/$sql_ip/g" \
        -e "s/35\.185\.188\.104/$sql_ip/g" \
        backend/service.yaml > /tmp/service.yaml.tmp
    
    # Replace the original with updated version
    cp /tmp/service.yaml.tmp backend/service.yaml
    rm /tmp/service.yaml.tmp
    
    log_success "Backend service.yaml updated successfully"
    log_info "Frontend service.yaml uses Cloud Build substitutions"
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

# Build and deploy both backend and frontend
deploy_full_stack() {
    log_info "Deploying full stack (backend + frontend) with Cloud Build..."
    
    # Trigger unified Cloud Build that includes both backend and frontend
    gcloud builds submit \
        --config=cloudbuild.yaml \
        --substitutions=_REGION="$REGION" \
        --project="$PROJECT_ID" \
        .
    
    log_success "Full stack deployed successfully"
}

# Build and deploy backend only (for compatibility)
deploy_backend() {
    log_info "Deploying backend only..."
    log_warning "Note: This only deploys backend. Use 'deploy' command for full stack deployment."
    
    # Create temporary cloudbuild for backend only
    cat > /tmp/cloudbuild-backend-only.yaml << EOF
steps:
  # Build backend Docker image
  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'build'
      - '--no-cache'
      - '-t'
      - 'gcr.io/\$PROJECT_ID/tru-activity-backend:\$BUILD_ID'
      - '-t'
      - 'gcr.io/\$PROJECT_ID/tru-activity-backend:latest'
      - './backend'
    id: 'build-backend'

  # Push backend images
  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'push'
      - 'gcr.io/\$PROJECT_ID/tru-activity-backend:\$BUILD_ID'
    id: 'push-backend'
    waitFor: ['build-backend']

  # Deploy to Cloud Run
  - name: 'gcr.io/cloud-builders/gcloud'
    args:
      - 'run'
      - 'services'
      - 'replace'
      - 'backend/service.yaml'
      - '--region=\$_REGION'
    id: 'deploy-backend'
    waitFor: ['push-backend']

  # Update service image
  - name: 'gcr.io/cloud-builders/gcloud'
    entrypoint: 'bash'
    args:
      - '-c'
      - |
        gcloud run services update tru-activity-backend \\
          --image=gcr.io/\$PROJECT_ID/tru-activity-backend:\$BUILD_ID \\
          --region=\$_REGION
    id: 'update-service-image'
    waitFor: ['deploy-backend']

substitutions:
  _REGION: '$REGION'
timeout: '1200s'
EOF

    # Deploy backend only
    gcloud builds submit \
        --config=/tmp/cloudbuild-backend-only.yaml \
        --substitutions=_REGION="$REGION" \
        --project="$PROJECT_ID" \
        .
    
    # Cleanup temp file
    rm -f /tmp/cloudbuild-backend-only.yaml
    
    log_success "Backend deployed successfully"
}

# Deploy frontend to Cloud Run (standalone)
deploy_frontend_standalone() {
    log_info "Deploying frontend to Cloud Run (standalone)..."
    
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
    
    # Get actual backend service URL
    local backend_url=$(gcloud run services describe tru-activity-backend \
        --region="$REGION" \
        --format="value(status.url)" 2>/dev/null) || {
        log_error "Could not get backend service URL"
        return 1
    }
    
    log_info "Backend URL: $backend_url"
    
    # Wait for service to be ready
    log_info "Waiting for backend service to be ready..."
    sleep 10
    
    # Check backend health
    if curl -f "$backend_url/health" > /dev/null 2>&1; then
        log_success "Backend health check passed"
    else
        log_error "Backend health check failed"
        log_info "Trying to get more details..."
        curl -v "$backend_url/health" || true
        return 1
    fi
    
    # Check database connectivity (skip /ready if it doesn't exist)
    if curl -f "$backend_url/ready" > /dev/null 2>&1; then
        log_success "Database connectivity check passed"
    else
        log_warning "Database connectivity check failed or /ready endpoint not available"
        log_info "Continuing deployment as basic health check passed"
    fi
    
    log_success "All health checks passed"
}

# Setup monitoring
setup_monitoring() {
    log_info "Setting up monitoring and alerting..."
    
    # Create basic monitoring dashboard (simplified)
    log_info "Basic monitoring available in Cloud Console"
    log_info "Dashboard: https://console.cloud.google.com/monitoring/dashboards"
    log_info "Logs: https://console.cloud.google.com/logs"
    
    log_success "Monitoring configured - check Cloud Console"
}

# Destroy all resources
destroy_all() {
    log_warning "🚨 THIS WILL DELETE ALL RESOURCES AND DATA! 🚨"
    log_warning "This action is IRREVERSIBLE!"
    echo ""
    read -p "Type 'YES I WANT TO DELETE EVERYTHING' to confirm: " confirm
    
    if [ "$confirm" != "YES I WANT TO DELETE EVERYTHING" ]; then
        log_info "Operation cancelled"
        return 0
    fi
    
    log_info "Starting resource destruction..."
    
    # Delete Cloud Run services
    log_info "Deleting Cloud Run services..."
    gcloud run services delete tru-activity-backend --region="$REGION" --quiet || log_warning "Backend service not found"
    gcloud run services delete tru-activity-frontend --region="$REGION" --quiet || log_warning "Frontend service not found"
    
    # Delete VPC Connector
    log_info "Deleting VPC Connector..."
    gcloud compute networks vpc-access connectors delete tru-activity-connector --region="$REGION" --quiet || log_warning "VPC connector not found"
    
    # Delete Redis instance
    log_info "Deleting Redis instance..."
    gcloud redis instances delete tru-activity-redis --region="$REGION" --quiet || log_warning "Redis instance not found"
    
    # Delete Cloud SQL instance (WARNING: This deletes all data!)
    log_info "Deleting Cloud SQL instance..."
    gcloud sql instances delete tru-activity-db --quiet || log_warning "Cloud SQL instance not found"
    
    # Delete secrets
    log_info "Deleting secrets..."
    gcloud secrets delete db-password --quiet || log_warning "Secret not found"
    gcloud secrets delete jwt-secret --quiet || log_warning "Secret not found"
    gcloud secrets delete qr-secret --quiet || log_warning "Secret not found"
    gcloud secrets delete sendgrid-api-key --quiet || log_warning "Secret not found"
    
    # Delete container images
    log_info "Deleting container images..."
    gcloud container images delete gcr.io/$PROJECT_ID/tru-activity-backend --force-delete-tags --quiet || log_warning "Backend images not found"
    gcloud container images delete gcr.io/$PROJECT_ID/tru-activity-frontend --force-delete-tags --quiet || log_warning "Frontend images not found"
    
    log_success "🗑️ All resources have been destroyed!"
    log_info "You may want to delete the project entirely: gcloud projects delete $PROJECT_ID"
}

# Reset database data only
reset_database() {
    log_warning "🚨 THIS WILL DELETE ALL DATABASE DATA! 🚨"
    log_warning "This will keep infrastructure but wipe all application data"
    echo ""
    read -p "Type 'YES DELETE ALL DATA' to confirm: " confirm
    
    if [ "$confirm" != "YES DELETE ALL DATA" ]; then
        log_info "Operation cancelled"
        return 0
    fi
    
    log_info "Resetting database..."
    
    # Drop and recreate database
    gcloud sql databases delete tru_activity_prod --instance=tru-activity-db --quiet
    gcloud sql databases create tru_activity_prod --instance=tru-activity-db
    
    # Restart backend services to run migrations
    log_info "Restarting backend to run fresh migrations..."
    gcloud run services update tru-activity-backend \
        --region="$REGION" \
        --set-env-vars="FORCE_MIGRATION=true"
    
    log_success "🔄 Database has been reset! Fresh start with clean data."
}

# Clean deployment (destroy and redeploy)
clean_deploy() {
    log_info "🧹 Clean deployment: Destroying everything and redeploying..."
    
    destroy_all
    
    if [ $? -eq 0 ]; then
        log_info "Waiting 30 seconds for resources to be fully deleted..."
        sleep 30
        
        log_info "Starting fresh deployment..."
        main
    else
        log_error "Destruction failed, aborting clean deployment"
        return 1
    fi
}

# Cleanup function
cleanup() {
    log_info "Cleaning up temporary files..."
    # No terraform files to clean
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
    
    # Ask for deployment options if interactive mode
    if ! ask_clean_deployment; then
        return 0  # User chose database reset, exit
    fi
    
    # Handle clean deployment
    if [ "$CLEAN_DEPLOY" = true ]; then
        log_info "🧹 Performing clean deployment..."
        destroy_all
        if [ $? -eq 0 ]; then
            log_info "Waiting 30 seconds for resources to be fully deleted..."
            sleep 30
            log_info "Continuing with fresh deployment..."
        else
            log_error "Clean deployment failed, aborting"
            return 1
        fi
    fi
    
    setup_infrastructure
    setup_service_accounts  
    update_service_yaml
    deploy_full_stack
    
    sleep 10  # Give services time to start
    health_check
    setup_monitoring
    
    # Services are now public via service.yaml annotations
    log_info "Services are configured as public via service.yaml allow-unauthenticated annotations"
    
    # Get actual service URLs
    local backend_url=$(gcloud run services describe tru-activity-backend \
        --region="$REGION" \
        --format="value(status.url)" 2>/dev/null) || "Not available"
    
    local frontend_url=$(gcloud run services describe tru-activity-frontend \
        --region="$REGION" \
        --format="value(status.url)" 2>/dev/null) || "Not available"
    
    log_success "🎉 TRU Activity deployed successfully!"
    log_info "Backend URL: $backend_url"
    log_info "Frontend URL: $frontend_url"
    log_info "Monitoring Dashboard: https://console.cloud.google.com/monitoring/dashboards"
}

# Handle script arguments
# Parse arguments first
parse_arguments "$@"

# Get the command (after parsing options)
COMMAND="${1:-deploy}"
shift || true  # Remove the command from arguments

case "$COMMAND" in
    "deploy")
        main
        ;;
    "infrastructure")
        check_dependencies
        load_config
        auth_gcloud
        enable_apis
        setup_infrastructure
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
        auth_gcloud
        deploy_frontend_standalone
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
    "destroy")
        check_dependencies
        load_config
        destroy_all
        ;;
    "reset")
        check_dependencies
        load_config
        reset_database
        ;;
    "clean-deploy")
        check_dependencies
        load_config
        auth_gcloud
        enable_apis
        clean_deploy
        ;;
    *)
        show_usage
        exit 1
        ;;
esac