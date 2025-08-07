# Custom MCP SDK

## 1. Overview

### 1.1. SDK Usage

- Init SDK Config  
> Before initialization, you need to obtain AccessId, AccessSecret, and Endpoint from the developer platform.
```python
from mcp_sdk import create_mcpsdk

# Create MCP SDK with configuration
mcpsdk = await create_mcpsdk(
    # Set custom MCP server endpoint
    custom_mcp_server_endpoint="http://localhost:8765/mcp",
    # Set Access ID, Access secret and Endpoint
    endpoint="your-gateway-endpoint",
    access_id="your-access-id", 
    access_secret="your-access-secret"
)
```

- Running SDK
```python
import asyncio

async def main():
    async with mcpsdk:
        print("🚀 MCP SDK connected successfully!")
        await mcpsdk.start_background()
        
        # Keep running
        while mcpsdk.is_running:
            await asyncio.sleep(10)

if __name__ == "__main__":
    asyncio.run(main())
```

### 1.2. Architecture Diagram
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Your App      │───▶│    MCP SDK       │───▶│  MCP Gateway    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │   MCP Server     │
                       └──────────────────┘
```

## 2. Quick Start 
**Prerequisites:**
> - The machine has Python 3.10+ environment for Python development.
> - Install imagesnap (optional, required for demonstrating computer photography control)

- Checkout source code
    ```shell
    git clone https://github.com/tuya/tuya-mcp-sdk.git

    cd mcp-python
    ```
  
- Run SDK Example
    ```shell
    # Create virtual environment
    python -m venv .venv
    source .venv/bin/activate  # Linux/Mac

    # Install development dependencies
    pip install -e ".[dev]"

    # Run with custom parameters
    python -m examples all --endpoint your-endpoint --access-id your-access-id --access-secret your-access-secret --custom_mcp_server_endpoint 'http://localhost:8765/mcp'
    ```
  - Parameter Description
    > - endpoint: Tuya developer platform MCP endpoint
    > - access_id: Tuya developer platform MCP access id
    > - access_secret: Tuya developer platform MCP access secret
    > - custom_mcp_server_endpoint: Address of custom MCP Server Endpoint declared in SDK; current demo includes an MCP Server example `http://localhost:8765/mcp`

## 3. Develop Custom MCP Server
> Developers develop custom MCP Servers based on business to provide capabilities for their devices.

### 3.1. Develop By [FastMCP](https://github.com/jlowin/fastmcp)

- Create a New MCP Server
```python
from fastmcp import FastMCP

mcp_server = FastMCP("Demo 🚀")
```

- Create a Tool
```python
@mcp_server.tool()
def hello_world(name: str) -> str:
    """Say hello to someone
    
    Args:
        name: Name of the person to greet
    """
    return f"Hello, {name}!"
```

### 3.2. Run MCP Server
```python
from fastmcp import FastMCP

def main():
    # Create a new MCP server
    mcp_server = FastMCP("Demo 🚀")

    # Add tool
    @mcp_server.tool()
    def hello_world(name: str) -> str:
        """Say hello to someone
        
        Args:
            name: Name of the person to greet
        """
        return f"Hello, {name}!"

    # Start the server
    mcp_server.run()

if __name__ == "__main__":
    main()
```

### 3.3. Examples

- [Examples](examples/README.md)
