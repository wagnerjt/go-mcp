import asyncio
from langchain_mcp_adapters.tools import load_mcp_tools
from mcp import ClientSession
from mcp.client.sse import sse_client

import os

base_url: str = os.getenv("LITELLM_BASE_URL", "http://localhost:4000/mcp/")
api_key: str = os.getenv("LITELLM_API_KEY", "sk-1234")


async def main():
    # Connect to the MCP server
    async with sse_client(url=base_url) as (read, write):
        async with ClientSession(read, write) as session:
            # Initialize the session
            print("Initializing session...")
            await session.initialize()
            print("Session initialized")

            # Load available tools from MCP
            print("Loading tools...")
            tools = await load_mcp_tools(session)
            print(f"Loaded {len(tools)} tools")

            # Call the tool
            print(await session.call_tool("get_current_time", {"format": "short"}))

            # Create a ReAct agent with the model and tools
            # agent = create_react_agent(model, tools)

            # Run the agent with a user query
            # user_query = "What's the weather in Tokyo?"
            # print(f"Asking: {user_query}")
            # agent_response = await agent.ainvoke({"messages": user_query})
            # print("Agent response:")
            # print(agent_response)


if __name__ == "__main__":
    asyncio.run(main())
