# OreGPT Agent Platform - Backend Service

The Backend Service is a core component of the OreGPT Agent Platform, providing REST APIs for managing organizations, agents, files, and users. It integrates with Google Cloud Spanner for data storage and Google Cloud Storage for file management.

## Features

- Multi-tenant organization management
- Agent creation and configuration
- File upload/download for agent knowledge bases (markdown files)
- User management and agent assignment
- Integration with Auth Service for authentication and authorization

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| PORT | HTTP server port | 8081 |
| AUTH_SERVICE_URL | URL of the Auth Service | http://localhost:8080 |
| GCP_PROJECT_ID | Google Cloud Project ID | oregpt-agent-platform |
| SPANNER_INSTANCE | Spanner instance name | (required) |
| SPANNER_DATABASE | Spanner database name | (required) |
| GCS_BUCKET | Google Cloud Storage bucket name | (required) |

## API Endpoints

### Health Check

- `GET /api/v1/health` - Check if the service is running

### Organizations

- `GET /api/v1/organizations` - List all organizations
- `POST /api/v1/organizations` - Create a new organization
- `GET /api/v1/organizations/:id` - Get organization details
- `PUT /api/v1/organizations/:id` - Update an organization
- `DELETE /api/v1/organizations/:id` - Delete an organization

### Agents

- `GET /api/v1/agents` - List all agents
- `POST /api/v1/agents` - Create a new agent
- `GET /api/v1/agents/:id` - Get agent details
- `PUT /api/v1/agents/:id` - Update an agent
- `DELETE /api/v1/agents/:id` - Delete an agent

### Files

- `GET /api/v1/files/agent/:agent_id` - List all files for an agent
- `POST /api/v1/files/agent/:agent_id` - Upload a file for an agent
- `GET /api/v1/files/:id` - Download a file
- `DELETE /api/v1/files/:id` - Delete a file

### Users

- `GET /api/v1/users` - List all users
- `POST /api/v1/users` - Create a new user
- `GET /api/v1/users/:id` - Get user details
- `PUT /api/v1/users/:id` - Update a user
- `DELETE /api/v1/users/:id` - Delete a user
- `POST /api/v1/users/assign` - Assign a user to an agent
- `DELETE /api/v1/users/:user_id/agents/:agent_id` - Remove a user from an agent
- `GET /api/v1/users/:user_id/agents` - List all agents for a user

## Authentication

All API endpoints (except `/api/v1/health`) require authentication using a Firebase JWT token in the `Authorization` header:

```
Authorization: Bearer <token>
```

The token is validated by the Auth Service, which extracts user information and organization ID for multi-tenant authorization.

## Development

### Prerequisites

- Go 1.21 or later
- Access to Google Cloud Platform with Spanner and Storage services
- Firebase project for authentication

### Running Locally

1. Set up the required environment variables
2. Run the service:

```bash
go run cmd/main.go
```

### Building

```bash
go build -o backend-service ./cmd
```

## Database Schema

The service uses the following Spanner tables:

- `Organizations` - Organization data
- `Agents` - Agent configurations (interleaved in Organizations)
- `Files` - File metadata (interleaved in Agents)
- `Users` - User information
- `UserAgents` - User-agent assignments

## File Storage

Files are stored in Google Cloud Storage with the following path structure:
`<organization_id>/<agent_id>/<file_id>`

Only markdown (.md) files are currently supported for agent knowledge bases.
