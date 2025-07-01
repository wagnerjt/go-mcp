# Spotify MCP Go Server

This is a Go server implementing the MCP protocol with Spotify OAuth2 PKCE authentication to embed with VsCode.

## Getting Started

### Prerequisites

- Go 1.20+
- Spotify Developer Account ([Create an app](https://developer.spotify.com/dashboard/applications))

### Environment Variables

Set the following environment variables (or edit the code directly):

- `SPOTIFY_CLIENT_ID` – Your Spotify app client ID
- `SPOTIFY_CLIENT_SECRET` – Your Spotify app client secret

### Running the Server

```sh
# Set environment variables (PowerShell example)
export SPOTIFY_CLIENT_ID = "your_spotify_client_id"
export SPOTIFY_CLIENT_SECRET = "your_spotify_client_secret"

# Run the server
cd spotify
go mod tidy
go run main.go
```

The server will start on `http://localhost:8080` by default.

### Endpoints

- `GET /health` – Health check
- `GET /.well-known/oauth-protected-resource` – OAuth resource metadata
- `GET /.well-known/oauth-authorization-server` – OAuth server metadata (stub)
  - Used to proxy [Spotify's OIDC .well-known config url](https://accounts.spotify.com/.well-known/openid-configuration)
- `GET /auth/smoke` – Auth test endpoint (protected)
- `POST /mcp` – MCP protocol endpoint (protected)

Sanity check to make sure the actual token generated via PKCE

- `GET /auth/spotify/login` – Start Spotify OAuth2 PKCE flow
- `GET /auth/callback` – OAuth2 redirect URI (set this in your Spotify app)

You can use the `spotify/client/main.go` to test the `/v1/me` once you have the token

### Example `mcp.json`

```json
{
  "inputs": [
    {
      "type": "promptString",
      "id": "mcp-token",
      "description": "Enter your MCP token",
      "password": true
    }
  ],
  "servers": {
    "local-spotify-server": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

Mcp server only returns an echo with any Authorization header set.

### Notes

- Set your Spotify app's redirect URI to `http://127.0.0.1:8080/auth/callback` in the Spotify Developer Dashboard (due to their restrictions with localhost).
- The PKCE code_verifier is stored in-memory for demo purposes..do not deploy this in production
- Replace client ID/secret in the code or use environment variables as shown above.
