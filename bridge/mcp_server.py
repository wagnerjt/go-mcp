import enum
import json
import asyncio
from typing import Literal, Optional, TypedDict, Dict, Any

from pydantic import BaseModel, ConfigDict
import contextlib
import logging
from collections.abc import AsyncIterator

import mcp.types as types
from mcp.server.lowlevel import Server
from mcp.server.streamable_http_manager import StreamableHTTPSessionManager
from starlette.applications import Starlette
from starlette.routing import Mount, Route
from starlette.types import Receive, Scope, Send
from starlette.requests import Request
from starlette.responses import JSONResponse
import uvicorn

logger = logging.getLogger(__name__)


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


# Create MCP server instance
mcp_server = Server("mcp-fastapi-server", "1.0.0")


@mcp_server.list_tools()
async def list_tools() -> list[types.Tool]:
    """List available tools."""
    return [
        types.Tool(
            name="echo",
            description="Echo back the provided message",
            inputSchema={
                "type": "object",
                "properties": {
                    "message": {
                        "type": "string",
                        "description": "Message to echo back",
                    }
                },
                "required": ["message"],
            },
        ),
        types.Tool(
            name="get_current_time",
            description="Get the current time",
            inputSchema={
                "type": "object",
                "properties": {
                    "format": {
                        "type": "string",
                        "description": "Format for the time (short, long, iso)",
                        "enum": ["short", "long", "iso"],
                        "default": "short",
                    }
                },
            },
        ),
        types.Tool(
            name="add_numbers",
            description="Add two numbers together",
            inputSchema={
                "type": "object",
                "properties": {
                    "a": {
                        "type": "number",
                        "description": "First number",
                    },
                    "b": {
                        "type": "number",
                        "description": "Second number",
                    },
                },
                "required": ["a", "b"],
            },
        ),
    ]


@mcp_server.call_tool()
async def call_tool(
    name: str, arguments: dict
) -> list[types.TextContent | types.ImageContent | types.EmbeddedResource]:
    """Handle tool calls."""

    if name == "echo":
        message = arguments.get("message", "")
        return [
            types.TextContent(
                type="text",
                text=f"Echo: {message}",
            )
        ]

    elif name == "get_current_time":
        from datetime import datetime

        format_type = arguments.get("format", "short")
        current_time = datetime.now()

        if format_type == "short":
            time_str = current_time.strftime("%H:%M")
        elif format_type == "long":
            time_str = current_time.strftime("%Y-%m-%d %H:%M:%S")
        elif format_type == "iso":
            time_str = current_time.isoformat()
        else:
            time_str = current_time.strftime("%H:%M")

        return [
            types.TextContent(
                type="text",
                text=time_str,
            )
        ]

    elif name == "add_numbers":
        a = arguments.get("a", 0)
        b = arguments.get("b", 0)
        result = a + b

        return [
            types.TextContent(
                type="text",
                text=f"The sum of {a} and {b} is {result}",
            )
        ]

    else:
        raise ValueError(f"Unknown tool: {name}")


async def health_check(request: Request) -> JSONResponse:
    """Health check endpoint."""
    return JSONResponse({"status": "healthy", "message": "MCP Server is running"})


async def handle_mcp(scope: Scope, receive: Receive, send: Send) -> None:
    """Handle MCP requests through StreamableHTTP."""
    await session_manager.handle_request(scope, receive, send)


@contextlib.asynccontextmanager
async def lifespan(app: Starlette) -> AsyncIterator[None]:
    """Application lifespan context manager."""
    async with session_manager.run():
        logger.info("MCP Server started with StreamableHTTP session manager!")
        try:
            yield
        finally:
            logger.info("MCP Server shutting down...")


# Create session manager
session_manager = StreamableHTTPSessionManager(
    app=mcp_server,
    event_store=None,
    json_response=True,  # Use JSON responses instead of SSE by default
    stateless=True,
)


# Create Starlette application
app = Starlette(
    debug=True,
    routes=[
        Route("/health", health_check, methods=["GET"]),
        Mount("/mcp", app=handle_mcp),
    ],
    lifespan=lifespan,
)


if __name__ == "__main__":
    # Configure logging
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    )

    # Run the server
    uvicorn.run(app, host="0.0.0.0", port=3000, log_level="info")
