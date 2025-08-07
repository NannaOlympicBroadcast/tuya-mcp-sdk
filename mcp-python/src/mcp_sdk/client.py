"""
MCP SDK Client - Main client class
"""

import logging
from typing import Optional, Dict, Any
import time

from .models import AuthConfig, MCPSdkRequest, MCPSdkResponse, TokenData
from .auth import AuthManager
from .websocket_adapter import WebSocketAdapter
from .mcp_client import MCPClientManager
from .heartbeat import HeartbeatManager
from .exceptions import MCPSdkError, ConnectionError

logger = logging.getLogger(__name__)


class MCPSdkClient:
    """MCP SDK client for connecting to MCP SDK services"""
    
    def __init__(
        self,
        endpoint: str,
        access_id: str,
        access_secret: str,
        custom_mcp_server_endpoint: Optional[str] = None,
        heartbeat_interval: int = 30
    ):
        """
        Initialize MCP SDK client
        
        Args:
            endpoint: SDK domain
            access_id: Developer access ID
            access_secret: access secret
            custom_mcp_server_endpoint: Custom MCP server endpoint
            heartbeat_interval: Heartbeat interval (seconds)
        """
        self.config = AuthConfig(
            endpoint=endpoint,
            access_id=access_id,
            access_secret=access_secret,
        )
        
        # Initialize components
        self.auth_manager = AuthManager(self.config)
        self.websocket_adapter = WebSocketAdapter(
            endpoint=endpoint,
            access_id=access_id,
            access_secret=access_secret,
            message_handler=self._handle_sdk_request,
            token_provider=self._get_token_for_reconnect
        )
        
        self.mcp_client_manager: Optional[MCPClientManager] = None
        if custom_mcp_server_endpoint:
            self.mcp_client_manager = MCPClientManager(custom_mcp_server_endpoint)
        
        # Use WebSocket ping/pong based heartbeat manager
        self.heartbeat_manager = HeartbeatManager(
            ping_interval=heartbeat_interval,
            ping_timeout=10  # Default 10 seconds timeout
        )
        
        self._connected = False
        self._running = False
    
    async def __aenter__(self):
        """Async context manager entry"""
        await self.connect()
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit"""
        await self.disconnect()
    
    async def connect(self):
        """Connect to SDK"""
        try:
            logger.info("Connecting to MCP SDK...")
            
            # 1. Authenticate and get token
            async with self.auth_manager:
                token_data = await self.auth_manager.get_token()
            
            # 2. Connect to MCP server (if configured)
            if self.mcp_client_manager:
                await self.mcp_client_manager.connect()
            
            # 3. Establish WebSocket connection
            await self.websocket_adapter.connect(token_data)
            
            # 4. Set WebSocket reference for heartbeat monitoring and start heartbeat
            self.websocket_adapter.set_heartbeat_manager(self.heartbeat_manager)
            self.heartbeat_manager.set_websocket(self.websocket_adapter._websocket)
            await self.heartbeat_manager.start()
            
            self._connected = True
            logger.info("MCP SDK connection successful")
            
        except Exception as e:
            await self.disconnect()
            raise MCPSdkError(f"Failed to connect to SDK: {e}")
    
    async def disconnect(self):
        """Disconnect"""
        try:
            self._connected = False
            self._running = False
            
            # Stop heartbeat
            await self.heartbeat_manager.stop()
            
            # Close WebSocket connection
            await self.websocket_adapter.close()
            
            # Disconnect from MCP server
            if self.mcp_client_manager:
                await self.mcp_client_manager.disconnect()
            
            logger.info("MCP SDK connection disconnected")
            
        except Exception as e:
            logger.error(f"Error occurred while disconnecting: {e}")
    
    async def start_listening(self):
        """Start listening for messages"""
        if not self._connected:
            raise ConnectionError("Client not connected")
        
        self._running = True
        logger.info("Started listening for SDK messages")
        
        try:
            await self.websocket_adapter.start_listening()
        except Exception as e:
            logger.error(f"Error occurred while listening for messages: {e}")
            self._running = False
            raise
    
    async def send_request(self, request: Dict[str, Any]) -> Dict[str, Any]:
        """
        Send MCP request
        
        Args:
            request: MCP request data
            
        Returns:
            MCP response data
            
        Raises:
            MCPSdkError: Request sending failed
        """
        if not self._connected:
            raise ConnectionError("Client not connected")
        
        if not self.mcp_client_manager:
            raise MCPSdkError("MCP client not configured")
        
        try:
            # Build SDK request
            sdk_request = MCPSdkRequest(
                request_id=f"req_{int(time.time() * 1000)}",
                request=request
            )
            
            # Send request through MCP client
            response = await self.mcp_client_manager.send_request(sdk_request)
            
            return response.response
            
        except Exception as e:
            raise MCPSdkError(f"Failed to send request: {e}")
    
    async def _handle_sdk_request(self, request: MCPSdkRequest) -> MCPSdkResponse:
        """
        Handle SDK request
        
        Args:
            request: SDK request
            
        Returns:
            SDK response
        """
        logger.debug(f"Handling SDK request: {request.request_id}")
        
        try:
            if not self.mcp_client_manager:
                import json
                error_string = json.dumps({"error": "MCP client not configured"}, separators=(',', ':'), ensure_ascii=False)
                return MCPSdkResponse(
                    request_id=request.request_id,
                    endpoint=request.endpoint,
                    version=request.version,
                    method=request.method,
                    ts=str(int(time.time() * 1000)),
                    response=error_string
                )
            
            # Process request through MCP client
            response = await self.mcp_client_manager.send_request(request)
            return response
            
        except Exception as e:
            logger.error(f"Failed to handle SDK request: {e}")
            import json
            import time
            error_string = json.dumps({"error": str(e)}, separators=(',', ':'), ensure_ascii=False)
            return MCPSdkResponse(
                request_id=request.request_id,
                endpoint=request.endpoint,
                version=request.version,
                method=request.method,
                ts=str(int(time.time() * 1000)),
                response=error_string
            )
    
    def set_mcp_server(self, mcp_server_endpoint: str):
        """Set MCP server URI"""
        self.mcp_client_manager = MCPClientManager(mcp_server_endpoint)
    
    async def _get_token_for_reconnect(self) -> Optional[TokenData]:
        """Get token for reconnection (force refresh)"""
        try:
            logger.info("Starting token acquisition for reconnection...")
            async with self.auth_manager:
                logger.info("AuthManager session created, requesting token...")
                token_data = await self.auth_manager.get_token(force_refresh=True)
                if token_data:
                    logger.info(f"Token acquired successfully for reconnection, client_id: {token_data.client_id}")
                else:
                    logger.error("Token acquisition returned None")
                return token_data
        except Exception as e:
            logger.error(f"Failed to get token for reconnection: {e}", exc_info=True)
            return None
    
    @property
    def is_connected(self) -> bool:
        """Check if connected"""
        return self._connected
    
    @property
    def is_running(self) -> bool:
        """Check if running"""
        return self._running