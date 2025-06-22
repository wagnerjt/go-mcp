package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	transport string
	port      string
)

type ToolName string

const (
	ECHO ToolName = "echo"
	ADD  ToolName = "add"
	AUTH ToolName = "check_auth"
)

type authKey struct{}

func withAuthKey(ctx context.Context, auth string) context.Context {
	return context.WithValue(ctx, authKey{}, auth)
}

func authFromRequest(ctx context.Context, r *http.Request) context.Context {
	return withAuthKey(ctx, r.Header.Get("Authorization"))
}

// tokenFromContext extracts the auth token from the context.
// This can be used by tools to extract the token regardless of the
// transport being used by the server.
func tokenFromContext(ctx context.Context) (string, error) {
	auth, ok := ctx.Value(authKey{}).(string)
	if !ok {
		return "", fmt.Errorf("missing auth")
	}
	return auth, nil
}

func NewMCPServer() *server.MCPServer {
	mcpServer := server.NewMCPServer(
		"go-mcp/tools",
		"0.0.1",
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	mcpServer.AddTool(mcp.NewTool(string(ECHO),
		mcp.WithDescription("Echoes back the input"),
		mcp.WithString("message",
			mcp.Description("Message to echo"),
			mcp.Required(),
		),
	), handleEchoTool)

	mcpServer.AddTool(mcp.NewTool("get_current_time",
		mcp.WithDescription("Get the current time"),
	), handleCurrentTime)

	mcpServer.AddTool(
		mcp.NewTool("notify"),
		handleSendNotification,
	)

	mcpServer.AddTool(mcp.NewTool(string(ADD),
		mcp.WithDescription("Adds two numbers"),
		mcp.WithNumber("a",
			mcp.Description("First number"),
			mcp.Required(),
		),
		mcp.WithNumber("b",
			mcp.Description("Second number"),
			mcp.Required(),
		),
	), handleAddTool)

	mcpServer.AddTool(mcp.NewTool(string(AUTH),
		mcp.WithDescription("Checks for auth calls in the header"),
		mcp.WithString("message",
			mcp.Description("Message to echo"),
			mcp.Required(),
		),
	), handleAuthTool)

	mcpServer.AddNotificationHandler("notification", handleNotification)

	return mcpServer
}

func handleEchoTool(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	message, ok := arguments["message"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid message argument")
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Echo: %s", message),
			},
		},
	}, nil
}

func handleCurrentTime(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Time: %s", time.Now().Format(time.RFC3339)),
			},
		},
	}, nil
}

func handleAddTool(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	a, ok1 := arguments["a"].(float64)
	b, ok2 := arguments["b"].(float64)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("invalid number arguments")
	}
	sum := a + b
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("The sum of %f and %f is %f.", a, b, sum),
			},
		},
	}, nil
}

func handleAuthTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	message, ok := request.GetArguments()["message"].(string)
	if !ok {
		return nil, fmt.Errorf("missing required message")
	}

	token, err := tokenFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(token, "Bearer ") && strings.Split(token, "Bearer ")[1] == "sk-1234" {

	} else {
		return nil, errors.New("token not correct")
	}

	return mcp.NewToolResultText(fmt.Sprintf("Echoing %s with auth successful", message)), nil
}

func handleSendNotification(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {

	server := server.ServerFromContext(ctx)

	err := server.SendNotificationToClient(
		ctx,
		"notifications/progress",
		map[string]interface{}{
			"progress":      10,
			"total":         10,
			"progressToken": 0,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send notification: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "notification sent successfully",
			},
		},
	}, nil
}

func handleNotification(
	ctx context.Context,
	notification mcp.JSONRPCNotification,
) {
	log.Printf("Received notification: %s", notification.Method)
}

func main() {
	flag.StringVar(&transport, "t", "sse", "Transport type (stdio, sse, or http)")
	flag.StringVar(&port, "p", "8080", "Port to listen on")
	flag.Parse()

	mcpServer := NewMCPServer()

	// Only check for "sse" since stdio is the default
	if transport == "sse" {
		sseServer := server.NewSSEServer(mcpServer, server.WithSSEContextFunc(authFromRequest))
		log.Printf("SSE server listening on port %s", port)
		if err := sseServer.Start(":" + port); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else if transport == "http" {
		httpServer := server.NewStreamableHTTPServer(mcpServer, server.WithHTTPContextFunc(authFromRequest))
		log.Printf("HTTP server listening on port %s", port)
		if err := httpServer.Start(":" + port); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else if transport == "stdio" {
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		log.Fatalf("Unsupported transport type: %s", transport)
		panic("Unsupported transport type")
	}
}
