"""
Exception Definitions Module
"""


class MCPSdkError(Exception):
    """MCP SDK base exception class"""
    pass


class AuthenticationError(MCPSdkError):
    """Authentication error"""
    pass


class ConnectionError(MCPSdkError):
    """Connection error"""
    pass


class SignatureError(MCPSdkError):
    """Signature verification error"""
    pass


class HeartbeatError(MCPSdkError):
    """Heartbeat error"""
    pass


class MCPClientError(MCPSdkError):
    """MCP client error"""
    pass
