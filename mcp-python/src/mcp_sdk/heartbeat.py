"""
Heartbeat Manager
Heartbeat Manager - Based on WebSocket ping/pong mechanism
"""

import asyncio
import logging
import time
from typing import Optional

from .exceptions import HeartbeatError

logger = logging.getLogger(__name__)


class HeartbeatManager:
    """
    Heartbeat Manager
    Heartbeat manager based on WebSocket ping/pong mechanism
    
    Note: Actual heartbeat is handled automatically by WebSocket client library (ping_interval, ping_timeout)
    This manager is mainly used for monitoring connection status
    """
    
    def __init__(
        self,
        ping_interval: int = 30,
        ping_timeout: int = 10
    ):
        """
        Initialize heartbeat manager
        
        Args:
            ping_interval: WebSocket ping interval (seconds)
            ping_timeout: WebSocket ping timeout (seconds)
        """
        self.ping_interval = ping_interval
        self.ping_timeout = ping_timeout
        
        self._running = False
        self._monitor_task: Optional[asyncio.Task] = None
        self._last_pong_time = 0
        self._websocket = None
    
    def set_websocket(self, websocket):
        """Set WebSocket connection"""
        self._websocket = websocket
        self._last_pong_time = time.time()
    
    async def start(self):
        """Start heartbeat monitoring"""
        if self._running:
            return
        
        self._running = True
        self._last_pong_time = time.time()
        self._monitor_task = asyncio.create_task(self._monitor_loop())
        logger.info(f"Heartbeat monitor started, ping_interval: {self.ping_interval}s, ping_timeout: {self.ping_timeout}s")
    
    
    async def stop(self):
        """Stop heartbeat monitoring"""
        self._running = False
        
        if self._monitor_task:
            self._monitor_task.cancel()
            try:
                await self._monitor_task
            except asyncio.CancelledError:
                pass
            self._monitor_task = None
        
        logger.info("Heartbeat monitor stopped")
    
    async def _monitor_loop(self):
        """
        Monitor loop
        Note: Actual ping/pong is handled automatically by WebSocket library
        This only monitors connection status
        """
        while self._running:
            try:
                await asyncio.sleep(self.ping_interval)
                
                if not self._running:
                    break
                
                # Check WebSocket connection status
                if self._websocket:
                    if hasattr(self._websocket, 'closed') and self._websocket.closed:
                        logger.warning("WebSocket connection is closed")
                        raise HeartbeatError("WebSocket connection closed")
                    
                    # Check if timeout (based on ping_interval + ping_timeout)
                    current_time = time.time()
                    expected_timeout = self.ping_interval + self.ping_timeout
                    if current_time - self._last_pong_time > expected_timeout * 2:
                        logger.warning(f"No pong received for {current_time - self._last_pong_time:.1f}s")
                        # Don't throw exception immediately as WebSocket library will handle reconnection automatically
                
                # Update last pong received time (simplified handling)
                self._last_pong_time = time.time()
                
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error(f"Heartbeat monitoring error: {e}")
                if self._running:
                    # Wait a while before retry
                    await asyncio.sleep(5)
    
    def on_websocket_message(self):
        """Called when WebSocket message is received (indicating connection is normal)"""
        self._last_pong_time = time.time()
    
    def mark_service_offline(self):
        """Mark service as offline"""
        logger.warning("Service marked as offline")
        self._running = False
        # Can add more offline state handling logic here
        # For example: notify other components, record status, etc.
    
    @property
    def is_running(self) -> bool:
        """Check if heartbeat monitoring is running"""
        return self._running
    
    @property
    def ping_config(self) -> dict:
        """Get ping configuration for WebSocket connection"""
        return {
            "ping_interval": self.ping_interval,
            "ping_timeout": self.ping_timeout
        }
