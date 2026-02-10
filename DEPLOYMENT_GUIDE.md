# Loki MCP Server - Deployment Guide

This guide consolidates all deployment, configuration, and technical information for the Loki MCP Server modified for AWS Bedrock AgentCore.

---

## Table of Contents

1. [What Changed from Original](#what-changed-from-original)
2. [Why the Library Change Works](#why-the-library-change-works)
3. [AWS Bedrock AgentCore Deployment](#aws-bedrock-agentcore-deployment)
4. [Network Configuration](#network-configuration)
5. [Testing Guide](#testing-guide)
6. [Troubleshooting](#troubleshooting)

---

## What Changed from Original

This is a modified version of [loki-mcp](https://github.com/scottlepp/loki-mcp) adapted for AWS Bedrock AgentCore.

### Key Modifications

| Component | Original | Modified Version |
|-----------|----------|------------------|
| **Communication** | stdin/stdout | HTTP (port 8000, endpoint `/mcp`) |
| **MCP Library** | mark3labs/mcp-go | ThinkInAIXYZ/go-mcp v0.2.24 |
| **Session Management** | Server-managed with validation | Platform-managed (stateless) |
| **Transport Mode** | stdio | Stateless HTTP |
| **Use Case** | Claude Desktop | AWS Bedrock AgentCore |
| **Platform** | Any | ARM64 optimized |
| **Server Mode** | Subprocess | Standalone HTTP server |

### Dockerfile Changes

The original repository uses a simple Dockerfile for stdin/stdout communication. This modified version includes two Dockerfiles:

#### Original Dockerfile (stdin/stdout)
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o loki-mcp-server ./cmd/server

FROM alpine:latest
COPY --from=builder /app/loki-mcp-server .
ENTRYPOINT ["./loki-mcp-server"]
```
- **Platform:** Any (defaults to build machine architecture)
- **Communication:** stdin/stdout
- **Use Case:** Claude Desktop (subprocess spawning)

#### Modified Dockerfile (HTTP, standard)
```dockerfile
FROM --platform=linux/arm64 golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o loki-mcp-server ./cmd/server

FROM --platform=linux/arm64 alpine:latest
COPY --from=builder /app/loki-mcp-server .
EXPOSE 8000
ENTRYPOINT ["./loki-mcp-server"]
```
- **Platform:** linux/arm64 (explicit)
- **Communication:** HTTP on port 8000
- **Use Case:** AWS Bedrock AgentCore, local testing
- **Changes:** Added `--platform`, `EXPOSE 8000`, downloads dependencies during build

#### Modified Dockerfile.ecr (HTTP, optimized)
```dockerfile
FROM --platform=linux/arm64 golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -mod=vendor -o loki-mcp-server ./cmd/server

FROM --platform=linux/arm64 alpine:latest
COPY --from=builder /app/loki-mcp-server .
EXPOSE 8000
ENTRYPOINT ["./loki-mcp-server"]
```
- **Platform:** linux/arm64 (explicit)
- **Communication:** HTTP on port 8000
- **Use Case:** AWS Bedrock AgentCore production deployment
- **Changes:** Added `--platform`, `EXPOSE 8000`, uses vendored dependencies (faster, more reliable)

**Key Dockerfile Differences:**
1. **Platform specification** - Original doesn't specify, modified explicitly targets ARM64
2. **Port exposure** - Original doesn't expose ports (stdin/stdout), modified exposes 8000 (HTTP)
3. **Dependency management** - Dockerfile.ecr uses vendored dependencies for faster builds
4. **Architecture** - Original builds for any platform, modified specifically for ARM64 (AWS Graviton)

### Code Changes

#### Server Implementation (`cmd/server/main.go`)

**Original (stdin/stdout):**
```go
package main

import (
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
    "github.com/mark3labs/mcp-go/transport/stdio"
)

func main() {
    // Create stdio transport
    s := server.NewMCPServer(
        "Loki MCP Server",
        "0.1.0",
        server.WithTransport(stdio.NewStdioServerTransport()),
    )
    
    // Register tools
    s.AddTool(/* ... */)
    
    // Run server (blocks, communicates via stdin/stdout)
    s.Serve()
}
```

**Modified (HTTP):**
```go
package main

import (
    "net/http"
    "github.com/ThinkInAIXYZ/go-mcp/server"
    "github.com/ThinkInAIXYZ/go-mcp/transport"
)

func main() {
    // Create Streamable HTTP transport (Stateless mode)
    streamableTransport, mcpHandler, err := transport.NewStreamableHTTPServerTransportAndHandler(
        transport.WithStreamableHTTPServerTransportAndHandlerOptionStateMode(transport.Stateless),
    )
    
    // Initialize MCP server
    mcpServer, err := server.NewServer(streamableTransport, /* ... */)
    
    // Register tools
    mcpServer.RegisterTool(/* ... */)
    
    // Start MCP server
    go mcpServer.Run()
    
    // Create HTTP server
    mux := http.NewServeMux()
    mux.Handle("/mcp", mcpHandler.HandleMCP())
    
    // Listen on port 8000
    http.ListenAndServe(":8000", mux)
}
```

**Key Changes:**
- Changed from `mark3labs/mcp-go` to `ThinkInAIXYZ/go-mcp`
- Changed from stdio transport to Streamable HTTP transport
- Added HTTP server on port 8000 with `/mcp` endpoint
- Added Stateless mode for platform-managed sessions
- Added graceful shutdown handling
- Added environment variable configuration (HOST, PORT, LOKI_URL, etc.)

#### Client Implementation (`cmd/client/main.go`)

**Original (subprocess spawning):**
```go
package main

import (
    "os/exec"
    "github.com/mark3labs/mcp-go/client"
)

func main() {
    // Spawn server as subprocess
    cmd := exec.Command("./loki-mcp-server")
    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()
    cmd.Start()
    
    // Create client with stdio transport
    c := client.NewStdioMCPClient(stdin, stdout)
    
    // Call tools
    result := c.CallTool("loki_query", args)
}
```

**Modified (HTTP client):**
```go
package main

import (
    "net/http"
    "github.com/ThinkInAIXYZ/go-mcp/client"
    "github.com/ThinkInAIXYZ/go-mcp/transport"
)

func main() {
    // Create HTTP transport client
    transportClient, err := transport.NewStreamableHTTPClientTransport(
        "http://localhost:8000/mcp",
    )
    
    // Initialize MCP client
    mcpClient, err := client.NewClient(transportClient)
    defer mcpClient.Close()
    
    // Call tools
    result, err := mcpClient.CallTool(ctx, &protocol.CallToolRequest{
        Name: "loki_query",
        RawArguments: argsJSON,
    })
}
```

**Key Changes:**
- Changed from subprocess spawning to HTTP client
- Changed from `mark3labs/mcp-go` to `ThinkInAIXYZ/go-mcp`
- Added configuration via environment variables (MCP_SERVER_URL, LOKI_QUERY_TIMEOUT)
- Added command-line flag support (--server-url)
- Added proper error handling and timeout configuration
- Server runs independently, client connects via HTTP

#### Dependencies (`go.mod`)

**Original:**
```go
module github.com/scottlepp/loki-mcp

go 1.24.4

require (
    github.com/mark3labs/mcp-go v0.32.0
)
```

**Modified:**
```go
module github.com/scottlepp/loki-mcp

go 1.24.4

require (
    github.com/ThinkInAIXYZ/go-mcp v0.2.24
    github.com/mark3labs/mcp-go v0.32.0  // Kept for reference
)

require (
    github.com/google/uuid v1.6.0
    github.com/orcaman/concurrent-map/v2 v2.0.1
    github.com/spf13/cast v1.7.1
    github.com/tidwall/gjson v1.18.0
    // ... other dependencies
)
```

**Key Changes:**
- Added `ThinkInAIXYZ/go-mcp v0.2.24` (new MCP library)
- Kept `mark3labs/mcp-go` for backward compatibility (can be removed)
- Added new transitive dependencies for HTTP transport

#### New Files Added

**`internal/handlers/loki_protocol.go`** (NEW)
- Protocol-based tool handlers for MCP
- Implements `loki_query`, `loki_label_names`, `loki_label_values` tools
- Uses `ThinkInAIXYZ/go-mcp` protocol types
- Handles JSON argument parsing and validation

**`Dockerfile.ecr`** (NEW)
- ARM64-optimized Dockerfile for AWS deployment
- Uses vendored dependencies for faster builds
- Produces minimal image (~10-30 MB)

**`cmd/client/main_test.go`** (NEW)
- Property-based tests for client
- Tests configuration loading and validation
- Ensures correct precedence (flag > env > default)

**`test-mcp-server.sh`** (NEW)
- Automated testing script for HTTP-based MCP server
- Tests all three tools (query, label_names, label_values)
- Uses curl to send JSON-RPC requests

**`vendor/`** (NEW)
- Vendored dependencies for reliable builds
- Used by Dockerfile.ecr for offline builds
- Ensures consistent dependency versions

### Why These Changes?

AWS Bedrock AgentCore requires:
1. **HTTP-based communication** - No subprocess spawning
2. **Stateless transport** - Platform manages sessions
3. **Port 8000** - Standard AgentCore port
4. **Endpoint `/mcp`** - Standard AgentCore endpoint
5. **ARM64 support** - AgentCore runs on ARM64

---

## Why the Library Change Works

### The Problem with mark3labs/mcp-go

The original `mark3labs/mcp-go` library maintains a **server-side session store** and validates session IDs:

```go
// mark3labs/mcp-go validates sessions
func (s *Server) handleRequest(sessionID string) error {
    session, exists := s.sessions[sessionID]
    if !exists {
        return errors.New("invalid session ID")
    }
    // Process request...
}
```

**Issue:** AWS Bedrock AgentCore generates its own session IDs at the platform level. When these platform-generated IDs are sent to the server, `mark3labs/mcp-go` rejects them because they don't exist in its internal session store.

### The Solution with ThinkInAIXYZ/go-mcp

The `ThinkInAIXYZ/go-mcp` library operates in **stateless mode** and accepts any session ID:

```go
// ThinkInAIXYZ/go-mcp accepts any session ID
func (t *StatelessTransport) HandleRequest(sessionID string) error {
    // No session validation - just process the request
    return t.processRequest(sessionID)
}
```

**Result:** The server accepts Bedrock's platform-generated session IDs without validation, allowing seamless integration.

### Session Management Comparison

#### mark3labs/mcp-go (Server-Managed)
```
Client → initialize → Server creates session → Returns session ID
Client → request (session ID) → Server validates → Processes if valid
```

#### ThinkInAIXYZ/go-mcp (Platform-Managed)
```
Platform → generates session ID → Includes in all requests
Client → request (platform session ID) → Server accepts → Processes
```

### AWS Bedrock AgentCore Protocol Requirements

Bedrock AgentCore requires:
- **Stateless streamable-http transport** - No server-side session state
- **Platform-managed sessions** - Bedrock generates and manages session IDs
- **HTTP endpoint** - Port 8000, path `/mcp`
- **ARM64 architecture** - Runs on AWS Graviton processors

Only `ThinkInAIXYZ/go-mcp` supports these requirements out of the box.

---

## AWS Bedrock AgentCore Deployment

### Prerequisites

- AWS CLI configured
- Docker with ARM64 support
- Container registry (ECR recommended)
- Existing Loki instance

### Step 1: Build ARM64 Container

**Why Dockerfile.ecr?**

The project includes two Dockerfiles:
- **Dockerfile** - Standard x86_64 build for local testing
- **Dockerfile.ecr** - ARM64 build optimized for AWS Bedrock AgentCore

**Dockerfile.ecr features:**
- Targets ARM64 architecture (AWS Graviton processors)
- Uses vendored dependencies (faster, no network calls during build)
- Produces minimal image (~10-30 MB)
- Optimized for AWS ECR and Bedrock AgentCore

```bash
# Build ARM64-optimized image
docker build --platform linux/arm64 -f Dockerfile.ecr -t loki-mcp-server:latest .

# Verify image
docker images loki-mcp-server:latest
```

### Step 2: Push to Container Registry

```bash
# Tag for your registry
docker tag loki-mcp-server:latest YOUR_REGISTRY/loki-mcp-server:latest

# Push to registry
docker push YOUR_REGISTRY/loki-mcp-server:latest
```

### Step 3: Create Bedrock AgentCore Agent

```bash
aws bedrock-agentcore-control create-agent-runtime \
  --agent-runtime-name loki-query-agent \
  --agent-runtime-artifact '{
    "containerConfiguration": {
      "containerUri": "YOUR_REGISTRY/loki-mcp-server:latest"
    }
  }' \
  --protocol-configuration '{
    "serverProtocol": "MCP"
  }' \
  --environment-variables '{
    "LOKI_URL": "http://YOUR_LOKI_IP:3100"
  }' \
  --network-configuration '{
    "networkMode": "VPC",
    "networkModeConfig": {
      "securityGroups": ["sg-YOUR_BEDROCK_SG"],
      "subnets": ["subnet-YOUR_SUBNET_1", "subnet-YOUR_SUBNET_2"]
    }
  }' \
  --region us-east-1
```

### Step 4: Test the Agent

```bash
# Test with Bedrock AgentCore sandbox
# Send JSON-RPC request:
{
  "jsonrpc": "2.0",
  "id": "test-1",
  "method": "tools/call",
  "params": {
    "name": "loki_label_names",
    "arguments": {}
  }
}
```

---

## Network Configuration

For Bedrock AgentCore to communicate with your Loki instance, configure:

### 1. VPC Configuration

Both Bedrock AgentCore and Loki must be in the same VPC or have VPC peering.

```bash
# Verify VPC IDs match
aws ec2 describe-instances \
  --instance-ids YOUR_LOKI_INSTANCE_ID \
  --query 'Reservations[0].Instances[0].VpcId'
```

### 2. Security Group Rules

Add inbound rule to Loki security group:

```bash
# Allow Bedrock AgentCore to access Loki on port 3100
aws ec2 authorize-security-group-ingress \
  --group-id sg-LOKI_SG_ID \
  --protocol tcp \
  --port 3100 \
  --source-group sg-BEDROCK_SG_ID
```

### 3. Configure Loki URL

Set the Loki URL in your agent configuration:

```json
{
  "environmentVariables": {
    "LOKI_URL": "http://LOKI_PRIVATE_IP:3100"
  }
}
```

**Important:** Use the **private IP address**, not public IP.

### Network Architecture

```
┌─────────────────────────────────────────┐
│   AWS Bedrock AgentCore Agent           │
│   (MCP Server Container)                │
│                                         │
│   Security Group: sg-BEDROCK            │
│   Subnets: subnet-A, subnet-B           │
│   VPC: vpc-XXXXX                        │
│                                         │
│   Environment:                          │
│   LOKI_URL=http://10.x.x.x:3100         │
└──────────────┬──────────────────────────┘
               │
               │ HTTP Request
               │ Port 3100
               │
               ▼
┌─────────────────────────────────────────┐
│   Loki Server (EC2 or Container)        │
│                                         │
│   Security Group: sg-LOKI               │
│   Subnet: subnet-LOKI                   │
│   VPC: vpc-XXXXX (same VPC)             │
│   Private IP: 10.x.x.x                  │
│   Port: 3100                            │
└─────────────────────────────────────────┘
```

### Troubleshooting Network Issues

**Connection Refused:**
- Verify Loki is running: `curl http://localhost:3100/ready`
- Check security group rules allow port 3100
- Verify correct IP address and port

**Timeout:**
- Check security groups allow traffic
- Verify subnets are in same VPC
- Check route tables have local routes
- Verify Network ACLs don't block traffic

---

## Testing Guide

### Local Testing

#### 1. Start Local Environment

```bash
# Start Loki, Grafana, and log generator
docker-compose up -d

# Verify services are running
docker-compose ps
```

#### 2. Build and Run Server

```bash
# Build server
go build -o loki-mcp-server ./cmd/server

# Run server
./loki-mcp-server
```

#### 3. Test with Client

```bash
# Build client
go build -o loki-mcp-client ./cmd/client

# Test label names
./loki-mcp-client loki_label_names

# Test label values
./loki-mcp-client loki_label_values job

# Test query
./loki-mcp-client loki_query '{job="varlogs"}' "-5m" "now" 10
```

#### 4. Run Automated Tests

```bash
# Run test script
./test-mcp-server.sh
```

### JSON-RPC Testing

Test the MCP server directly with JSON-RPC requests:

#### Initialize Session

```bash
curl -X POST http://localhost:8000/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "capabilities": {},
      "clientInfo": {
        "name": "test-client",
        "version": "1.0.0"
      }
    }
  }'
```

#### List Tools

```bash
curl -X POST http://localhost:8000/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list",
    "params": {}
  }'
```

#### Call Tool

```bash
curl -X POST http://localhost:8000/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "loki_query",
      "arguments": {
        "query": "{job=\"varlogs\"}",
        "start": "-5m",
        "end": "now",
        "limit": 10
      }
    }
  }'
```

### Testing with Bedrock AgentCore

Use the Bedrock AgentCore sandbox to test:

```json
{
  "jsonrpc": "2.0",
  "id": "test-1",
  "method": "tools/call",
  "params": {
    "name": "loki_label_names",
    "arguments": {}
  }
}
```

Expected response:

```json
{
  "jsonrpc": "2.0",
  "id": "test-1",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "app\nenvironment\njob\nlevel"
      }
    ],
    "isError": false
  }
}
```

---

## Troubleshooting

### Common Issues

#### 1. Server Won't Start

**Error:** `bind: address already in use`

**Solution:**
```bash
# Find process using port 8000
lsof -i :8000

# Kill the process
kill -9 <PID>

# Or use a different port
PORT=8001 ./loki-mcp-server
```

#### 2. Client Can't Connect

**Error:** `connection refused`

**Solution:**
- Verify server is running: `curl http://localhost:8000/mcp`
- Check server logs for errors
- Verify correct URL: `http://localhost:8000/mcp`

#### 3. Loki Query Fails

**Error:** `dial tcp: connect: connection refused`

**Solution:**
- Verify Loki is running: `curl http://localhost:3100/ready`
- Check LOKI_URL environment variable
- Verify network connectivity

#### 4. Bedrock AgentCore Timeout

**Error:** `context deadline exceeded`

**Solution:**
- Check security group rules allow port 3100
- Verify Loki private IP is correct
- Check VPC configuration
- Verify subnets can communicate

#### 5. Invalid Session ID

**Error:** `invalid session ID`

**Solution:**
- This shouldn't happen with `ThinkInAIXYZ/go-mcp`
- If it does, verify you're using the correct library version
- Check that server is using stateless transport

### Debug Mode

Enable debug logging:

```bash
# Server
DEBUG=true ./loki-mcp-server

# Client
DEBUG=true ./loki-mcp-client loki_query '{job="varlogs"}'
```

### Verify Configuration

```bash
# Check server is listening
netstat -an | grep 8000

# Check Loki is accessible
curl http://localhost:3100/ready

# Check MCP endpoint
curl http://localhost:8000/mcp
```

---

## Environment Variables

### Server Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `HOST` | Server host | `0.0.0.0` |
| `PORT` | Server port | `8000` |

### Loki Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `LOKI_URL` | Loki server URL | `http://localhost:3100` |
| `LOKI_ORG_ID` | Organization ID for multi-tenancy | - |
| `LOKI_USERNAME` | Username for basic auth | - |
| `LOKI_PASSWORD` | Password for basic auth | - |
| `LOKI_TOKEN` | Bearer token for auth | - |

### Client Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_SERVER_URL` | MCP server URL | `http://localhost:8000/mcp` |
| `LOKI_QUERY_TIMEOUT` | Request timeout (seconds) | `30` |

---

## Summary

This deployment guide covers:

1. **What Changed** - Key modifications from the original repository
2. **Why It Works** - Technical explanation of the library change
3. **Deployment** - Step-by-step AWS Bedrock AgentCore deployment
4. **Network Setup** - VPC and security group configuration
5. **Testing** - Local and production testing procedures
6. **Troubleshooting** - Common issues and solutions

For basic usage and API reference, see [README.md](README.md).
