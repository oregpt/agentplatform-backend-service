# OreGPT Agent Platform GCP Deployment Guide

This guide provides instructions for deploying the OreGPT Agent Platform to Google Cloud Platform (GCP) using Docker containers.

## Prerequisites

1. Google Cloud SDK installed and configured
2. Docker installed locally
3. Access to GCP project with necessary permissions
4. Service account with appropriate permissions for Spanner and GCS

## Deployment Steps

### 1. Build and Push Docker Images

For each service, build and push the Docker image to Google Container Registry (GCR):

#### Auth Service

```bash
cd /Users/orphil/Documents/personal/Apps/AgentPlatform/auth-service
docker build -t gcr.io/agentplatform-464918/auth-service:latest .
docker push gcr.io/agentplatform-464918/auth-service:latest
```

#### Backend Service

```bash
cd /Users/orphil/Documents/personal/Apps/AgentPlatform/backend-service
docker build -t gcr.io/agentplatform-464918/backend-service:latest .
docker push gcr.io/agentplatform-464918/backend-service:latest
```

#### Frontend Service

```bash
cd /Users/orphil/Documents/personal/Apps/AgentPlatform/frontend-service
docker build -t gcr.io/agentplatform-464918/frontend-service:latest .
docker push gcr.io/agentplatform-464918/frontend-service:latest
```

### 2. Deploy to Cloud Run

#### Auth Service

```bash
gcloud run deploy auth-service \
  --image gcr.io/agentplatform-464918/auth-service:latest \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars="PORT=8082,FIREBASE_PROJECT_ID=agentplatform-6cfd7,JWT_SECRET=REPLACE_WITH_SECURE_SECRET,JWT_EXPIRATION_HOURS=24"
```

#### Backend Service

```bash
gcloud run deploy backend-service \
  --image gcr.io/agentplatform-464918/backend-service:latest \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars="PORT=8081,AUTH_SERVICE_URL=https://auth-service-HASH.a.run.app,GCP_PROJECT_ID=agentplatform-464918,SPANNER_INSTANCE=agentplatform,SPANNER_DATABASE=agentplatform,GCS_BUCKET=agentplatfrom" \
  --service-account=YOUR_SERVICE_ACCOUNT@agentplatform-464918.iam.gserviceaccount.com
```

Replace `https://auth-service-HASH.a.run.app` with the actual URL of your deployed auth service.

#### Frontend Service

```bash
gcloud run deploy frontend-service \
  --image gcr.io/agentplatform-464918/frontend-service:latest \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars="VITE_AUTH_API_URL=https://auth-service-HASH.a.run.app,VITE_BACKEND_API_URL=https://backend-service-HASH.a.run.app,VITE_FIREBASE_API_KEY=YOUR_API_KEY,VITE_FIREBASE_AUTH_DOMAIN=YOUR_AUTH_DOMAIN,VITE_FIREBASE_PROJECT_ID=agentplatform-6cfd7,VITE_FIREBASE_STORAGE_BUCKET=YOUR_STORAGE_BUCKET,VITE_FIREBASE_MESSAGING_SENDER_ID=YOUR_SENDER_ID,VITE_FIREBASE_APP_ID=YOUR_APP_ID,VITE_FIREBASE_MEASUREMENT_ID=YOUR_MEASUREMENT_ID"
```

Replace the placeholder URLs and Firebase configuration values with your actual values.

### 3. Set up Domain Mapping (Optional)

If you want to use custom domains:

```bash
gcloud beta run domain-mappings create --service auth-service --domain auth.yourdomain.com --region us-central1
gcloud beta run domain-mappings create --service backend-service --domain api.yourdomain.com --region us-central1
gcloud beta run domain-mappings create --service frontend-service --domain app.yourdomain.com --region us-central1
```

## Security Considerations

1. **JWT Secret**: Use a strong, randomly generated JWT secret in production.
2. **Service Account**: Use a service account with the minimum necessary permissions.
3. **Environment Variables**: Store sensitive environment variables in Secret Manager.

## Monitoring and Logging

1. Set up Cloud Monitoring for your Cloud Run services.
2. Configure log-based alerts for critical errors.
3. Set up uptime checks for each service.

## Scaling

Cloud Run automatically scales based on traffic. You can configure:

```bash
gcloud run services update auth-service --min-instances=1 --max-instances=10
gcloud run services update backend-service --min-instances=1 --max-instances=10
gcloud run services update frontend-service --min-instances=1 --max-instances=10
```
