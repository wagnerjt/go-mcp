import asyncio
import click
from datetime import timedelta
from mcp import ClientSession
from mcp.client.sse import sse_client
from mcp.client.streamable_http import streamablehttp_client
import os

base_url: str = os.getenv("LITELLM_BASE_URL", "http://localhost:3000/mcp")


class MCPClient:
    """MCP Client supporting both SSE and HTTP transports."""

    def __init__(self, server_url: str, transport_type: str = "http"):
        self.server_url = server_url
        self.transport_type = transport_type
        self.session = None

    async def connect_and_test(self):
        """Connect to the MCP server and test the get_current_time tool."""
        print(
            f"🔗 Connecting to {self.server_url} using {self.transport_type.upper()} transport..."
        )

        try:
            if self.transport_type == "sse":
                await self._connect_sse()
            else:
                await self._connect_http()
        except Exception as e:
            print(f"❌ Failed to connect: {e}")
            import traceback

            traceback.print_exc()

    async def _connect_sse(self):
        """Connect using SSE transport."""
        print("📡 Opening SSE transport connection...")
        async with sse_client(
            url=self.server_url,
            timeout=60,
        ) as (read_stream, write_stream):
            await self._run_session(read_stream, write_stream)

    async def _connect_http(self):
        """Connect using HTTP transport."""
        print("📡 Opening StreamableHTTP transport connection...")
        async with streamablehttp_client(
            url=self.server_url,
            timeout=timedelta(seconds=60),
        ) as (read_stream, write_stream, get_session_id):
            await self._run_session(read_stream, write_stream, get_session_id)

    async def _run_session(self, read_stream, write_stream, get_session_id=None):
        """Run the MCP session with the given streams."""
        print("🤝 Initializing MCP session...")
        async with ClientSession(read_stream, write_stream) as session:
            self.session = session
            print("⚡ Starting session initialization...")
            await session.initialize()
            print("✅ Session initialization complete!")

            if get_session_id:
                session_id = get_session_id()
                if session_id:
                    print(f"Session ID: {session_id}")

            # List available tools
            await self._list_tools()

            # Test the get_current_time tool
            await self._test_get_current_time()

    async def _list_tools(self):
        """List available tools from the server."""
        if not self.session:
            print("❌ Not connected to server")
            return

        try:
            result = await self.session.list_tools()
            if hasattr(result, "tools") and result.tools:
                print(f"\n📋 Available tools ({len(result.tools)}):")
                for i, tool in enumerate(result.tools, 1):
                    print(f"{i}. {tool.name}")
                    if tool.description:
                        print(f"   Description: {tool.description}")
                    print()
            else:
                print("No tools available")
        except Exception as e:
            print(f"❌ Failed to list tools: {e}")

    async def _test_get_current_time(self):
        """Test the get_current_time tool with different formats."""
        if not self.session:
            print("❌ Not connected to server")
            return

        formats = ["short", "long", "iso"]

        for format_type in formats:
            try:
                print(f"\n🔧 Testing get_current_time with format: {format_type}")
                result = await self.session.call_tool(
                    "get_current_time", {"format": format_type}
                )

                if hasattr(result, "content") and result.content:
                    for content in result.content:
                        if hasattr(content, "text"):
                            print(f"   Result: {content.text}")
                        else:
                            print(f"   Result: {content}")
                else:
                    print(f"   Result: {result}")

            except Exception as e:
                print(
                    f"❌ Failed to call get_current_time with format {format_type}: {e}"
                )

        # Test echo tool
        try:
            print(f"\n🔧 Testing echo tool")
            result = await self.session.call_tool(
                "echo", {"message": "Hello from MCP client!"}
            )

            if hasattr(result, "content") and result.content:
                for content in result.content:
                    if hasattr(content, "text"):
                        print(f"   Result: {content.text}")
                    else:
                        print(f"   Result: {content}")
        except Exception as e:
            print(f"❌ Failed to call echo tool: {e}")

        # Test add_numbers tool
        try:
            print(f"\n🔧 Testing add_numbers tool")
            result = await self.session.call_tool("add_numbers", {"a": 15, "b": 27})

            if hasattr(result, "content") and result.content:
                for content in result.content:
                    if hasattr(content, "text"):
                        print(f"   Result: {content.text}")
                    else:
                        print(f"   Result: {content}")
        except Exception as e:
            print(f"❌ Failed to call add_numbers tool: {e}")


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

    # Create and run the client
    client = MCPClient(url, transport.lower())
    asyncio.run(client.connect_and_test())


if __name__ == "__main__":
    main()
