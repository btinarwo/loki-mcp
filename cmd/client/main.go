package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/client"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/transport"
)

// Config holds the client configuration
type Config struct {
	ServerURL string
	Timeout   time.Duration
}

// LoadConfig loads configuration from environment variables and command-line flags
func LoadConfig() *Config {
	return LoadConfigWithArgs(os.Args[1:])
}

// LoadConfigWithArgs loads configuration from environment variables and provided arguments
// This function is useful for testing
func LoadConfigWithArgs(args []string) *Config {
	// Create a new flag set for parsing
	fs := flag.NewFlagSet("client", flag.ContinueOnError)
	serverURL := fs.String("server-url", "", "Server URL (overrides MCP_SERVER_URL environment variable)")

	// Parse the provided arguments
	fs.Parse(args)

	// Default values
	cfg := &Config{
		ServerURL: "http://localhost:8000/mcp",
		Timeout:   30 * time.Second,
	}

	// Check environment variable for server URL
	if envURL := os.Getenv("MCP_SERVER_URL"); envURL != "" {
		cfg.ServerURL = envURL
	}

	// Command-line flag takes precedence
	if *serverURL != "" {
		cfg.ServerURL = *serverURL
	}

	// Check environment variable for timeout
	if envTimeout := os.Getenv("LOKI_QUERY_TIMEOUT"); envTimeout != "" {
		if timeoutSecs, err := strconv.Atoi(envTimeout); err == nil && timeoutSecs > 0 {
			cfg.Timeout = time.Duration(timeoutSecs) * time.Second
		}
	}

	return cfg
}

func main() {
	// Load configuration
	cfg := LoadConfig()

	// Parse remaining arguments (after flags)
	fs := flag.NewFlagSet("client", flag.ContinueOnError)
	fs.String("server-url", "", "Server URL (overrides MCP_SERVER_URL environment variable)")
	fs.Parse(os.Args[1:])
	args := fs.Args()

	if len(args) < 1 {
		showUsage()
		os.Exit(1)
	}

	// Create transport client
	transportClient, err := transport.NewStreamableHTTPClientTransport(cfg.ServerURL)
	if err != nil {
		log.Fatalf("Failed to create transport client: %v", err)
	}

	// Initialize MCP client
	mcpClient, err := client.NewClient(transportClient)
	if err != nil {
		log.Fatalf("Failed to create MCP client: %v", err)
	}
	defer mcpClient.Close()

	ctx := context.Background()

	// Process commands
	switch args[0] {
	case "loki_query":
		if len(args) < 2 {
			fmt.Println("Usage: client loki_query [url] <query> [start] [end] [limit]")
			fmt.Println("Examples:")
			fmt.Println("  client loki_query \"{job=\\\"varlogs\\\"}\"")
			fmt.Println("  client loki_query http://localhost:3100 \"{job=\\\"varlogs\\\"}\"")
			fmt.Println("  client loki_query \"{job=\\\"varlogs\\\"}\" \"-1h\" \"now\" 100")
			os.Exit(1)
		}

		var lokiURL, query, start, end, org string
		var limit float64

		// Check if the first argument is a URL or a query
		if strings.HasPrefix(args[1], "http") {
			// First arg is URL, second is query
			if len(args) < 3 {
				fmt.Println("Error: When providing a URL, you must also provide a query")
				os.Exit(1)
			}
			lokiURL = args[1]
			query = args[2]
			argOffset := 3

			// Optional parameters with URL
			if len(args) > argOffset {
				start = args[argOffset]
			}

			if len(args) > argOffset+1 {
				end = args[argOffset+1]
			}

			if len(args) > argOffset+2 {
				limitVal, err := strconv.ParseFloat(args[argOffset+2], 64)
				if err != nil {
					log.Fatalf("Invalid number for limit: %v", err)
				}
				limit = limitVal
			}

			if len(args) > argOffset+3 {
				org = args[argOffset+3]
			}
		} else {
			// First arg is the query (URL comes from environment)
			query = args[1]
			argOffset := 2

			// Optional parameters without URL
			if len(args) > argOffset {
				start = args[argOffset]
			}

			if len(args) > argOffset+1 {
				end = args[argOffset+1]
			}

			if len(args) > argOffset+2 {
				limitVal, err := strconv.ParseFloat(args[argOffset+2], 64)
				if err != nil {
					log.Fatalf("Invalid number for limit: %v", err)
				}
				limit = limitVal
			}

			if len(args) > argOffset+3 {
				org = args[argOffset+3]
			}
		}

		// Create arguments map
		toolArgs := map[string]interface{}{
			"query": query,
		}

		// Add URL parameter if provided
		if lokiURL != "" {
			toolArgs["url"] = lokiURL
		}

		// Add optional parameters if provided
		if start != "" {
			toolArgs["start"] = start
		}

		if end != "" {
			toolArgs["end"] = end
		}

		if limit > 0 {
			toolArgs["limit"] = limit
		}

		if org != "" {
			toolArgs["org"] = org
		}

		// Marshal arguments to JSON
		argsJSON, err := json.Marshal(toolArgs)
		if err != nil {
			log.Fatalf("Failed to marshal arguments: %v", err)
		}

		// Call the tool
		result, err := mcpClient.CallTool(ctx, &protocol.CallToolRequest{
			Name:         "loki_query",
			RawArguments: argsJSON,
		})
		if err != nil {
			log.Fatalf("Failed to call tool: %v", err)
		}

		// Print the result
		for _, content := range result.Content {
			if textContent, ok := content.(*protocol.TextContent); ok {
				fmt.Println(textContent.Text)
			}
		}

	case "loki_label_names":
		// Create arguments map
		toolArgs := map[string]interface{}{}

		// Check for optional URL parameter
		if len(args) > 1 && strings.HasPrefix(args[1], "http") {
			toolArgs["url"] = args[1]
		}

		// Marshal arguments to JSON
		argsJSON, err := json.Marshal(toolArgs)
		if err != nil {
			log.Fatalf("Failed to marshal arguments: %v", err)
		}

		// Call the tool
		result, err := mcpClient.CallTool(ctx, &protocol.CallToolRequest{
			Name:         "loki_label_names",
			RawArguments: argsJSON,
		})
		if err != nil {
			log.Fatalf("Failed to call tool: %v", err)
		}

		// Print the result
		for _, content := range result.Content {
			if textContent, ok := content.(*protocol.TextContent); ok {
				fmt.Println(textContent.Text)
			}
		}

	case "loki_label_values":
		if len(args) < 2 {
			fmt.Println("Usage: client loki_label_values <label> [url]")
			fmt.Println("Examples:")
			fmt.Println("  client loki_label_values job")
			fmt.Println("  client loki_label_values job http://localhost:3100")
			os.Exit(1)
		}

		// Create arguments map
		toolArgs := map[string]interface{}{
			"label": args[1],
		}

		// Check for optional URL parameter
		if len(args) > 2 && strings.HasPrefix(args[2], "http") {
			toolArgs["url"] = args[2]
		}

		// Marshal arguments to JSON
		argsJSON, err := json.Marshal(toolArgs)
		if err != nil {
			log.Fatalf("Failed to marshal arguments: %v", err)
		}

		// Call the tool
		result, err := mcpClient.CallTool(ctx, &protocol.CallToolRequest{
			Name:         "loki_label_values",
			RawArguments: argsJSON,
		})
		if err != nil {
			log.Fatalf("Failed to call tool: %v", err)
		}

		// Print the result
		for _, content := range result.Content {
			if textContent, ok := content.(*protocol.TextContent); ok {
				fmt.Println(textContent.Text)
			}
		}

	case "list_tools":
		// Get available tools
		tools, err := mcpClient.ListTools(ctx)
		if err != nil {
			log.Fatalf("Failed to list tools: %v", err)
		}

		fmt.Println("Available tools:")
		for _, tool := range tools.Tools {
			fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
		}

	default:
		showUsage()
		os.Exit(1)
	}
}

func showUsage() {
	fmt.Println("Usage:")
	fmt.Println("  client loki_query [url] <query> [start] [end] [limit]")
	fmt.Println("    Examples:")
	fmt.Println("      client loki_query \"{job=\\\"varlogs\\\"}\"")
	fmt.Println("      client loki_query http://localhost:3100 \"{job=\\\"varlogs\\\"}\"")
	fmt.Println("      client loki_query \"{job=\\\"varlogs\\\"}\" \"-1h\" \"now\" 100")
	fmt.Println("      client loki_query \"{job=\\\"varlogs\\\"}\" \"-1h\" \"now\" 100 \"tenant-123\"")
	fmt.Println()
	fmt.Println("  client loki_label_names [url]")
	fmt.Println("    Examples:")
	fmt.Println("      client loki_label_names")
	fmt.Println("      client loki_label_names http://localhost:3100")
	fmt.Println()
	fmt.Println("  client loki_label_values <label> [url]")
	fmt.Println("    Examples:")
	fmt.Println("      client loki_label_values job")
	fmt.Println("      client loki_label_values job http://localhost:3100")
	fmt.Println()
	fmt.Println("  client list_tools")
	fmt.Println("    List all available tools")
}
