package handlers

import (
"context"
"fmt"
"os"
"time"

"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// LokiQueryRequest represents the arguments for loki_query tool
type LokiQueryRequest struct {
	Query    string  `json:"query" description:"LogQL query string"`
	URL      string  `json:"url,omitempty" description:"Loki server URL"`
	Username string  `json:"username,omitempty" description:"Username for basic authentication"`
	Password string  `json:"password,omitempty" description:"Password for basic authentication"`
	Token    string  `json:"token,omitempty" description:"Bearer token for authentication"`
	Start    string  `json:"start,omitempty" description:"Start time for the query"`
	End      string  `json:"end,omitempty" description:"End time for the query"`
	Limit    float64 `json:"limit,omitempty" description:"Maximum number of entries to return"`
	Org      string  `json:"org,omitempty" description:"Organization ID for the query"`
	Format   string  `json:"format,omitempty" description:"Output format: raw, json, or text"`
}

// LokiLabelNamesRequest represents the arguments for loki_label_names tool
type LokiLabelNamesRequest struct {
	URL      string `json:"url,omitempty" description:"Loki server URL"`
	Username string `json:"username,omitempty" description:"Username for basic authentication"`
	Password string `json:"password,omitempty" description:"Password for basic authentication"`
	Token    string `json:"token,omitempty" description:"Bearer token for authentication"`
	Start    string `json:"start,omitempty" description:"Start time for the query"`
	End      string `json:"end,omitempty" description:"End time for the query"`
	Org      string `json:"org,omitempty" description:"Organization ID for the query"`
	Format   string `json:"format,omitempty" description:"Output format: raw, json, or text"`
}

// LokiLabelValuesRequest represents the arguments for loki_label_values tool
type LokiLabelValuesRequest struct {
	Label    string `json:"label" description:"Label name to get values for"`
	URL      string `json:"url,omitempty" description:"Loki server URL"`
	Username string `json:"username,omitempty" description:"Username for basic authentication"`
	Password string `json:"password,omitempty" description:"Password for basic authentication"`
	Token    string `json:"token,omitempty" description:"Bearer token for authentication"`
	Start    string `json:"start,omitempty" description:"Start time for the query"`
	End      string `json:"end,omitempty" description:"End time for the query"`
	Org      string `json:"org,omitempty" description:"Organization ID for the query"`
	Format   string `json:"format,omitempty" description:"Output format: raw, json, or text"`
}

// NewLokiQueryToolProtocol creates a tool using the protocol library
func NewLokiQueryToolProtocol() (*protocol.Tool, error) {
	return protocol.NewTool("loki_query", "Run a query against Grafana Loki", LokiQueryRequest{})
}

// NewLokiLabelNamesToolProtocol creates a tool using the protocol library
func NewLokiLabelNamesToolProtocol() (*protocol.Tool, error) {
	return protocol.NewTool("loki_label_names", "Get all label names from Grafana Loki", LokiLabelNamesRequest{})
}

// NewLokiLabelValuesToolProtocol creates a tool using the protocol library
func NewLokiLabelValuesToolProtocol() (*protocol.Tool, error) {
	return protocol.NewTool("loki_label_values", "Get all values for a specific label from Grafana Loki", LokiLabelValuesRequest{})
}

// HandleLokiQueryProtocol handles Loki query tool requests using protocol library
func HandleLokiQueryProtocol(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	req := new(LokiQueryRequest)
	if err := protocol.VerifyAndUnmarshal(request.RawArguments, req); err != nil {
		return nil, err
	}

	lokiURL := getEnvOrDefault(req.URL, EnvLokiURL, DefaultLokiURL)
	username := getEnvOrDefault(req.Username, EnvLokiUsername, "")
	password := getEnvOrDefault(req.Password, EnvLokiPassword, "")
	token := getEnvOrDefault(req.Token, EnvLokiToken, "")
	orgID := getEnvOrDefault(req.Org, EnvLokiOrgID, "")

	start := time.Now().Add(-1 * time.Hour).Unix()
	end := time.Now().Unix()
	limit := 100

	if req.Start != "" {
		startTime, err := parseTime(req.Start)
		if err != nil {
			return nil, fmt.Errorf("invalid start time: %v", err)
		}
		start = startTime.Unix()
	}

	if req.End != "" {
		endTime, err := parseTime(req.End)
		if err != nil {
			return nil, fmt.Errorf("invalid end time: %v", err)
		}
		end = endTime.Unix()
	}

	if req.Limit > 0 {
		limit = int(req.Limit)
	}

	format := "raw"
	if req.Format != "" {
		format = req.Format
	}

	queryURL, err := buildLokiQueryURL(lokiURL, req.Query, start, end, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to build query URL: %v", err)
	}

	result, err := executeLokiQuery(ctx, queryURL, username, password, token, orgID)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %v", err)
	}

	formattedResult, err := formatLokiResults(result, format)
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %v", err)
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: formattedResult,
			},
		},
	}, nil
}

// HandleLokiLabelNamesProtocol handles Loki label names tool requests using protocol library
func HandleLokiLabelNamesProtocol(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	req := new(LokiLabelNamesRequest)
	if err := protocol.VerifyAndUnmarshal(request.RawArguments, req); err != nil {
		return nil, err
	}

	lokiURL := getEnvOrDefault(req.URL, EnvLokiURL, DefaultLokiURL)
	username := getEnvOrDefault(req.Username, EnvLokiUsername, "")
	password := getEnvOrDefault(req.Password, EnvLokiPassword, "")
	token := getEnvOrDefault(req.Token, EnvLokiToken, "")
	orgID := getEnvOrDefault(req.Org, EnvLokiOrgID, "")

	start := time.Now().Add(-1 * time.Hour).Unix()
	end := time.Now().Unix()

	if req.Start != "" {
		startTime, err := parseTime(req.Start)
		if err != nil {
			return nil, fmt.Errorf("invalid start time: %v", err)
		}
		start = startTime.Unix()
	}

	if req.End != "" {
		endTime, err := parseTime(req.End)
		if err != nil {
			return nil, fmt.Errorf("invalid end time: %v", err)
		}
		end = endTime.Unix()
	}

	format := "raw"
	if req.Format != "" {
		format = req.Format
	}

	labelsURL, err := buildLokiLabelsURL(lokiURL, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to build labels URL: %v", err)
	}

	result, err := executeLokiLabelsQuery(ctx, labelsURL, username, password, token, orgID)
	if err != nil {
		return nil, fmt.Errorf("labels query execution failed: %v", err)
	}

	formattedResult, err := formatLokiLabelsResults(result, format)
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %v", err)
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: formattedResult,
			},
		},
	}, nil
}

// HandleLokiLabelValuesProtocol handles Loki label values tool requests using protocol library
func HandleLokiLabelValuesProtocol(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	req := new(LokiLabelValuesRequest)
	if err := protocol.VerifyAndUnmarshal(request.RawArguments, req); err != nil {
		return nil, err
	}

	lokiURL := getEnvOrDefault(req.URL, EnvLokiURL, DefaultLokiURL)
	username := getEnvOrDefault(req.Username, EnvLokiUsername, "")
	password := getEnvOrDefault(req.Password, EnvLokiPassword, "")
	token := getEnvOrDefault(req.Token, EnvLokiToken, "")
	orgID := getEnvOrDefault(req.Org, EnvLokiOrgID, "")

	start := time.Now().Add(-1 * time.Hour).Unix()
	end := time.Now().Unix()

	if req.Start != "" {
		startTime, err := parseTime(req.Start)
		if err != nil {
			return nil, fmt.Errorf("invalid start time: %v", err)
		}
		start = startTime.Unix()
	}

	if req.End != "" {
		endTime, err := parseTime(req.End)
		if err != nil {
			return nil, fmt.Errorf("invalid end time: %v", err)
		}
		end = endTime.Unix()
	}

	format := "raw"
	if req.Format != "" {
		format = req.Format
	}

	labelValuesURL, err := buildLokiLabelValuesURL(lokiURL, req.Label, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to build label values URL: %v", err)
	}

	result, err := executeLokiLabelValuesQuery(ctx, labelValuesURL, username, password, token, orgID)
	if err != nil {
		return nil, fmt.Errorf("label values query execution failed: %v", err)
	}

	formattedResult, err := formatLokiLabelValuesResults(req.Label, result, format)
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %v", err)
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: formattedResult,
			},
		},
	}, nil
}

// getEnvOrDefault returns the value if not empty, otherwise checks environment variable, otherwise returns default
func getEnvOrDefault(value, envKey, defaultValue string) string {
	if value != "" {
		return value
	}
	if envValue := os.Getenv(envKey); envValue != "" {
		return envValue
	}
	return defaultValue
}
