#!/bin/bash

# PFC Deployment Script
# Deploys Go API and Python Model to Cloud Run

set -e

PROJECT_ID="pfc-portfolio-forecast"
REGION="us-central1"
REPO_NAME="pfc-services"

echo "🚀 Starting PFC Deployment..."

# 1. Deploy Python Model
echo "🐍 Deploying Python Forecasting Service..."
gcloud builds submit ./python-model \
  --tag "$REGION-docker.pkg.dev/$PROJECT_ID/$REPO_NAME/python-model:latest"

gcloud run deploy pfc-python-model \
  --image "$REGION-docker.pkg.dev/$PROJECT_ID/$REPO_NAME/python-model:latest" \
  --platform managed \
  --region $REGION \
  --allow-unauthenticated \
  --memory 1Gi \
  --set-env-vars "NUM_SIMULATIONS=10000,FORECAST_HOURS=24"

PYTHON_URL=$(gcloud run services describe pfc-python-model --region $REGION --format 'value(status.url)')
echo "✅ Python Service deployed at: $PYTHON_URL"

# 2. Deploy Go API
echo "🐹 Deploying Go API Service..."
gcloud builds submit ./go-api \
  --tag "$REGION-docker.pkg.dev/$PROJECT_ID/$REPO_NAME/go-api:latest"

gcloud run deploy pfc-go-api \
  --image "$REGION-docker.pkg.dev/$PROJECT_ID/$REPO_NAME/go-api:latest" \
  --platform managed \
  --region $REGION \
  --allow-unauthenticated \
  --memory 512Mi \
  --set-env-vars "PYTHON_SERVICE_URL=$PYTHON_URL,FIRESTORE_PROJECT_ID=$PROJECT_ID" \
  --set-secrets "ALPHA_VANTAGE_KEY=alpha-vantage-key:latest"

GO_URL=$(gcloud run services describe pfc-go-api --region $REGION --format 'value(status.url)')
echo "✅ Go API deployed at: $GO_URL"

echo "🎉 Deployment Complete!"
echo "----------------------------------------"
echo "API URL: $GO_URL"
echo "----------------------------------------"
echo "👉 Update the API_URL in chrome-extension/popup/popup.js with this URL."
