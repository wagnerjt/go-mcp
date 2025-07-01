package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/grokify/go-pkce"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/oauth2"
)

var (
	port              string
	well_known_config []byte
	// In-memory store for PKCE state and code_verifier
	pkceStore            = make(map[string]string) // state -> code_verifier
	Client_Id     string = getEnv("SPOTIFY_CLIENT_ID")
	Client_Secret string = getEnv("SPOTIFY_CLIENT_SECRET")
)

const (
	AuthorizationHeader string = "Authorization"
	QueryState          string = "state"
	QueryCode           string = "code"
	RedirectURL         string = "http://127.0.0.1:8080/auth/callback"
	// Spotify endpoints from .well-known (hardcoded for now)
	SpotifyAuthEndpoint  = "https://accounts.spotify.com/authorize"
	SpotifyTokenEndpoint = "https://accounts.spotify.com/api/token"
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

type OAuthRedirectHandler struct {
	State        string
	CodeVerifier string
	OAuthConfig  *oauth2.Config
}

type AuthUrl struct {
	URL          string
	State        string
	CodeVerifier string
}

// Implement the http.Handler interface
func (h *OAuthRedirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract the code from the request
	code := r.URL.Query().Get(QueryCode)
	state := r.URL.Query().Get(QueryState)
	if code == "" || state == "" {
		http.Error(w, "Missing code or state parameter", http.StatusBadRequest)
		return
	}
	// TODO: Validate the state does not have timing attacks on it..

	codeVerifier, ok := pkceStore[state]
	if !ok {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}
	delete(pkceStore, state) // Clean up

	// Use the code to exchange for an access token
	token, err := h.OAuthConfig.Exchange(context.Background(), code,
		oauth2.SetAuthURLParam(pkce.ParamCodeVerifier, codeVerifier),
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to exchange token: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Received token: %s", token.AccessToken)
	// Redirect to a success page or return a message
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"authenticated"}`))
}

func getEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("Environment variable %s not set", key)
	}
	return value
}

func AuthorizationUrl(config *oauth2.Config) (*AuthUrl, error) {
	codeVerifier, _ := pkce.NewCodeVerifier(48)

	codeChallenge := pkce.CodeChallengeS256(codeVerifier)
	state := "spotify-auth-state"
	authUrl := config.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam(pkce.ParamCodeChallenge, codeChallenge),
		oauth2.SetAuthURLParam(pkce.ParamCodeChallengeMethod, pkce.MethodS256),
	)

	return &AuthUrl{
		URL:          authUrl,
		State:        state,
		CodeVerifier: codeVerifier,
	}, nil
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

func textResponse(rw http.ResponseWriter, status int, body string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	if body != "" {
		rw.Write([]byte(body))
	}
}

func rejectWithOAuthResponseCodes(rw http.ResponseWriter) {
	resource_metadata := "http://127.0.0.1:8080/.well-known/oauth-protected-resource"
	authorization_uri := SpotifyAuthEndpoint
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

func GetResponseBodyBytes(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Failed to fetch %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Failed to fetch %s: status code %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body from %s: %v", url, err)
	}

	return body
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
	// body := OAuthProtectedResource{
	// 	Resource:               "https://accounts.spotify.com",
	// 	AuthorizationServers:   []string{SpotifyAuthEndpoint},
	// 	BearerMethodsSupported: []string{"header"},
	// 	ScopesSupported:        []string{"user-read-private", "user-read-email"},
	// }

	proxy_body := OAuthProtectedResource{
		Resource:               "http://127.0.0.1:8080/",
		AuthorizationServers:   []string{SpotifyAuthEndpoint},
		BearerMethodsSupported: []string{"header"},
		ScopesSupported:        []string{"user-read-private", "user-read-email"},
	}

	// ignore error for simplicity
	bodyJSON, _ := json.Marshal(proxy_body)
	w.Write(bodyJSON)
}

func returnWellKnownProxy(w http.ResponseWriter, r *http.Request) {
	// Spotify does not have a well-known endpoint for OAuth authorization resources, proxy it for now
	fmt.Println("Returning well-known OAuth protected server metadata")
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(well_known_config))
}

// Handler to start the PKCE OAuth flow
func handleSpotifyLogin(w http.ResponseWriter, r *http.Request) {
	clientID := Client_Id
	redirectURI := RedirectURL
	scopes := "user-read-private user-read-email"

	codeVerifier, _ := pkce.NewCodeVerifier(48)
	codeChallenge := pkce.CodeChallengeS256(codeVerifier)
	state := fmt.Sprintf("state-%d", len(pkceStore)+1) // simple state
	pkceStore[state] = codeVerifier

	authURL := fmt.Sprintf("%s?client_id=%s&response_type=code&redirect_uri=%s&scope=%s&state=%s&code_challenge=%s&code_challenge_method=S256",
		SpotifyAuthEndpoint, clientID, redirectURI, scopes, state, codeChallenge)

	http.Redirect(w, r, authURL, http.StatusFound)
}

func main() {
	flag.StringVar(&port, "port", "8080", "Port to run the MCP server on")
	flag.Parse()

	// Get spotify's well-known configuration initially for proxying
	well_known_config = GetResponseBodyBytes("https://accounts.spotify.com/.well-known/openid-configuration")

	mux := http.NewServeMux()

	// Simple health endpoint
	mux.HandleFunc("/health", handleHealth)

	// Adding MCP spec endpoints
	mux.HandleFunc("/.well-known/oauth-protected-resource", returnWellKnownAuthServer)
	mux.HandleFunc("/.well-known/oauth-authorization-server", returnWellKnownProxy)
	// Provide a valid OAuthConfig to the callback handler
	mux.Handle("/auth/callback", &OAuthRedirectHandler{
		OAuthConfig: &oauth2.Config{
			ClientID:     Client_Id,
			ClientSecret: Client_Secret,
			RedirectURL:  RedirectURL,
			Scopes:       []string{"user-read-private", "user-read-email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  SpotifyAuthEndpoint,
				TokenURL: SpotifyTokenEndpoint,
			},
		},
	})
	// Add the login endpoint
	mux.HandleFunc("/auth/spotify/login", handleSpotifyLogin)

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
