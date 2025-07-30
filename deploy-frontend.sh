#!/bin/bash

# Deploy Frontend to Google Cloud Run
# Usage: ./deploy-frontend.sh [PROJECT_ID]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ID=${1:-"your-project-id"}
SERVICE_NAME="tru-activity-frontend"
REGION="asia-southeast1"

echo -e "${YELLOW}🚀 Deploying TRU Activity Frontend to Google Cloud Run${NC}"
echo -e "${YELLOW}Project ID: ${PROJECT_ID}${NC}"
echo -e "${YELLOW}Service: ${SERVICE_NAME}${NC}"
echo -e "${YELLOW}Region: ${REGION}${NC}"
echo ""

# Check if gcloud is installed
if ! command -v gcloud &> /dev/null; then
    echo -e "${RED}❌ gcloud CLI is not installed. Please install it first.${NC}"
    exit 1
fi

# Check if user is authenticated
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | head -n 1 > /dev/null; then
    echo -e "${RED}❌ Not authenticated with gcloud. Please run 'gcloud auth login' first.${NC}"
    exit 1
fi

# Set the project
echo -e "${YELLOW}📋 Setting project to ${PROJECT_ID}...${NC}"
gcloud config set project ${PROJECT_ID}

# Enable required APIs if not already enabled
echo -e "${YELLOW}🔧 Enabling required APIs...${NC}"
gcloud services enable cloudbuild.googleapis.com
gcloud services enable run.googleapis.com
gcloud services enable containerregistry.googleapis.com

# Navigate to frontend directory
cd frontend

# Submit build to Cloud Build
echo -e "${YELLOW}🏗️  Building and deploying frontend...${NC}"
gcloud builds submit --config cloudbuild.yaml

# Get the service URL
SERVICE_URL=$(gcloud run services describe ${SERVICE_NAME} --region=${REGION} --format="value(status.url)")

echo ""
echo -e "${GREEN}✅ Frontend deployment completed!${NC}"
echo -e "${GREEN}🌐 Service URL: ${SERVICE_URL}${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo -e "1. Update your backend CORS settings to allow: ${SERVICE_URL}"
echo -e "2. Update .env.production with your actual backend URL"
echo -e "3. Test the deployment to ensure everything works correctly"
echo ""
echo -e "${YELLOW}📝 To update environment variables:${NC}"
echo -e "gcloud run services update ${SERVICE_NAME} --region=${REGION} --set-env-vars PUBLIC_API_URL=https://your-backend-url"