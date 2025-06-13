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


async def handle_sse(scope: Scope, receive: Receive, send: Send) -> None:
    """Handle MCP requests through SSE."""
    await sse_session_manager.handle_request(scope, receive, send)


@contextlib.asynccontextmanager
async def lifespan(app: Starlette) -> AsyncIterator[None]:
    """Application lifespan context manager."""
    async with session_manager.run():
        async with sse_session_manager.run():
            logger.info(
                "MCP Server started with StreamableHTTP and SSE session managers!"
            )
            try:
                yield
            finally:
                logger.info("MCP Server shutting down...")


# Create session managers
session_manager = StreamableHTTPSessionManager(
    app=mcp_server,
    event_store=None,
    json_response=True,  # Use JSON responses instead of SSE by default
    stateless=True,
)

# Create SSE session manager
sse_session_manager = StreamableHTTPSessionManager(
    app=mcp_server,
    event_store=None,
    json_response=False,  # Use SSE responses for this endpoint
    stateless=True,
)


# Create Starlette application
app = Starlette(
    debug=True,
    routes=[
        Route("/health", health_check, methods=["GET"]),
        # works in python's http transport and /mcp and /sse
        # works in and go's sse transport for /mcp and /sse
        Mount("/mcp", app=handle_mcp),
        Mount("/sse", app=handle_sse),
        # works in go for both
        # Route("/mcp", endpoint=handle_mcp, methods=["GET"]),
        # Route("/sse", endpoint=handle_sse, methods=["GET"]),
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
