package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/server"
	"github.com/ThinkInAIXYZ/go-mcp/transport"

	"github.com/scottlepp/loki-mcp/internal/handlers"
)

const (
	version = "0.1.0"
)

func main() {
	log.Println("=== Loki MCP Server Starting ===")
	log.Printf("Version: %s", version)

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
		log.Println("PORT environment variable not set, using default: 8000")
	} else {
		log.Printf("PORT environment variable set to: %s", port)
	}

	// Get host from environment variable or use default (0.0.0.0 to listen on all interfaces)
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
		log.Println("HOST environment variable not set, using default: 0.0.0.0 (all interfaces)")
	} else {
		log.Printf("HOST environment variable set to: %s", host)
	}

	// Log Loki configuration
	log.Println("Checking Loki configuration...")
	if lokiURL := os.Getenv("LOKI_URL"); lokiURL != "" {
		log.Printf("  - LOKI_URL: %s", lokiURL)
	} else {
		log.Println("  - LOKI_URL: not set (will use default or per-request URL)")
	}
	if lokiOrgID := os.Getenv("LOKI_ORG_ID"); lokiOrgID != "" {
		log.Printf("  - LOKI_ORG_ID: %s", lokiOrgID)
	} else {
		log.Println("  - LOKI_ORG_ID: not set")
	}
	if lokiUsername := os.Getenv("LOKI_USERNAME"); lokiUsername != "" {
		log.Printf("  - LOKI_USERNAME: %s", lokiUsername)
	} else {
		log.Println("  - LOKI_USERNAME: not set")
	}
	if os.Getenv("LOKI_PASSWORD") != "" {
		log.Println("  - LOKI_PASSWORD: ****** (set)")
	} else {
		log.Println("  - LOKI_PASSWORD: not set")
	}
	if os.Getenv("LOKI_TOKEN") != "" {
		log.Println("  - LOKI_TOKEN: ****** (set)")
	} else {
		log.Println("  - LOKI_TOKEN: not set")
	}

	// Create Streamable HTTP transport
	// The message endpoint is where the MCP protocol messages are sent
	log.Println("Creating Streamable HTTP transport...")

	streamableTransport, mcpHandler, err := transport.NewStreamableHTTPServerTransportAndHandler(
		transport.WithStreamableHTTPServerTransportAndHandlerOptionStateMode(transport.Stateless),
	)
	if err != nil {
		log.Fatalf("Failed to create streamable HTTP transport: %v", err)
	}
	log.Println("Streamable HTTP transport created successfully (Stateless mode)")

	// Initialize MCP server
	log.Println("Initializing MCP server...")
	mcpServer, err := server.NewServer(streamableTransport, server.WithServerInfo(protocol.Implementation{
		Name:    "Loki MCP Server",
		Version: version,
	}))
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}
	log.Println("MCP server initialized successfully")

	// Register Loki query tool
	log.Println("Registering Loki tools...")

	// Create and register loki_query tool
	lokiQueryTool, err := handlers.NewLokiQueryToolProtocol()
	if err != nil {
		log.Fatalf("Failed to create loki_query tool: %v", err)
	}
	mcpServer.RegisterTool(lokiQueryTool, handlers.HandleLokiQueryProtocol)
	log.Println("  - loki_query tool registered")

	// Create and register loki_label_names tool
	lokiLabelNamesTool, err := handlers.NewLokiLabelNamesToolProtocol()
	if err != nil {
		log.Fatalf("Failed to create loki_label_names tool: %v", err)
	}
	mcpServer.RegisterTool(lokiLabelNamesTool, handlers.HandleLokiLabelNamesProtocol)
	log.Println("  - loki_label_names tool registered")

	// Create and register loki_label_values tool
	lokiLabelValuesTool, err := handlers.NewLokiLabelValuesToolProtocol()
	if err != nil {
		log.Fatalf("Failed to create loki_label_values tool: %v", err)
	}
	mcpServer.RegisterTool(lokiLabelValuesTool, handlers.HandleLokiLabelValuesProtocol)
	log.Println("  - loki_label_values tool registered")

	log.Println("All tools registered successfully")

	// Start MCP server in a goroutine
	go func() {
		log.Println("Starting MCP server...")
		if err := mcpServer.Run(); err != nil {
			log.Fatalf("MCP server error: %v", err)
		}
	}()

	// Create HTTP server with the MCP handler
	mux := http.NewServeMux()

	// Register the MCP endpoint (Bedrock AgentCore compliant)
	mux.Handle("/mcp", mcpHandler.HandleMCP())
	log.Println("Registered endpoint: /mcp (Bedrock AgentCore compliant)")

	// Start HTTP server
	addr := fmt.Sprintf("%s:%s", host, port)
	log.Println("=== Starting HTTP Server ===")
	log.Printf("Server Address: http://%s", addr)
	log.Printf("Streamable HTTP Endpoint: http://%s/mcp", addr)
	log.Println("Server is ready to accept connections")
	log.Println("Press Ctrl+C to shutdown")

	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("HTTP server listening on %s...", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("=== Shutdown signal received ===")
	log.Println("Shutting down server gracefully...")

	// Shutdown MCP server
	if err := mcpServer.Shutdown(context.Background()); err != nil {
		log.Printf("Error shutting down MCP server: %v", err)
	}

	// Shutdown HTTP server
	if err := httpServer.Shutdown(context.Background()); err != nil {
		log.Printf("Error shutting down HTTP server: %v", err)
	}

	log.Println("Server stopped")
}
