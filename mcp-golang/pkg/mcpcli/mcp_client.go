package mcp

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type Client struct {
	hosts  string
	client *client.Client // 内部MCP客户端
}

func NewClient(hosts string) (*Client, error) {
	mcpClient, err := NewSSEMCPClient(hosts)
	if err != nil {
		return nil, err
	}

	err = mcpClient.Start(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to start MCP client: %w", err)
	}

	ctx := context.Background()
	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	return &Client{
		hosts:  hosts,
		client: mcpClient,
	}, nil
}

func NewSSEMCPClient(baseURL string) (*client.Client, error) {
	_, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid MCP base URL: %w", err)
	}

	mcpClient, err := client.NewSSEMCPClient(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE MCP client: %w", err)
	}

	return mcpClient, nil
}

func NewStreamableHttpClient(baseURL string) (*client.Client, error) {
	_, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid MCP base URL: %w", err)
	}

	mcpClient, err := client.NewStreamableHttpClient(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE MCP client: %w", err)
	}

	return mcpClient, nil
}

func (c *Client) GetClient() *client.Client {
	return c.client
}

func (c *Client) ListTools(request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	tools, err := c.client.ListTools(context.Background(), request)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}
	return tools, nil
}

func (c *Client) CallTool(request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tool, err := c.client.CallTool(context.Background(), request)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool: %w", err)
	}
	return tool, nil
}

func (c *Client) Close() {
	if err := c.client.Close(); err != nil {
		log.Printf("failed to close MCP client: %v", err)
	}
}
