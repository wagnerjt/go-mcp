import asyncio
from mcp import ClientSession
from mcp.types import Tool
from mcp.client.sse import sse_client

from litellm import experimental_mcp_client

import os

base_url: str = os.getenv("LITELLM_BASE_URL", "http://localhost:8080")
api_key: str = os.getenv("LITELLM_API_KEY", "sk-1234")


async def client_execute():
    async with sse_client(base_url + "/sse", sse_read_timeout=60) as streams:
        async with ClientSession(*streams) as session:
            await session.initialize()

            # get tools mcp
            tools: list[Tool] = await experimental_mcp_client.load_mcp_tools(
                session=session, format="mcp"
            )
            print(tools)

            # get tools openai
            tools: list[Tool] = await experimental_mcp_client.load_mcp_tools(
                session=session, format="openai"
            )
            print(tools)

            # if tools successfully loaded
            if tools and len(tools) > 0:
                # test mcp sdk for add tool
                result = await session.call_tool("add", {"a": 1.5, "b": 50.5})
                print(result)

            # test litellm sdk for call tool
            # for tool in tools:
            #     # test the call tools result
            #     tool_call_result = await experimental_mcp_client.(
            #         session=session,
            #         openai_tool=tool,
            #     )
            #     # experimental_mcp_client.call_openai_tool
            #     print(tool_call_result)


def main():
    asyncio.run(client_execute())


if __name__ == "__main__":
    main()
