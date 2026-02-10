# Loki MCP Server (AWS Bedrock AgentCore Edition)

[![CI](https://github.com/scottlepp/loki-mcp/workflows/CI/badge.svg)](https://github.com/scottlepp/loki-mcp/actions/workflows/ci.yml)

A Go-based server implementation for the Model Context Protocol (MCP) with Grafana Loki integration, modified for AWS Bedrock AgentCore deployment.

## ⚠️ Important Notice

This is a **modified version** of the original [loki-mcp](https://github.com/scottlepp/loki-mcp) repository. Key differences:

- **Original:** Uses stdin/stdout communication with `mark3labs/mcp-go` library
- **This Version:** Uses HTTP-based communication with `ThinkInAIXYZ/go-mcp` library for AWS Bedrock AgentCore compatibility

**📄 See [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md) for complete deployment, configuration, and technical details.**

### Quick Comparison

| Feature | Original | This Version |
|---------|----------|--------------|
| Communication | stdin/stdout | HTTP (port 8000) |
| MCP Library | mark3labs/mcp-go | ThinkInAIXYZ/go-mcp |
| Endpoint | N/A | `/mcp` |
| Session Management | Server-managed | Platform-managed (stateless) |
| Claude Desktop | ✅ Supported | ❌ Not supported |
| AWS Bedrock AgentCore | ❌ Not supported | ✅ Fully compliant |
| Platform | Any | ARM64 optimized |

## Getting Started

### Prerequisites

- Go 1.16 or higher
- Docker (optional, for containerized deployment)

### Building and Running

**Note:** This version runs as an HTTP server on port 8000, not via stdin/stdout.

Build and run the server:

```bash
# Build the server
go build -o loki-mcp-server ./cmd/server

# Run the server (HTTP mode on port 8000)
./loki-mcp-server
```

Or run directly with Go:

```bash
go run ./cmd/server
```

The server will start an HTTP server on port 8000 (configurable via `PORT` environment variable) and expose the MCP endpoint at `/mcp`. This makes it suitable for use with AWS Bedrock AgentCore and other HTTP-based MCP clients.

**Default Configuration:**
- Host: `0.0.0.0` (all interfaces, configurable via `HOST` env var)
- Port: `8000` (configurable via `PORT` env var)
- Endpoint: `/mcp`
- Transport: Stateless HTTP

## Project Structure

```
.
├── cmd/
│   ├── server/       # MCP server implementation
│   └── client/       # Client for testing the MCP server
├── internal/
│   ├── handlers/     # Tool handlers
│   └── models/       # Data models
├── pkg/
│   └── utils/        # Utility functions and shared code
└── go.mod            # Go module definition
```

## MCP Server

The Loki MCP Server implements the Model Context Protocol (MCP) and provides the following tools:

### Loki Query Tool

The `loki_query` tool allows you to query Grafana Loki log data:

- Required parameters:
  - `query`: LogQL query string

- Optional parameters:
  - `url`: The Loki server URL (default: from LOKI_URL environment variable or http://localhost:3100)
  - `start`: Start time for the query (default: 1h ago)
  - `end`: End time for the query (default: now)
  - `limit`: Maximum number of entries to return (default: 100)
  - `org`: Organization ID for the query (sent as X-Scope-OrgID header)

#### Environment Variables

The Loki query tool supports the following environment variables:

- `LOKI_URL`: Default Loki server URL to use if not specified in the request
- `LOKI_ORG_ID`: Default organization ID to use if not specified in the request
- `LOKI_USERNAME`: Default username for basic authentication if not specified in the request
- `LOKI_PASSWORD`: Default password for basic authentication if not specified in the request
- `LOKI_TOKEN`: Default bearer token for authentication if not specified in the request

**Security Note**: When using authentication environment variables, be careful not to expose sensitive credentials in logs or configuration files. Consider using token-based authentication over username/password when possible.

### Testing the MCP Server

You can test the MCP server using the provided HTTP-based client. The client connects to a running MCP server via HTTP instead of spawning it as a subprocess.

#### Starting the Server

First, start the MCP server separately:

```bash
# Build and run the server
go build -o loki-mcp-server ./cmd/server
./loki-mcp-server

# Or run directly with Go
go run ./cmd/server

# Or use Docker
docker-compose up -d
```

The server will start on port 8000 by default and expose the MCP endpoint at `http://localhost:8000/mcp`.

#### Using the Client

Build and run the client to query the running server:

```bash
# Build the client
go build -o loki-mcp-client ./cmd/client

# Basic Loki query examples (connects to default http://localhost:8000/mcp):
./loki-mcp-client loki_query "{job=\"varlogs\"}"
./loki-mcp-client loki_query "{job=\"varlogs\"}" "-1h" "now" 100

# Using a custom server URL via environment variable:
export MCP_SERVER_URL="http://localhost:8000/mcp"
./loki-mcp-client loki_query "{job=\"varlogs\"}"

# Using a custom server URL via command-line flag (overrides environment variable):
./loki-mcp-client --server-url http://localhost:8000/mcp loki_query "{job=\"varlogs\"}"

# Configuring request timeout (default: 30 seconds):
export LOKI_QUERY_TIMEOUT=60  # Set timeout to 60 seconds
./loki-mcp-client loki_query "{job=\"varlogs\"}"

# Using environment variables for Loki configuration:
export LOKI_URL="http://localhost:3100"
./loki-mcp-client loki_query "{job=\"varlogs\"}"

# Using environment variables for both URL and org:
export LOKI_URL="http://localhost:3100"
export LOKI_ORG_ID="tenant-123"
./loki-mcp-client loki_query "{job=\"varlogs\"}"

# Using environment variables for authentication:
export LOKI_URL="http://localhost:3100"
export LOKI_USERNAME="admin"
export LOKI_PASSWORD="password"
./loki-mcp-client loki_query "{job=\"varlogs\"}"

# Using environment variables with bearer token:
export LOKI_URL="http://localhost:3100"
export LOKI_TOKEN="your-bearer-token"
./loki-mcp-client loki_query "{job=\"varlogs\"}"

# Using all environment variables together:
export MCP_SERVER_URL="http://localhost:8000/mcp"
export LOKI_QUERY_TIMEOUT=60
export LOKI_URL="http://localhost:3100"
export LOKI_ORG_ID="tenant-123"
export LOKI_USERNAME="admin"
export LOKI_PASSWORD="password"
./loki-mcp-client loki_query "{job=\"varlogs\"}"

# Using org parameter for multi-tenant setups:
./loki-mcp-client loki_query "{job=\"varlogs\"}" "" "" "" "" "" "tenant-123"
```

#### Client Configuration

The client supports the following configuration options:

- **MCP_SERVER_URL**: Environment variable to set the MCP server URL (default: `http://localhost:8000/mcp`)
- **--server-url**: Command-line flag to set the MCP server URL (overrides environment variable)
- **LOKI_QUERY_TIMEOUT**: Environment variable to set the HTTP request timeout in seconds (default: 30)

**Configuration Priority** (highest to lowest):
1. Command-line flag `--server-url`
2. Environment variable `MCP_SERVER_URL`
3. Default value `http://localhost:8000/mcp`

#### Error Handling

The client provides clear error messages for common issues:

- **Server not running**: "Failed to connect to server at {url}: connection refused"
- **Request timeout**: "Request timed out after {timeout} seconds"
- **HTTP errors**: "Server returned error: {status_code} {status_text}"
- **Invalid JSON**: "Failed to parse server response: invalid JSON"
- **JSON-RPC errors**: "Error from server: {error_message}"

## Docker Support

The project includes two Dockerfiles for different use cases:

### Dockerfile (Standard)
- **Platform:** linux/amd64 (x86_64)
- **Use Case:** Local testing, development
- **Build:** Downloads dependencies during build

```bash
# Build standard image
docker build -t loki-mcp-server .

# Run the server
docker run -p 8000:8000 --rm loki-mcp-server
```

### Dockerfile.ecr (AWS Optimized)
- **Platform:** linux/arm64 (ARM64)
- **Use Case:** AWS Bedrock AgentCore deployment
- **Build:** Uses vendored dependencies (faster, more reliable)
- **Size:** Minimal (~10-30 MB)
- **Architecture:** Optimized for AWS Graviton processors

```bash
# Build ARM64 image
docker build --platform linux/arm64 -f Dockerfile.ecr -t loki-mcp-server:latest .

# Run locally (if on ARM64 machine)
docker run -p 8000:8000 loki-mcp-server:latest
```

### Docker Compose

For local testing with a complete Loki environment:

```bash
# Build and run with Docker Compose
docker-compose up --build
```

### Local Testing with Loki

The project includes a complete Docker Compose setup to test Loki queries locally:

1. Start the Docker Compose environment:
   ```bash
   docker-compose up -d
   ```

   This will start:
   - A Loki server on port 3100
   - A Grafana instance on port 3000 (pre-configured with Loki as a data source)
   - A log generator container that sends sample logs to Loki
   - The Loki MCP server

2. Use the provided test script to query logs:
   ```bash
   # Run with default parameters (queries last 15 minutes of logs)
   ./test-loki-query.sh
   
   # Query for error logs
   ./test-loki-query.sh '{job="varlogs"} |= "ERROR"'
   
   # Specify a custom time range and limit
   ./test-loki-query.sh '{job="varlogs"}' '-1h' 'now' 50
   ```

3. Insert dummy logs for testing:
   ```bash
   # Insert 10 dummy logs with default settings
   ./insert-loki-logs.sh
   
   # Insert 20 logs with custom job and app name
   ./insert-loki-logs.sh --num 20 --job "custom-job" --app "my-app"
   
   # Insert logs with custom environment and interval
   ./insert-loki-logs.sh --env "production" --interval 0.5
   
   # Show help message
   ./insert-loki-logs.sh --help
   ```

4. Access the Grafana UI at http://localhost:3000 to explore logs visually.

## ⚠️ Claude Desktop Compatibility

**This modified version is NOT compatible with Claude Desktop** because it uses HTTP-based communication instead of stdin/stdout.

If you need Claude Desktop support, please use the [original repository](https://github.com/scottlepp/loki-mcp) which uses stdin/stdout communication.

### Why Not Compatible?

Claude Desktop expects MCP servers to:
1. Communicate via stdin/stdout
2. Be spawned as subprocesses
3. Use the `mark3labs/mcp-go` library

This modified version:
1. Communicates via HTTP on port 8000
2. Runs as a standalone HTTP server
3. Uses the `ThinkInAIXYZ/go-mcp` library for Bedrock AgentCore compatibility

## AWS Bedrock AgentCore Deployment

This version is specifically designed for AWS Bedrock AgentCore. To deploy:

### 1. Build and Push Container

```bash
# Build ARM64 image
docker build --platform linux/arm64 -f Dockerfile.ecr -t loki-mcp-server:latest .

# Tag and push to your container registry
docker tag loki-mcp-server:latest your-registry/loki-mcp-server:latest
docker push your-registry/loki-mcp-server:latest
```

### 2. Create Bedrock AgentCore Agent

```bash
aws bedrock-agentcore create-agent \
    --agent-name loki-query-agent \
    --runtime-config '{
      "type": "MCP_SERVER",
      "containerImage": "your-registry/loki-mcp-server:latest",
      "port": 8000,
      "endpointPath": "/mcp",
      "environmentVariables": {
        "LOKI_URL": "http://your-loki-server:3100"
      }
    }' \
    --region us-east-1
```

### 3. Test the Agent

```bash
aws bedrock-agentcore invoke-agent \
    --agent-id YOUR_AGENT_ID \
    --session-id test-$(date +%s) \
    --input-text "Show me the last 10 logs from the production job" \
    --region us-east-1
```

## Architecture

The Loki MCP Server uses a modular architecture:

- **Server**: The main MCP server implementation in `cmd/server/main.go`
  - HTTP server on port 8000
  - Stateless transport mode
  - Platform-managed sessions
- **Client**: A test client in `cmd/client/main.go` for interacting with the MCP server via HTTP
- **Handlers**: Individual tool handlers in `internal/handlers/`
  - `loki.go`: Grafana Loki query utilities
  - `loki_protocol.go`: Protocol-based tool handlers for MCP

## Key Technical Details

### Session Management

This implementation uses **platform-managed sessions** (stateless mode):
- The server accepts any session ID without validation
- AWS Bedrock AgentCore manages sessions at the platform level
- No server-side session store required

### Transport

- **Protocol**: HTTP with JSON-RPC 2.0
- **Endpoint**: `/mcp`
- **Port**: 8000 (configurable via `PORT` env var)
- **Host**: 0.0.0.0 (configurable via `HOST` env var)
- **Mode**: Stateless

### MCP Library

Uses `github.com/ThinkInAIXYZ/go-mcp v0.2.24` which provides:
- Stateless transport support
- Platform-managed session compatibility
- Bedrock AgentCore compliance

## Documentation

- **[DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md)** - Complete deployment, configuration, network setup, testing, and troubleshooting guide

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Running Tests

The project includes comprehensive unit tests and CI/CD workflows to ensure reliability:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Run tests with race detection  
go test -race ./...
```
