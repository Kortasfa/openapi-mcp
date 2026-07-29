package main

import (
	openapimcp "github.com/Kortasfa/openapi-mcp/pkg/openapi-mcp"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func createServerWithOptions(name, version string, doc *openapi3.T, operations []openapimcp.OpenAPIOperation, _ string, _ bool) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: name, Version: version}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{},
		SchemaCache:  mcp.NewSchemaCache(),
	})
	openapimcp.RegisterOpenAPITools(server, operations, doc, nil)
	return server
}
