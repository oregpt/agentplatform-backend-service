# OreGPT Agent Platform Repository Guide

## Repository Structure

The OreGPT Agent Platform consists of three separate repositories, each in its own directory:

| Local Directory | GitHub Repository | Branch |
|----------------|-------------------|--------|
| `/Users/orphil/Documents/personal/Apps/AgentPlatform/auth-service` | https://github.com/oregpt/agentplatform-auth-service | PROD |
| `/Users/orphil/Documents/personal/Apps/AgentPlatform/backend-service` | https://github.com/oregpt/agentplatform-backend-service | PROD |
| `/Users/orphil/Documents/personal/Apps/AgentPlatform/frontend-service` | https://github.com/oregpt/agentplatform-frontend-service | PROD |

## Commit and Push Instructions

When making changes to the platform, remember to commit and push to each repository separately:

### Auth Service
```bash
cd /Users/orphil/Documents/personal/Apps/AgentPlatform/auth-service
git add .
git commit -m "Your commit message"
git push origin PROD
```

### Backend Service
```bash
cd /Users/orphil/Documents/personal/Apps/AgentPlatform/backend-service
git add .
git commit -m "Your commit message"
git push origin PROD
```

### Frontend Service
```bash
cd /Users/orphil/Documents/personal/Apps/AgentPlatform/frontend-service
git add .
git commit -m "Your commit message"
git push origin PROD
```

## Local Development Environment

### Auth Service (Port 8082)
```bash
cd /Users/orphil/Documents/personal/Apps/AgentPlatform/auth-service
PORT=8082 FIREBASE_PROJECT_ID=agentplatform-6cfd7 JWT_SECRET=changeinprodlater JWT_EXPIRATION_HOURS=24 go run cmd/main.go
```

### Backend Service (Port 8081)
```bash
cd /Users/orphil/Documents/personal/Apps/AgentPlatform/backend-service
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/your/service-account-key.json
PORT=8081 AUTH_SERVICE_URL=http://localhost:8082 GCP_PROJECT_ID=agentplatform-464918 SPANNER_INSTANCE=agentplatform SPANNER_DATABASE=agentplatform GCS_BUCKET=agentplatfrom go run cmd/main.go
```

### Frontend Service (Port 3000)
```bash
cd /Users/orphil/Documents/personal/Apps/AgentPlatform/frontend-service
npm run dev
```
