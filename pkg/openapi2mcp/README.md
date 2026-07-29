# `openapi2mcp`

This package turns a curated OpenAPI 3.x document into MCP tools. It does not run an MCP transport or manage credentials.

## Flow

1. `spec.go` loads and validates the OpenAPI document, then selects operations marked for MCP.
2. `schema.go` creates the JSON Schema for each tool input.
3. `profile.go` stores the selected operations and schemas in a versioned compiled profile.
4. `registrar.go` registers tools on an official MCP Go SDK server.
5. `request.go` translates a tool call into an upstream HTTP request.
6. `http_client.go` forwards the client `Authorization` header to that request.
7. `response.go` converts the upstream HTTP response into an MCP tool result.

The root CLI owns stateless Streamable HTTP and hot-reloads compiled profiles. See the root `README.md` for deployment commands.

## Library usage

```go
doc, err := openapi2mcp.LoadOpenAPISpec("openapi.yaml")
if err != nil {
	log.Fatal(err)
}

server := mcp.NewServer(&mcp.Implementation{Name: "my-api", Version: doc.Info.Version}, nil)
operations := openapi2mcp.ExtractOpenAPIOperations(doc)
openapi2mcp.RegisterOpenAPITools(server, operations, doc, nil)
```

The application creates the SDK's `mcp.NewStreamableHTTPHandler` itself. This keeps the transport and any authentication policy outside the OpenAPI conversion package.
