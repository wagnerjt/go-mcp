import enum
import json
import asyncio
from typing import Literal, Optional, TypedDict, Dict, Any

from pydantic import BaseModel, ConfigDict
from fastapi import FastAPI, Request, Response, HTTPException, Header
from fastapi.responses import StreamingResponse
from fastapi.middleware.cors import CORSMiddleware
import uvicorn


class MCPTransport(str, enum.Enum):
    sse = "sse"
    http = "http"


class MCPSpecVersion(str, enum.Enum):
    nov_2024 = "2024-11-05"
    mar_2025 = "2025-03-26"


class MCPAuth(str, enum.Enum):
    none = "none"
    api_key = "api_key"
    bearer_token = "bearer_token"
    basic = "basic"


# MCP Literals
MCPTransportType = Literal[MCPTransport.sse, MCPTransport.http]
MCPSpecVersionType = Literal[MCPSpecVersion.nov_2024, MCPSpecVersion.mar_2025]
MCPAuthType = Optional[
    Literal[MCPAuth.none, MCPAuth.api_key, MCPAuth.bearer_token, MCPAuth.basic]
]


class MCPInfo(TypedDict, total=False):
    server_name: str
    description: Optional[str]
    logo_url: Optional[str]


class MCPServer(BaseModel):
    server_id: str
    name: str
    url: str
    # TODO: alter the types to be the Literal explicit
    transport: MCPTransportType
    spec_version: MCPSpecVersionType
    auth_type: Optional[MCPAuthType] = None
    mcp_info: Optional[MCPInfo] = None
    model_config = ConfigDict(arbitrary_types_allowed=True)


# FastAPI application
app = FastAPI(
    title="MCP Server", description="Model Context Protocol Server", version="1.0.0"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy", "message": "MCP Server is running"}


@app.post("/mcp")
async def mcp_endpoint(
    request: Request,
    accept: Optional[str] = Header(None),
    content_type: Optional[str] = Header(None, alias="content-type"),
):
    """
    MCP endpoint supporting HTTP streamable transport with SSE fallback
    """
    # Determine transport method based on Accept header
    transport_method = MCPTransport.http

    if accept and "text/event-stream" in accept.lower():
        transport_method = MCPTransport.sse

    # Get request body
    try:
        body = await request.json()
    except Exception:
        raise HTTPException(status_code=400, detail="Invalid JSON in request body")

    # Mock MCP response structure
    mcp_response = {
        "jsonrpc": "2.0",
        "id": body.get("id", 1),
        "result": {
            "capabilities": {"tools": {}, "resources": {}, "prompts": {}},
            "serverInfo": {"name": "MCP Server", "version": "1.0.0"},
            "protocolVersion": "2024-11-05",
        },
    }

    if transport_method == MCPTransport.sse:
        # SSE Transport
        async def generate_sse():
            yield f"data: {json.dumps(mcp_response)}\n\n"

        return StreamingResponse(
            generate_sse(),
            media_type="text/event-stream",
            headers={
                "Cache-Control": "no-cache",
                "Connection": "keep-alive",
                "Access-Control-Allow-Origin": "*",
            },
        )
    else:
        # HTTP Transport (default)
        return Response(
            content=json.dumps(mcp_response),
            media_type="application/json",
            headers={
                "Cache-Control": "no-cache",
                "Access-Control-Allow-Origin": "*",
            },
        )


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=3000, log_level="info")
