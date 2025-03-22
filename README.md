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
uv pip install -r bridge/requirements.txt

# cd into each and install dependencies
cd client;go mod download; cd ..
cd server; go mod download; cd ..
```

### Setup litellm proxy

Or via docker

```sh
cd bridge
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
