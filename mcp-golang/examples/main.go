package main

import (
	"log"
	"mcp-sdk/examples/mcp"
	"mcp-sdk/pkg/config"
	sdk "mcp-sdk/pkg/mcpsdk"
)

func main() {
	conf := config.InitializeConfig()

	// Running Custom MCP Server
	go mcp.NewMCPServer().StartHTTP(conf.CustomMcpServerEndpoint)

	ch := make(chan struct{})
	// need block running
	println("MCP SDK starting...")
	mcpsdk, err := sdk.NewMCPSdk(
		// Set custom MCP server hosts
		sdk.WithMCPServerEndpoint(conf.CustomMcpServerEndpoint),
		// Set Tuya access key, access secret and endpoint
		sdk.WithAccessParams(conf.AccessId, conf.AccessSecret, conf.Endpoint),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = mcpsdk.Run()
	if err != nil {
		log.Fatal(err)
	}
	println("MCP SDK started")
	<-ch
}
