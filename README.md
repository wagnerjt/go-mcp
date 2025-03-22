# Getting Started

This project uses the LiteLLM MCP implementation with a few other MCP servers and clients. In particular the go-based mcp server and client from [mark3labs](https://github.com/mark3labs/mcp-go/tree/main/examples/everything)

## Requirements

You need to have the following installed

* go 1.24.1
* uv

### Running Litellm instance

```sh
uv venv --python 3.13
source .venv/bin/activate # windows -- source .venv/Scripts/active
uv pip install -r bridge/requirements.txt
```

Or via docker

```sh
cd bridge
docker compose up
```

### Running MCP server

```sh
cd server
go mod download
go run main.go -t sse -p 8080 # transport over http network with port 8080
```

### Running MCP client

```sh
cd client
go mod download
go run main.go -mcpUri 'http://localhost:2020/sse' # connect to mcp server on uri
```
