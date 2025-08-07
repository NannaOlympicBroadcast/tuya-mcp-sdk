#!/usr/bin/env python3
"""
MCP Examples Launcher
Convenient script to start MCP servers and client examples
"""

import asyncio
import subprocess
import sys
import logging
import argparse
from pathlib import Path

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class MCPLauncher:
    """MCP examples launcher"""
    
    def __init__(self):
        self.processes = []
        self.base_dir = Path(__file__).parent
    
    async def start_mcp_server(self):
        """Start MCP server"""
        logger.info("🔧 Starting Mock MCP Server...")
        
        cmd = [sys.executable, str(self.base_dir / "mcp" / "mock_mcp_server.py")]
        process = subprocess.Popen(
            cmd,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            universal_newlines=True,
            bufsize=1
        )
        
        self.processes.append(("MCP Server", process))
        return process
    
    async def start_mcpsdk_client(self, endpoint=None, access_id=None, access_secret=None, custom_mcp_server_endpoint=None):
        """Start MCP SDK client (quick_start.py)"""
        logger.info("🚀 Starting MCP SDK Client...")
        
        # Prepare environment variables for quick_start.py
        env = {}
        if endpoint:
            env['ENDPOINT'] = endpoint
        if access_id:
            env['ACCESS_ID'] = access_id
        if access_secret:
            env['ACCESS_SECRET'] = access_secret
        if custom_mcp_server_endpoint:
            env['CUSTOM_MCP_SERVER_ENDPOINT'] = custom_mcp_server_endpoint
            
        cmd = [sys.executable, str(self.base_dir / "quick_start.py")]
        
        # Pass environment variables to the subprocess
        import os
        process_env = os.environ.copy()
        process_env.update(env)
        
        process = subprocess.Popen(
            cmd,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            universal_newlines=True,
            bufsize=1,
            env=process_env
        )
        
        self.processes.append(("MCP SDK Client", process))
        return process
    
    
    async def start_all(self, mode="all", **kwargs):
        """Start services based on mode"""
        print("🎯 MCP Examples Launcher")
        print("=" * 50)
        
        if mode == "server" or mode == "all":
            print("Starting mock MCP server for testing...")
            print()
            
            # Start MCP server
            mcp_process = await self.start_mcp_server()
            await asyncio.sleep(1)  # Wait for startup
            
            logger.info("✅ Mock MCP server started successfully!")
        
        if mode == "client" or mode == "all":
            print("Starting MCP SDK client...")
            print()
            
            # Start MCP SDK client
            mcpsdk_process = await self.start_mcpsdk_client(
                endpoint=kwargs.get('endpoint'),
                access_id=kwargs.get('access_id'),
                access_secret=kwargs.get('access_secret'),
                custom_mcp_server_endpoint=kwargs.get('custom_mcp_server_endpoint')
            )
            await asyncio.sleep(1)  # Wait for startup
            
            logger.info("✅ MCP SDK client started successfully!")
        
        if mode == "all":
            print()
            print("📋 Services Information:")
            print("─" * 30)
            print("🔧 Mock MCP Server:")
            print()
            print("MCP SDK Client:")
            print("   Endpoint:", kwargs.get('endpoint', 'your-endpoint'))
            print("   Custom MCP Server Endpoint:", kwargs.get('custom_mcp_server_endpoint', 'http://localhost:8765/mcp'))
            print()
        
        print("�📊 Monitoring:")
        
        try:
            # Monitor process status
            while True:
                await asyncio.sleep(10)
                self.check_processes()
                
        except KeyboardInterrupt:
            logger.info("👋 Shutting down services...")
            await self.stop_all()
    
    
    def check_processes(self):
        """Check process status"""
        active_processes = []
        for name, process in self.processes:
            if process.poll() is not None:
                logger.error(f"❌ {name} has stopped unexpectedly!")
            else:
                active_processes.append((name, process))
        
        if active_processes:
            pids = [str(p[1].pid) for p in active_processes]
            names = [p[0] for p in active_processes]
            logger.info(f"💚 Services running: {', '.join(names)} (PIDs: {', '.join(pids)})")
            return True
        return False
    
    async def stop_all(self):
        """Stop all services"""
        for name, process in self.processes:
            try:
                logger.info(f"Stopping {name}...")
                process.terminate()
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    logger.warning(f"Force killing {name}...")
                    process.kill()
                    process.wait()
                logger.info(f"✅ {name} stopped")
            except Exception as e:
                logger.error(f"Error stopping {name}: {e}")
        
        self.processes.clear()


def parse_args():
    """Parse command line arguments"""
    parser = argparse.ArgumentParser(description='MCP Examples Launcher')
    parser.add_argument('mode', choices=['all', 'server', 'client'], default='all', nargs='?',
                      help='Launch mode: all (server+client), server only, or client only')
    parser.add_argument('--endpoint', default='your-endpoint',
                      help='Endpoint')
    parser.add_argument('--access-id', default='your-access-id',
                      help='Access ID')
    parser.add_argument('--access-secret', default='your-access-secret',
                      help='Access secret')
    parser.add_argument('--custom-mcp-server-endpoint', default='http://localhost:8765/mcp',
                      help='Custom MCP server endpoint')
    return parser.parse_args()


async def main():
    """Main function"""
    args = parse_args()
    
    launcher = MCPLauncher()
    
    try:
        await launcher.start_all(
            mode=args.mode,
            endpoint=args.endpoint,
            access_id=args.access_id,
            access_secret=args.access_secret,
            custom_mcp_server_endpoint=args.custom_mcp_server_endpoint
        )
    except Exception as e:
        logger.error(f"Error starting services: {e}")
        await launcher.stop_all()
        sys.exit(1)


if __name__ == "__main__":
    print("🎯 MCP Examples Launcher")
    print("=" * 50)
    print("This script starts MCP servers and client examples for testing.")
    print()
    print("Available modes:")
    print("• all     - Start both MCP server and MCP SDK client")
    print("• server  - Start only MCP server")
    print("• client  - Start only MCP SDK client")
    print()
    print("Usage examples:")
    print("  python -m examples                                    # Start all with default settings")
    print("  python -m examples server                             # Start only server")
    print("  python -m examples client --endpoint your-endpoint    # Start only client with custom endpoint")
    print("  python -m examples all --endpoint your-endpoint \\")
    print("                         --access-id your-access-id \\")
    print("                         --access-secret your-access-secret")
    print()
    print("Press Ctrl+C to stop the services")
    print("=" * 50)
    print()
    
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\n👋 Services stopped.")
