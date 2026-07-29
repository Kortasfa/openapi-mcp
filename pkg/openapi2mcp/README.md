# openapi2mcp Go Library

This package provides a Go library for converting OpenAPI 3.x specifications into MCP (Model Context Protocol) tool servers.

## Installation

```bash
go get github.com/jedisct1/openapi-mcp/pkg/openapi2mcp
```

For direct access to MCP types and tools, use the official SDK:
```bash
go get github.com/modelcontextprotocol/go-sdk/mcp
```

## Usage

```go
package main

import (
        "log"
        "github.com/jedisct1/openapi-mcp/pkg/openapi2mcp"
)

func main() {
        // Load OpenAPI spec
        doc, err := openapi2mcp.LoadOpenAPISpec("openapi.yaml")
        if err != nil {
                log.Fatal(err)
        }

        // Create MCP server
        srv := openapi2mcp.NewServer("myapi", doc.Info.Version, doc)

        // Serve over HTTP (StreamableHTTP is now the default)
        if err := openapi2mcp.ServeStreamableHTTP(srv, ":8080", "/mcp"); err != nil {
                log.Fatal(err)
        }

        // Or serve over stdio
        // if err := openapi2mcp.ServeStdio(srv); err != nil {
        //     log.Fatal(err)
        // }
}
```

## Features

- Convert OpenAPI 3.x specifications to MCP tool servers
- Support for stateless Streamable HTTP and stdio transports through the official MCP Go SDK
- Automatic tool generation from OpenAPI operations
- Built-in validation and error handling
- AI-optimized responses with structured output

## API Documentation

See [GoDoc](https://pkg.go.dev/github.com/jedisct1/openapi-mcp/pkg/openapi2mcp) for complete API documentation.

### HTTP Client Development

When using HTTP mode, openapi-mcp serves the official MCP Go SDK's stateless Streamable HTTP handler. Clients send independent POST requests to the same endpoint without session management.

See the [StreamableHTTP specification](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#streamable-http) for protocol details.
