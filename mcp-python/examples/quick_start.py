"""
MCP SDK Quick Start Example
Simple usage example
"""

import asyncio
import logging
import os
from mcp_sdk import create_mcpsdk

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def quick_start():
    """Quick start example"""
    
    # Configure your parameters - can be overridden by environment variables
    endpoint = os.getenv('ENDPOINT', 'your-endpoint')
    access_id = os.getenv('ACCESS_ID', 'your-access-id')
    access_secret = os.getenv('ACCESS_SECRET', 'your-access-secret')
    custom_mcp_server_endpoint = os.getenv('CUSTOM_MCP_SERVER_ENDPOINT', 'http://localhost:8765/mcp')
    
    # Start MCP SDK with one line of code
    # When custom_mcp_server_endpoint is provided, SDK will automatically set up default message handler
    # This handler forwards all received requests to the specified MCP server
    async with create_mcpsdk(
        endpoint=endpoint,
        access_id=access_id,
        access_secret=access_secret,
        custom_mcp_server_endpoint=custom_mcp_server_endpoint  # SDK will automatically forward requests to this MCP server
    ) as mcpsdk:
        
        logger.info("🚀 MCP SDK started successfully!")
        logger.info(f"Connected: {mcpsdk.is_connected}")
        logger.info("📡 SDK will automatically forward requests to MCP server")
        logger.info(f"🔧 Custom MCP Server Endpoint: {custom_mcp_server_endpoint}")
        
        # Start background listening
        await mcpsdk.start_background()
        
        # Keep running
        logger.info("MCP SDK is running in background...")
        logger.info("💡 You can now send requests to the gateway endpoint")
        
        # Add your business logic here
        while True:
            await asyncio.sleep(10)
            if not mcpsdk.is_running:
                logger.warning("MCP SDK connection lost")
                break
            logger.info("MCP SDK is still running...")


if __name__ == "__main__":
    print("📖 MCP SDK - Quick Start Example")
    print("=" * 50)
    print()
    print("🔧 Configuration Options:")
    print("1. Environment Variables (recommended):")
    print("   export ENDPOINT='your-endpoint'")
    print("   export ACCESS_ID='your-access-id'")
    print("   export ACCESS_SECRET='your-access-secret'")
    print("   export CUSTOM_MCP_SERVER_ENDPOINT='http://localhost:8765/mcp'")
    print()
    print("2. Or provide parameters directly to the script")
    print()
    print("🚀 Available MCP Server Options:")
    print("• HTTP MCP Server (recommended for production):")
    print("  CUSTOM_MCP_SERVER_ENDPOINT='http://localhost:8765/mcp'")
    print("• HTTPS MCP Server (for production deployment):")
    print("  CUSTOM_MCP_SERVER_ENDPOINT='https://your-mcp-server.com/api/mcp'")
    print("• HTTP Server:")
    print("  CUSTOM_MCP_SERVER_ENDPOINT='http://localhost:3000/mcp'")
    print()
    print("▶️  Run Commands:")
    print("• Quick start with parameters:")
    print("  python examples/quick_start.py --endpoint your-endpoint --access-id your-access-id --access-secret your-access-secret")
    print("• Or use the examples launcher with parameters:")
    print("  python -m examples --endpoint your-endpoint --access-id your-access-id --access-secret your-access-secret")
    print("• Or run specific mode with parameters:")
    print("  python -m examples all --endpoint your-endpoint --access-id your-access-id --access-secret your-access-secret")
    print()
    print("⚙️  MCP SDK Features:")
    print("✓ Automatic HTTP authentication (CID & Token)")
    print("✓ WebSocket connection management")
    print("✓ Request forwarding to MCP servers")
    print("✓ Response signing and validation")
    print("✓ Built-in error handling and retry logic")
    print("✓ Background message processing")
    print()
    print("📋 HTTP MCP Server Features:")
    print("• RESTful API - Standard HTTP/HTTPS communication")
    print("• JSON-RPC 2.0 - MCP protocol over HTTP")  
    print("• Scalable deployment - Easy to deploy and scale")
    print("• Load balancing - Support multiple server instances")
    print("• HTTPS encryption - Secure communication in production")
    print()
    print("Press Ctrl+C to stop the MCP SDK")
    print("=" * 50)
    print()
    
    try:
        asyncio.run(quick_start())
    except KeyboardInterrupt:
        logger.info("👋 MCP SDK stopped by user")
