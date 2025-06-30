package main

import (
	"context"
	"encoding/json"
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

type OAuthProtectedResource struct {
	// Required: The uri that uniquely identifies the resource.
	Resource string `json:"resource"`
	// Lists the authorization servers that can be used to access the resource.
	AuthorizationServers []string `json:"authorization_servers"`
	// Optional: The OAuth 2.0 presentation methods supported by the resource.
	BearerMethodsSupported []string `json:"bearer_methods_supported,omitempty"`
	// Optional: Where the resource's public keys live
	JwksURI string `json:"jwks_uri,omitempty"`
	// Recommended
	ScopesSupported []string `json:"scopes_supported,omitempty"`
}

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

func rejectWithOAuthResponseCodes(rw http.ResponseWriter) {
	resource_metadata := "http://localhost:8080/.well-known/oauth-protected-resource"
	authorization_uri := "http://localhost:8080/auth/smoke"
	header_response := fmt.Sprintf(`Bearer realm="spotify-go-server",resource_metadata="%s",authorization_uri="%s",error="unauthorized"`, resource_metadata, authorization_uri)
	rw.Header().Set("WWW-Authenticate", header_response)
	rw.WriteHeader(http.StatusUnauthorized)
	body := `{"error":"unauthorized","error_description":"You must authenticate to access this resource"}`
	bodyJson, _ := json.Marshal(body)
	rw.Write(bodyJson)
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get(AuthorizationHeader)
		if auth == "" {
			// TODO: make better instead of just missing auth header
			log.Printf("Missing Authorization header, redirecting to the oauth endpoints")
			rejectWithOAuthResponseCodes(w)
			return
		} else if !ValidateJWT(r) {
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

func returnWellKnownAuthServer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Returning well-known OAuth protected resource endpoint")

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	body := OAuthProtectedResource{
		Resource:               "http://localhost:8080/",
		AuthorizationServers:   []string{"http://localhost:8080/auth/smoke"},
		BearerMethodsSupported: []string{"header"},
		ScopesSupported:        []string{"openid", "profile", "email"},
	}

	// ignore error for simplicity
	bodyJSON, _ := json.Marshal(body)
	w.Write(bodyJSON)
}

func returnWellKnownProxy(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Returning well-known OAuth protected server metadata")
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// TODO: Update this to return the actual OAuth server metadatas
	w.Write([]byte(`{"test":"test"}`))
}

func main() {
	flag.StringVar(&port, "port", "8080", "Port to run the MCP server on")
	flag.Parse()

	mux := http.NewServeMux()

	// Simple health endpoint
	mux.HandleFunc("/health", handleHealth)

	// Adding MCP spec endpoints
	mux.HandleFunc("/.well-known/oauth-protected-resource", returnWellKnownAuthServer)
	mux.HandleFunc("/.well-known/oauth-authorizatioin-server", returnWellKnownProxy)

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
