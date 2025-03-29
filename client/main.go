package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

var mcpUri string

// pulled from https://github.com/mark3labs/mcp-go/blob/main/client/sse_test.go
func main() {
	flag.StringVar(&mcpUri, "mcpUri", "http://localhost:8080/sse", "Fully qualified mcpUri to connect to including port i.e. http://localhost:8080/sse")
	flag.Parse()

	// Create MCP client
	c, err := client.NewSSEMCPClient(mcpUri)
	if err != nil {
		log.Fatalf("Error creating client: %v\n", err)

	}
	defer c.Close()

	// Start the client
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Start(ctx); err != nil {
		log.Fatalf("Error starting client: %v", err)
	}

	// init request
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "go-client",
		Version: "0.0.1",
	}

	result, err := c.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	log.Printf("Connected to server with name %s", result.ServerInfo.Name)

	// Test Ping
	if err := c.Ping(ctx); err != nil {
		log.Fatalf("Ping failed: %v", err)
	}

	log.Printf("Ping successful")

	// Test ListTools
	toolsRequest := mcp.ListToolsRequest{}

	// toolsResponse = mcp.ListToolsResponse{}
	toolsResponse, err := c.ListTools(ctx, toolsRequest)
	if err != nil {
		log.Fatalf("ListTools failed: %v", err)
	}

	log.Printf("Found %d tools", len(toolsResponse.Tools))

	for _, tool := range toolsResponse.Tools {
		log.Printf("Tool: %s", tool.Name)
	}

	// callToolGoServer(ctx, c)
	callToolLiteLLMServer(ctx, c)
}

func callToolGoServer(ctx context.Context, c *client.SSEMCPClient) {
	log.Printf("Calling add tool")

	request := mcp.CallToolRequest{}
	request.Params.Name = "add"
	request.Params.Arguments = map[string]interface{}{
		"a": 1.50,
		"b": 5,
	}

	result, err := c.CallTool(ctx, request)
	if err != nil || result.IsError {
		log.Fatalf("CallTool failed: %v", err)
	}

	if len(result.Content) != 1 {
		log.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}
	log.Printf("Result: %s", result.Content[0].(mcp.TextContent).Text)
}

func callToolLiteLLMServer(ctx context.Context, c *client.SSEMCPClient) {
	log.Printf("Calling get_current_time tool")

	request := mcp.CallToolRequest{}
	request.Params.Name = "get_current_time"
	request.Params.Arguments = map[string]interface{}{
		"format": "short",
	}

	result, err := c.CallTool(ctx, request)
	if err != nil || result.IsError {
		log.Fatalf("CallTool failed: %v", err)
	}

	if len(result.Content) != 1 {
		log.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}
	log.Printf("Result: %s", result.Content[0].(mcp.TextContent).Text)
}
