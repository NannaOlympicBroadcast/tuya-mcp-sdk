"""
WebSocket Adapter for MCP SDK
"""

import asyncio
import json
import logging
from typing import Optional, Callable, Dict, Any, Union, Awaitable
import time
import websockets
from websockets.exceptions import ConnectionClosed, WebSocketException
from websockets.asyncio.client import ClientConnection
from urllib.parse import quote
from pydantic import ValidationError

from .models import MCPSdkRequest, MCPSdkResponse, TokenData
from .signature import SignatureUtils
from .exceptions import ConnectionError

logger = logging.getLogger(__name__)


class WebSocketAdapter:
    """WebSocket adapter for connecting to MCP SDK"""
    
    def __init__(
        self,
        endpoint: str,
        access_id: str,
        access_secret: str,
        message_handler: Optional[Callable[[MCPSdkRequest], Union[MCPSdkResponse, Awaitable[MCPSdkResponse], None]]] = None,
        token_provider: Optional[Callable[[], Union[TokenData, Awaitable[TokenData], None]]] = None
    ):
        self.endpoint = endpoint
        self.access_id = access_id
        self.access_secret = access_secret
        self.message_handler = message_handler
        self.token_provider = token_provider  # Token provider function
        
        self._websocket: Optional[ClientConnection] = None
        self._token_data: Optional[TokenData] = None
        self._running = False
        self._base_reconnect_interval = 1  # Base reconnection interval (seconds)
        self._max_reconnect_interval = 120  # Maximum reconnection interval (seconds)
        self._current_reconnect_interval = 1  # Current reconnection interval
        self._reconnect_attempts = 0
        self._stop_event = asyncio.Event()
        self._heartbeat_manager = None  # Heartbeat manager reference
        self._is_reconnecting = False  # Flag to indicate if reconnecting
    
    def set_heartbeat_manager(self, heartbeat_manager):
        """Set heartbeat manager reference"""
        self._heartbeat_manager = heartbeat_manager
    
    async def connect(self, token_data: TokenData):
        """
        Establish WebSocket connection
        
        Args:
            token_data: Authentication token data
            
        Raises:
            ConnectionError: Raised when connection fails
        """
        self._token_data = token_data
        
        try:
            # Build WebSocket URL with cid as query parameter
            ws_url = f"{self.endpoint.rstrip('/')}/ws/mcp"
            if ws_url.startswith('http'):
                ws_url = ws_url.replace('http', 'ws', 1)
            elif not ws_url.startswith('ws'):
                ws_url = f"ws://{ws_url}"  # Use ws:// for local testing
            
            # Add cid as query parameter
            ws_url = f"{ws_url}?client_id={quote(self._token_data.client_id)}"
            
            # Create connection headers (without body)
            headers = self._create_connection_headers()
            
            logger.info(f"Connecting to WebSocket: {ws_url}")
            
            # Establish WebSocket connection
            self._websocket = await websockets.connect(
                ws_url, 
                additional_headers=headers,
                ping_interval=30,
                ping_timeout=10
            )
            
            # No need to send separate auth message since cid is in URL
            
            # Reset reconnection parameters on successful connection
            self._reconnect_attempts = 0
            self._current_reconnect_interval = self._base_reconnect_interval
            logger.info("WebSocket connection established successfully")
            
        except Exception as e:
            raise ConnectionError(f"WebSocket connection failed: {e}")
    
    def _create_connection_headers(self) -> Dict[str, str]:
        """Create connection headers"""
        if not self._token_data:
            raise ConnectionError("Token data not set")
        
        # Create WebSocket connection headers with cid as query parameter
        return SignatureUtils.create_websocket_headers(
            access_id=self.access_id,
            access_secret=self._token_data.token,
            client_id=self._token_data.client_id,
            path="/ws/mcp"
        )
    
    async def start_listening(self):
        """Start listening for messages"""
        # Remove initial WebSocket check, let reconnection logic handle it
        self._running = True
        self._stop_event.clear()
        logger.info("Started listening for WebSocket messages")
        
        while self._running:
            try:
                if not self._websocket:
                    if not self._is_reconnecting:
                        logger.warning("WebSocket connection lost, waiting for reconnection...")
                        # Wait for reconnection to complete or timeout
                        waited_time = 0
                        while waited_time < 30 and self._running:  # Wait up to 30 seconds
                            if self._websocket:
                                break
                            await asyncio.sleep(1)
                            waited_time += 1
                        
                        if not self._websocket and self._running:
                            logger.warning("Reconnection timeout, attempting manual reconnect")
                            await self._attempt_reconnect()
                            # Continue waiting for reconnection result
                            continue
                    else:
                        # Currently reconnecting, wait for reconnection to complete
                        await asyncio.sleep(1)
                    continue
                
                async for message in self._websocket:
                    if self._stop_event.is_set():
                        break
                    
                    try:
                        await self._handle_message(message)
                    except Exception as e:
                        logger.error(f"Error handling message: {e}")
                        
            except ConnectionClosed:
                logger.warning("WebSocket connection closed")
                if self._running and not self._is_reconnecting:
                    await self._attempt_reconnect()
            except WebSocketException as e:
                logger.error(f"WebSocket error: {e}")
                if self._running and not self._is_reconnecting:
                    await self._attempt_reconnect()
            except Exception as e:
                logger.error(f"Unexpected error in message listening: {e}")
                if self._running and not self._is_reconnecting:
                    await self._attempt_reconnect()
            
            # If still running, wait a bit before retrying to listen
            if self._running and not self._stop_event.is_set():
                await asyncio.sleep(0.1)
    
    async def _handle_message(self, message: str):
        """Handle received message"""
        try:
            data = json.loads(message)
            
            # Notify heartbeat manager that message received (indicating connection is normal)
            if self._heartbeat_manager:
                self._heartbeat_manager.on_websocket_message()
            
            # Check if it's a sys/error method, if so just log and don't process further
            if "method" in data and data.get("method") == "sys/error":
                logger.warning(f"Received sys/error message: {data}")
                return
            
            # Perform signature verification
            if not self._verify_message_signature(data):
                logger.error(f"Message signature verification failed: {data}")
                return
            
            # Check if it's a new format SDK request (containing request_id and request fields)
            if "request_id" in data and "request" in data:
                # New format: request_id, request are strings
                try:
                    # Convert new format to format expected by MCPSdkRequest
                    converted_data = {
                        "request_id": data["request_id"],  # Keep request_id as-is
                        "endpoint": data.get("endpoint", ""),
                        "version": data.get("version", "1.0"),
                        "method": data.get("method", ""),
                        "ts": data.get("ts", str(int(time.time() * 1000))),  # Use millisecond timestamp
                        "request": data["request"] # JSON string kept as-is, let MCPSdkRequest handle it
                    }
                    
                    # Parse as SDK request
                    request = MCPSdkRequest(**converted_data)
                    logger.debug(f"Received new format request: {request.request_id}")
                    
                    # Call message handler
                    if self.message_handler:
                        response = await self._call_message_handler(request)
                        if response:
                            # Pass original request data for use in response
                            await self._send_response(response, original_request_data=data)
                        else:
                            logger.debug(f"No response required for request: {request.request_id}")
                    else:
                        logger.warning("No message handler set, ignoring request")
                    return
                    
                except (json.JSONDecodeError, KeyError) as e:
                    logger.error(f"Failed to parse new format request: {e}")
                    return
            
            else:
                logger.debug(f"Ignoring non-request message: {data}")
                return
                
        except json.JSONDecodeError as e:
            logger.error(f"JSON parsing failed: {e}")
        except ValidationError as e:
            logger.error(f"Message validation failed: {e}")
        except Exception as e:
            logger.error(f"Message handling failed: {e}")
    
    def _verify_message_signature(self, data: Dict[str, Any]) -> bool:
        """Verify message signature"""
        try:
            # Check if signature field is present
            if "sign" not in data:
                logger.warning("Message missing signature field")
                return False
            
            # Use token to verify signature
            if not self._token_data:
                logger.error("Token data not available for signature verification")
                return False
            
            # Verify signature (pass complete data including signature)
            is_valid = SignatureUtils.verify_message_signature(
                access_secret=self._token_data.token,
                message_data=data
            )
            
            if not is_valid:
                logger.error("Message signature verification failed")
                return False
            
            logger.debug("Message signature verification successful")
            return True
            
        except Exception as e:
            logger.error(f"Error during signature verification: {e}")
            return False
    
    async def _call_message_handler(self, request: MCPSdkRequest) -> Optional[MCPSdkResponse]:
        """Call message handler"""
        try:
            # Check for special methods that require special handling
            method = request.method
            
            if method == "root/migrate":
                logger.info("Received root/migrate request, initiating reconnection")
                # Schedule reconnection in background
                asyncio.create_task(self._handle_migrate_reconnect())
                # No response returned
                return None
            
            elif method == "root/kickout":
                logger.warning("Received root/kickout request, terminating connection")
                # Schedule connection termination in background
                asyncio.create_task(self._handle_kickout())
                # No response returned
                return None
            
            # Handle normal requests
            if asyncio.iscoroutinefunction(self.message_handler):
                return await self.message_handler(request)
            else:
                return self.message_handler(request)
        except Exception as e:
            logger.error(f"Message handler execution failed: {e}")
            # Return error response as JSON string
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
    
    async def _send_response(self, response: MCPSdkResponse, original_request_data: Optional[Dict[str, Any]] = None):
        """Send response"""
        try:
            response_dict = response.model_dump()
            
            # Only use new format response (request_id, response as string)
            new_format_response = {
                "request_id": original_request_data["request_id"],
                "endpoint": original_request_data.get("endpoint", ""),
                "version": original_request_data.get("version", "1.0.0"),
                "method": original_request_data.get("method", ""),
                "ts":str(int(time.time() * 1000)),
                "response": response_dict["response"]  # response is already a string, use directly
            }
            
            # Sign response and add sign field
            signed_dict = SignatureUtils.sign_message(self._token_data.token, new_format_response)
            new_format_response["sign"] = signed_dict["sign"]
            
            message = json.dumps(new_format_response, ensure_ascii=False)
            
            # Send response through WebSocket using binary mode
            if self._websocket:
                # Convert string to bytes for binary transmission
                # message_bytes = message.encode('utf-8')
                # await self._websocket.send_data(message_bytes)
                await self._websocket.send(message)
                logger.info(f"Response sent: {response.request_id}")
            else:
                logger.error("WebSocket connection not available")
            
        except Exception as e:
            logger.error(f"Failed to send response: {e}")
    
    async def _handle_migrate_reconnect(self):
        """Handle migrate reconnection request"""
        try:
            logger.info("Handling migrate reconnection...")
            
            # Set reconnection flag to prevent simultaneous reconnection from other places
            self._is_reconnecting = True
            
            # Close current connection
            if self._websocket:
                await self._websocket.close()
                self._websocket = None
            
            # Reset reconnection interval for immediate reconnection after migrate
            self._current_reconnect_interval = self._base_reconnect_interval
            
            # Wait a moment before reconnecting
            await asyncio.sleep(1)
            
            # Attempt reconnection
            await self._attempt_reconnect()
            
        except Exception as e:
            logger.error(f"Error during migrate reconnection: {e}")
        finally:
            # Ensure reconnection flag is cleared
            self._is_reconnecting = False
    
    async def _handle_kickout(self):
        """Handle kickout request - terminate connection and mark as offline"""
        try:
            logger.warning("Handling kickout request - terminating connection")
            # Mark as not running to prevent reconnection attempts
            self._running = False
            self._stop_event.set()
            
            # Close WebSocket connection
            if self._websocket:
                await self._websocket.close()
                self._websocket = None
            
            # Notify heartbeat manager that service is offline
            if self._heartbeat_manager:
                self._heartbeat_manager.mark_service_offline()
            
            logger.info("Service marked as offline due to kickout")
            
        except Exception as e:
            logger.error(f"Error during kickout handling: {e}")
    
    async def _attempt_reconnect(self):
        """Attempt reconnection with exponential backoff strategy"""
        # Set reconnection flag
        self._is_reconnecting = True
        
        try:
            self._reconnect_attempts += 1
            logger.info(f"Attempting reconnection #{self._reconnect_attempts} (interval: {self._current_reconnect_interval}s)")
            
            # Wait for current reconnection interval
            await asyncio.sleep(self._current_reconnect_interval)
            
            try:
                # Re-acquire token_data (token may have expired)
                if self.token_provider:
                    logger.info("Re-acquiring token for reconnection...")
                    try:
                        if asyncio.iscoroutinefunction(self.token_provider):
                            new_token_data = await self.token_provider()
                        else:
                            new_token_data = self.token_provider()
                        
                        if new_token_data:
                            logger.info(f"Got new token for reconnection, client_id: {new_token_data.client_id}")
                            await self.connect(new_token_data)
                            logger.info("Reconnection successful with new token")
                            # Reconnection successful, reset parameters handled in connect()
                            # Re-setup heartbeat manager
                            await self._restore_heartbeat_manager()
                            return  # Reconnection successful, exit reconnection loop
                        else:
                            logger.error("Failed to get new token for reconnection - token_provider returned None")
                            # Don't make recursive call, let reconnection counter control retry
                    except Exception as e:
                        logger.error(f"Exception occurred while getting new token: {e}", exc_info=True)
                        # Continue trying to use old token
                
                # If getting new token failed, try using old token
                if not self._websocket and self._token_data:
                    # If no token_provider or getting new token failed, try using old token_data
                    logger.warning("Using cached token for reconnection")
                    try:
                        await self.connect(self._token_data)
                        logger.info("Reconnection successful with cached token")
                        # Reconnection successful, reset parameters handled in connect()
                        # Re-setup heartbeat manager
                        await self._restore_heartbeat_manager()
                        return  # Reconnection successful, exit reconnection loop
                    except Exception as e:
                        logger.error(f"Reconnection with cached token failed: {e}")
                
                # If all failed, update reconnection interval using exponential backoff
                if not self._websocket:
                    # Double the reconnection interval, up to maximum
                    self._current_reconnect_interval = min(
                        self._current_reconnect_interval * 2, 
                        self._max_reconnect_interval
                    )
                    
                    if not self.token_provider and not self._token_data:
                        logger.error("No token available for reconnection, stopping reconnection attempts")
                        self._running = False
                        return
                    else:
                        logger.error(f"Reconnection failed, will retry in {self._current_reconnect_interval}s")
                    
            except Exception as e:
                logger.error(f"Reconnection failed: {e}", exc_info=True)
                # Update reconnection interval on failure
                self._current_reconnect_interval = min(
                    self._current_reconnect_interval * 2, 
                    self._max_reconnect_interval
                )
                
        finally:
            # Ensure reconnection flag is cleared
            self._is_reconnecting = False
    
    async def _restore_heartbeat_manager(self):
        """Restore heartbeat manager after successful reconnection"""
        try:
            if self._heartbeat_manager and self._websocket:
                logger.info("Restoring heartbeat manager after reconnection")
                # Re-set WebSocket reference
                self._heartbeat_manager.set_websocket(self._websocket)
                # If heartbeat manager is not running, restart it
                if not self._heartbeat_manager.is_running:
                    await self._heartbeat_manager.start()
                    logger.info("Heartbeat manager restarted after reconnection")
                else:
                    logger.info("Heartbeat manager was already running")
        except Exception as e:
            logger.error(f"Failed to restore heartbeat manager: {e}")
    
    async def send_message(self, message: MCPSdkResponse, original_request_data: Optional[Dict[str, Any]] = None):
        """Send message to WebSocket"""
        if not self._websocket:
            raise ConnectionError("WebSocket connection not established")
        
        await self._send_response(message, original_request_data)
    
    async def close(self):
        """Close connection"""
        self._running = False
        self._stop_event.set()
        
        if self._websocket:
            try:
                await self._websocket.close()
                logger.info("WebSocket connection closed")
            except Exception as e:
                logger.error(f"Error closing WebSocket: {e}")
            finally:
                self._websocket = None
    
    @property
    def is_connected(self) -> bool:
        """Check if connected (including reconnecting state)"""
        return self._websocket is not None or (self._running and self._is_reconnecting)
