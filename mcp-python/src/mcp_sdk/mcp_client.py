"""
MCP Client Manager for handling MCP server connections using FastMCP
"""

import asyncio
import json
import logging
import time
from typing import Optional, Dict, Any

from fastmcp import Client
from .models import MCPSdkRequest, MCPSdkResponse
from .exceptions import MCPClientError

logger = logging.getLogger(__name__)


class MCPClientManager:
    """MCP client manager using FastMCP Client for handling connections to MCP servers"""
    
    def __init__(self, mcp_server_endpoint: str):
        self.mcp_server_endpoint = mcp_server_endpoint
        self._client: Optional[Client] = None
        self._connected = False
        self._reconnect_interval = 5
        self._max_reconnect_attempts = 3
        self._reconnect_attempts = 0
    
    async def connect(self):
        """Connect to MCP server using FastMCP Client"""
        try:
            logger.info(f"Connecting to MCP server: {self.mcp_server_endpoint}")
            
            # Create FastMCP Client with transport endpoint
            self._client = Client(transport=self.mcp_server_endpoint)
            
            # Connect to the MCP server
            await self._client.__aenter__()
            
            self._connected = True
            self._reconnect_attempts = 0
            logger.info("MCP server connection successful")
            
        except Exception as e:
            await self.disconnect()
            raise MCPClientError(f"Failed to connect to MCP server: {e}")
    
    async def disconnect(self):
        """Disconnect from MCP server"""
        try:
            if self._client:
                await self._client.__aexit__(None, None, None)
                self._client = None
            
            self._connected = False
            logger.info("MCP server connection disconnected")
            
        except Exception as e:
            logger.error(f"Error occurred while disconnecting from MCP server: {e}")
            self._client = None
            self._connected = False
    
    async def send_request(self, request: MCPSdkRequest) -> MCPSdkResponse:
        """
        Send request through FastMCP client
        
        Args:
            request: SDK request
            
        Returns:
            SDK response
            
        Raises:
            MCPClientError: MCP client error
        """
        if not self._connected or not self._client:
            raise MCPClientError("MCP client not connected")
        
        try:
            logger.debug(f"Sending MCP request: {request.request_id}")
            
            # Extract MCP request from Gateway request
            # request.request is a JSON string, need to parse it
            if isinstance(request.request, str):
                mcp_request = json.loads(request.request)
            else:
                mcp_request = request.request
                
            method = mcp_request.get("method")
            params = mcp_request.get("params", {})
            
            # Forward request to MCP server using FastMCP (returns JSON string)
            response_string = await self._forward_mcp_request(method, params)
            
            # Build SDK response with string response
            response = MCPSdkResponse(
                request_id=request.request_id,
                endpoint=request.endpoint,
                version=request.version,
                method=request.method,
                ts=str(int(time.time() * 1000)),
                response=response_string  # Now response is a string
            )
            
            logger.debug(f"MCP request processing completed: {request.request_id}")
            return response
            
        except Exception as e:
            logger.error(f"MCP request processing failed: {e}")
            # Return error response as JSON string
            error_string = json.dumps({"error": str(e)}, separators=(',', ':'), ensure_ascii=False)
            return MCPSdkResponse(
                request_id=request.request_id,
                endpoint=request.endpoint,
                version=request.version,
                method=request.method,
                ts=str(int(time.time() * 1000)),
                response=error_string
            )
    
    async def _forward_mcp_request(self, method: str, params: Dict[str, Any]) -> str:
        """
        Forward MCP request to server using FastMCP
        
        Args:
            method: MCP method name
            params: MCP method parameters
            
        Returns:
            MCP response data as JSON string
        """
        if not self._client:
            raise MCPClientError("MCP client not available")
        
        try:
            # Route to appropriate FastMCP method based on MCP method type
            if method == "tools/list":
                tools = await self._client.list_tools()
                # Convert tool object to dictionary structure conforming to MCP protocol
                tools_list = []
                for tool in tools:
                    tool_dict = {
                        "name": tool.name,
                        "description": tool.description,
                    }
                    # Add title field (if available, otherwise use name)
                    if hasattr(tool, 'title') and tool.title:
                        tool_dict["title"] = tool.title
                    else:
                        tool_dict["title"] = tool.name
                    
                    # Add inputSchema field
                    if hasattr(tool, 'inputSchema') and tool.inputSchema:
                        tool_dict["inputSchema"] = tool.inputSchema
                    elif hasattr(tool, 'parameters') and tool.parameters:
                        tool_dict["inputSchema"] = tool.parameters
                    else:
                        # Provide default inputSchema
                        tool_dict["inputSchema"] = {
                            "type": "object",
                            "properties": {},
                            "required": []
                        }
                    
                    tools_list.append(tool_dict)
                
                # Build MCP protocol compliant response structure
                response_data = {
                    "tools": tools_list
                }
                
                # Add nextCursor field (if pagination is needed)
                # Can be set based on actual pagination logic
                cursor = params.get("cursor")
                if cursor or len(tools_list) > 0:
                    # If pagination is needed, can set nextCursor here
                    # response_data["nextCursor"] = "next-page-cursor"
                    pass
                
            elif method == "tools/call":
                tool_name = params.get("name")
                arguments = params.get("arguments", {})
                
                if not tool_name:
                    raise MCPClientError("Tool call missing name parameter")
                
                result = await self._client.call_tool(tool_name, arguments)
                
                # Build MCP protocol compliant tools/call response structure
                response_data = {
                    "content": [],
                    "isError": False
                }
                
                # Handle FastMCP returned results
                if hasattr(result, 'content') and result.content:
                    # result.content is an array, iterate through each content item
                    for content_item in result.content:
                        if hasattr(content_item, 'text') and hasattr(content_item, 'type'):
                            # Standard content object, add directly
                            response_data["content"].append({
                                "type": content_item.type,
                                "text": content_item.text
                            })
                        elif hasattr(content_item, 'text'):
                            # Only text attribute, default type is text
                            response_data["content"].append({
                                "type": "text",
                                "text": content_item.text
                            })
                        else:
                            # Other types, try to convert to string
                            response_data["content"].append({
                                "type": "text",
                                "text": str(content_item)
                            })
                else:
                    # If no content attribute or content is empty, use result directly
                    response_data["content"].append({
                        "type": "text",
                        "text": str(result)
                    })
                
            else:
                raise MCPClientError(f"Unsupported MCP method: {method}")
            
            # Convert response data to JSON string
            return json.dumps(response_data, separators=(',', ':'), ensure_ascii=False)
                
        except Exception as e:
            # Return error as JSON string
            error_response = {"error": str(e)}
            return json.dumps(error_response, separators=(',', ':'), ensure_ascii=False)
    
    async def _attempt_reconnect(self):
        """Attempt to reconnect to MCP server"""
        if self._reconnect_attempts >= self._max_reconnect_attempts:
            logger.error("Maximum MCP server reconnection attempts reached")
            return False
        
        self._reconnect_attempts += 1
        logger.info(f"Attempting MCP server reconnection ({self._reconnect_attempts}/{self._max_reconnect_attempts})")
        
        try:
            await asyncio.sleep(self._reconnect_interval)
            await self.connect()
            return True
        except Exception as e:
            logger.error(f"MCP server reconnection failed: {e}")
            return await self._attempt_reconnect()
    
    @property
    def is_connected(self) -> bool:
        """Check if connected"""
        return self._connected and self._client is not None
