"""
MCP SDK

A Python SDK for connecting to MCP SDK with WebSocket support.
"""

from .client import MCPSdkClient
from .mcp_sdk import (
    MCPSdk,
    MCPSdkConfig,
    MCPClientConfig,
    create_mcpsdk,
    run_mcpsdk,
)
from .exceptions import (
    MCPSdkError,
    AuthenticationError,
    ConnectionError,
    SignatureError,
)
from .models import (
    MCPSdkRequest,
    MCPSdkResponse,
    AuthConfig,
)

__version__ = "0.1.0"

__all__ = [
    # Main interfaces
    "MCPSdk",
    "MCPSdkConfig",
    "MCPClientConfig",
    "create_mcpsdk",
    "run_mcpsdk",
    # Low-level client
    "MCPSdkClient",
    # Exception classes
    "MCPSdkError",
    "AuthenticationError",
    "ConnectionError",
    "SignatureError",
    # Data models
    "MCPSdkRequest",
    "MCPSdkResponse",
    "AuthConfig",
]
