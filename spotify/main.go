package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	port string
)

const (
	AuthorizationHeader string = "Authorization"
)

type authKey struct{}

func withAuthKey(ctx context.Context, auth string) context.Context {
	return context.WithValue(ctx, authKey{}, auth)
}

func authFromRequest(ctx context.Context, r *http.Request) context.Context {
	return withAuthKey(ctx, r.Header.Get(AuthorizationHeader))
}

func ValidateJWT(r *http.Request) bool {
	return true
}

func handleEchoTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	message, ok := arguments["message"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid arguments: message is required")
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Echo: %s", message)),
		},
	}, nil
}

func NewMCPServer() *server.MCPServer {
	hooks := &server.Hooks{}

	mcpServer := server.NewMCPServer("vscode-spotify/tools", "0.0.1",
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithHooks(hooks),
	)

	// Add a simple echo tool
	mcpServer.AddTool(mcp.NewTool("echo",
		mcp.WithDescription("Echoes back the input"),
		mcp.WithString("message",
			mcp.Description("Message to echo"),
			mcp.Required(),
		),
	), handleEchoTool)

	return mcpServer
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ValidateJWT(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
		// ctx := authFromRequest(r.Context(), r)
		// next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HTTP endpoints
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"UP"}`))
}

func handleAuthSmokeTest(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get(AuthorizationHeader)
	fmt.Printf("Received header %s request for auth smoke test\n", auth)

	if auth == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"AUTHENTICATED"}`))
}

func main() {
	flag.StringVar(&port, "port", "8080", "Port to run the MCP server on")
	flag.Parse()

	mux := http.NewServeMux()

	// Simple health endpoint
	mux.HandleFunc("/health", handleHealth)

	// Add the mcp server endpoint with the auth middleware
	mcpServer := NewMCPServer()
	httpServer := server.NewStreamableHTTPServer(mcpServer, server.WithHTTPContextFunc(authFromRequest))
	mux.Handle("/mcp", authMiddleware(http.HandlerFunc(httpServer.ServeHTTP)))
	mux.Handle("/auth/smoke", authMiddleware(http.HandlerFunc(handleAuthSmokeTest)))

	// Start the server
	log.Printf("HTTP server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
