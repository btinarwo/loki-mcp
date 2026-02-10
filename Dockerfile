# Build stage - use ARM64 platform for AWS Bedrock AgentCore compatibility
FROM --platform=linux/arm64 golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application for ARM64 architecture (required by AWS Bedrock AgentCore)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o loki-mcp-server ./cmd/server

# Runtime stage - use ARM64 platform for AWS Bedrock AgentCore compatibility
FROM --platform=linux/arm64 alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/loki-mcp-server .

# Expose port for unified MCP server (both SSE and Streamable HTTP)
EXPOSE 8000

# Set the entry point
ENTRYPOINT ["./loki-mcp-server"]
