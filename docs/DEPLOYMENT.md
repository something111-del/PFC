# Deployment Guide

## 1. Google Cloud Setup

 the following resources created:
- Project ID: `project_id`
- Artifact Registry: ``
- Secret Manager: `alpha-vantage-key`
- Service Account: `github-actions@project_id.iam.gserviceaccount.com`

## 2. GitHub Secrets

Add the following secrets to repository:
- `GCP_PROJECT_ID`: `project_id`
- `GCP_REGION`: `us-central1`
- `GCP_SA_KEY`: (JSON key content)

## 3. Automatic Deployment

Pushing to the `main` branch will trigger the GitHub Actions workflow `.github/workflows/deploy.yml`.

This workflow will:
1. Build Docker images for Go API and Python Model
2. Push images to Artifact Registry
3. Deploy services to Cloud Run
4. Link services (Go API calls Python Model)

## 4. Manual Deployment

You can manually deploy using the script:
```bash
./infrastructure/scripts/deploy-all.sh
```

## 5. Post-Deployment

After deployment, get the Go API URL:
```bash
gcloud run services describe pfc-go-api --region us-central1 --format 'value(status.url)'
```

Update `chrome-extension/popup/popup.js` with this URL:
```javascript
const API_URL = 'https://YOUR-GO-API-URL/v1';
```
