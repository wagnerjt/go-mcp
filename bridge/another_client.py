from typing import List, Optional
import base64
from datetime import timedelta
from mcp import ClientSession
from mcp.types import Tool as MCPTool
from mcp.types import CallToolResult as MCPCallToolResult
from mcp.types import CallToolRequestParams as MCPCallToolRequestParams
from mcp.client.sse import sse_client
from mcp.client.streamable_http import streamablehttp_client

from mcp_types import MCPAuth, MCPTransport, MCPTransportType, MCPAuthType

import asyncio
import click

# default dummy token
token = "sk-1234"


def to_basic_auth(auth_value: str) -> str:
    """Convert auth value to Basic Auth format."""
    return base64.b64encode(auth_value.encode("utf-8")).decode()


class MCPClient:
    """
    MCP Client supporting:
      SSE and HTTP transports
      Authentication via Bearer token, Basic Auth, or API Key
      Tool calling with error handling and result parsing
    """

    def __init__(
        self,
        server_url: str,
        transport_type: MCPTransportType = MCPTransport.http,
        auth_type: MCPAuthType = None,
        auth_value: Optional[str] = None,
        timeout: float = 60.0,
    ):
        self.server_url: str = server_url
        self.transport_type: MCPTransport = transport_type
        self.auth_type: MCPAuthType = auth_type
        self.timeout: float = timeout
        self._mcp_auth_value: Optional[str] = None
        self._session: Optional[ClientSession] = None
        self._context = None
        self._transport_ctx = None
        self._transport = None
        self._session_ctx = None

        # handle the basic auth value if provided
        if auth_value:
            self.update_auth_value(auth_value)

    async def __aenter__(self):
        """
        Enable async context manager support.
          Initializes the transport and session.
        """
        headers = self._get_auth_headers()

        if self.transport_type == MCPTransport.sse:
            self._transport_ctx = sse_client(
                url=self.server_url,
                timeout=self.timeout,
                headers=headers,
            )
            self._transport = await self._transport_ctx.__aenter__()
            self._session_ctx = ClientSession(self._transport[0], self._transport[1])
            self._session = await self._session_ctx.__aenter__()
            await self._session.initialize()
        else:
            self._transport_ctx = streamablehttp_client(
                url=self.server_url,
                timeout=timedelta(seconds=self.timeout),
                headers=headers,
            )
            self._transport = await self._transport_ctx.__aenter__()
            self._session_ctx = ClientSession(self._transport[0], self._transport[1])
            self._session = await self._session_ctx.__aenter__()
            await self._session.initialize()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Cleanup when exiting context manager."""
        if self._session:
            await self._session_ctx.__aexit__(exc_type, exc_val, exc_tb)
        if self._transport_ctx:
            await self._transport_ctx.__aexit__(exc_type, exc_val, exc_tb)

    async def disconnect(self):
        """Clean up session and connections."""
        if self._session:
            try:
                await self._session.close()  # Ensure session is properly closed
            except Exception:
                pass
            self._session = None

        if self._context:
            try:
                await self._context.__aexit__(None, None, None)
            except Exception:
                pass
            self._context = None

    def update_auth_value(self, mcp_auth_value: str):
        """
        Set the authentication header for the MCP client.
        """
        if self.auth_type == MCPAuth.basic:
            # Assuming mcp_auth_value is in format "username:password", convert it when updating
            mcp_auth_value = to_basic_auth(mcp_auth_value)
        self._mcp_auth_value = mcp_auth_value

    def _get_auth_headers(self) -> dict:
        """Generate authentication headers based on auth type."""
        if not self._mcp_auth_value:
            return {}

        if self.auth_type == MCPAuth.bearer_token:
            return {"Authorization": f"Bearer {self._mcp_auth_value}"}
        elif self.auth_type == MCPAuth.basic:
            return {"Authorization": f"Basic {self._mcp_auth_value}"}
        elif self.auth_type == MCPAuth.api_key:
            return {"X-API-Key": self._mcp_auth_value}
        return {}

    async def list_tools(self) -> List[MCPTool]:
        """List available tools from the server."""
        if not self._session:
            await self.connect()

        result = await self._session.list_tools()
        if hasattr(result, "tools") and result.tools:
            return result.tools
        return result

    async def call_tool(
        self, call_tool_request_params: MCPCallToolRequestParams
    ) -> MCPCallToolResult:
        """
        Call an MCP Tool.
        """
        if not self._session:
            await self.connect()

        tool_result = await self._session.call_tool(
            name=call_tool_request_params.name,
            arguments=call_tool_request_params.arguments,
        )
        return tool_result


async def test(
    url: str,
    transport: MCPTransportType,
    auth_type: MCPAuthType,
    auth_value: Optional[str] = None,
):
    """Test the MCP client connection and tool listing."""
    async with MCPClient(
        url, transport_type=transport, auth_value=token, timeout=10
    ) as client:
        tools = await client.list_tools()
        print("Available tools:")
        if hasattr(tools, "tools"):
            for tool in tools.tools:
                print(f" - {tool.name}: {tool.description}")
        else:
            for tool in tools:
                print(f" - {tool.name}: {tool.description}")


@click.command()
@click.option(
    "--transport",
    "-t",
    type=click.Choice(["http", "sse"], case_sensitive=False),
    default="http",
    help="Transport type to use (http or sse)",
)
@click.option(
    "--url",
    "-u",
    default="http://localhost:3000/mcp",
    help="MCP server URL",
)
@click.option(
    "--verbose",
    "-v",
    is_flag=True,
    help="Enable verbose output",
)
def main(transport: str, url: str, verbose: bool):
    """MCP Client supporting both SSE and HTTP transports."""
    if verbose:
        print(f"🚀 Starting MCP Client")
        print(f"   Transport: {transport.upper()}")
        print(f"   Server URL: {url}")
        print()
    # client = SimpleClient("http://localhost:8080/mcp", "http")
    # asyncio.run(client.connect())

    asyncio.run(test(url, transport.lower(), MCPAuth.bearer_token, token))


if __name__ == "__main__":
    main()
