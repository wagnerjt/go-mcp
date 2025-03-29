# Getting Started

This project tests the initial [LiteLLM MCP](https://github.com/BerriAI/litellm/pull/9436) implementation with a few other MCP servers and clients. In particular the go-based mcp server and client from [mark3labs](https://github.com/mark3labs/mcp-go/tree/main/examples/everything)

## Requirements

You need to have the following installed

* go 1.24.1
* uv

### Setup deps

```sh
uv venv --python 3.13
source .venv/bin/activate # windows -- source .venv/Scripts/active
uv pip install -r requirements.txt

# cd into each and install dependencies
cd client;go mod download; cd ..
cd server; go mod download; cd ..
```

### Setup litellm proxy

Or via docker

```sh
docker compose up
```

### Running MCP Go server

```sh
cd server
go mod download
go run main.go -t sse -p 8080 # transport over http network with port 8080
```

### Running MCP Go client

```sh
# run go mcp server first on sse transport
cd client
go run main.go -mcpUri 'http://localhost:8080/sse' # connect to mcp server on uri
```

### Testing Litellm sdk MCP client

```sh
# run go mcp server first on sse transport

# run client
cd bridge
python litellm_client.py
```

### Testing python sdk on LiteLLM Proxy MCP

```sh
# run go litellm proxy
docker compose up

# run client
cd bridge
python mcp_client.py
# Initializing session...
# Session initialized
# Loading tools...
# Loaded 1 tools
# meta=None content=[TextContent(type='text', text='13:04', annotations=None)] isError=Fals
```

### Testing Go client on LiteLLM proxy MCP

```sh
# run go litellm proxy
docker compose up

cd bridge
go run main.go -mcpUri http://localhost:4000/mcp
# 2025/03/29 06:02:11 Connected to server with name litellm-mcp-server
# 2025/03/29 06:02:11 Ping successful
# 2025/03/29 06:02:11 Found 1 tools
# 2025/03/29 06:02:11 Tool: get_current_time
# 2025/03/29 06:02:11 Calling get_current_time tool
# 2025/03/29 06:02:11 Result: 13:02
```
