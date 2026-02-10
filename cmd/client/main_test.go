package main

import (
	"os"
	"testing"
	"time"
)

// TestLoadConfigDefaults verifies that LoadConfig returns default values
// when no environment variables or flags are set
func TestLoadConfigDefaults(t *testing.T) {
	// Clear any environment variables
	oldServerURL := os.Getenv("MCP_SERVER_URL")
	oldTimeout := os.Getenv("LOKI_QUERY_TIMEOUT")
	os.Unsetenv("MCP_SERVER_URL")
	os.Unsetenv("LOKI_QUERY_TIMEOUT")
	defer func() {
		if oldServerURL != "" {
			os.Setenv("MCP_SERVER_URL", oldServerURL)
		}
		if oldTimeout != "" {
			os.Setenv("LOKI_QUERY_TIMEOUT", oldTimeout)
		}
	}()

	cfg := LoadConfigWithArgs([]string{})

	// Verify default server URL
	expectedURL := "http://localhost:8000/mcp"
	if cfg.ServerURL != expectedURL {
		t.Errorf("Expected default ServerURL '%s', got '%s'", expectedURL, cfg.ServerURL)
	}

	// Verify default timeout
	expectedTimeout := 30 * time.Second
	if cfg.Timeout != expectedTimeout {
		t.Errorf("Expected default Timeout %v, got %v", expectedTimeout, cfg.Timeout)
	}
}

// TestLoadConfigEnvironmentVariable verifies that LoadConfig uses
// the MCP_SERVER_URL environment variable when set
func TestLoadConfigEnvironmentVariable(t *testing.T) {
	// Set environment variable
	testURL := "http://test-server:9000/api"
	oldServerURL := os.Getenv("MCP_SERVER_URL")
	os.Setenv("MCP_SERVER_URL", testURL)
	defer func() {
		if oldServerURL != "" {
			os.Setenv("MCP_SERVER_URL", oldServerURL)
		} else {
			os.Unsetenv("MCP_SERVER_URL")
		}
	}()

	cfg := LoadConfigWithArgs([]string{})

	// Verify environment variable is used
	if cfg.ServerURL != testURL {
		t.Errorf("Expected ServerURL from environment '%s', got '%s'", testURL, cfg.ServerURL)
	}
}

// TestLoadConfigCommandLineFlag verifies that LoadConfig uses
// the --server-url command-line flag when provided
func TestLoadConfigCommandLineFlag(t *testing.T) {
	// Clear environment variable
	oldServerURL := os.Getenv("MCP_SERVER_URL")
	os.Unsetenv("MCP_SERVER_URL")
	defer func() {
		if oldServerURL != "" {
			os.Setenv("MCP_SERVER_URL", oldServerURL)
		}
	}()

	testURL := "http://flag-server:7000/endpoint"
	cfg := LoadConfigWithArgs([]string{"--server-url", testURL})

	// Verify command-line flag is used
	if cfg.ServerURL != testURL {
		t.Errorf("Expected ServerURL from flag '%s', got '%s'", testURL, cfg.ServerURL)
	}
}

// TestLoadConfigPrecedence verifies that command-line flag takes precedence
// over environment variable, which takes precedence over default
func TestLoadConfigPrecedence(t *testing.T) {
	// Set environment variable
	envURL := "http://env-server:8000/mcp"
	flagURL := "http://flag-server:9000/api"

	oldServerURL := os.Getenv("MCP_SERVER_URL")
	os.Setenv("MCP_SERVER_URL", envURL)
	defer func() {
		if oldServerURL != "" {
			os.Setenv("MCP_SERVER_URL", oldServerURL)
		} else {
			os.Unsetenv("MCP_SERVER_URL")
		}
	}()

	cfg := LoadConfigWithArgs([]string{"--server-url", flagURL})

	// Verify command-line flag takes precedence over environment variable
	if cfg.ServerURL != flagURL {
		t.Errorf("Expected ServerURL from flag (highest precedence) '%s', got '%s'", flagURL, cfg.ServerURL)
	}
}
