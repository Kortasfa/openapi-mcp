package main

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ubermorgenland/openapi-mcp/pkg/openapi2mcp"
)

func createServerWithOptions(name, version string, doc *openapi3.T, operations []openapi2mcp.OpenAPIOperation, _ string, _ bool) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: name, Version: version}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{},
		SchemaCache:  mcp.NewSchemaCache(),
	})
	openapi2mcp.RegisterOpenAPITools(server, operations, doc, nil)
	return server
}
