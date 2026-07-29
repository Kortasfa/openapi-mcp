// Package openapi2mcp compiles a curated OpenAPI 3.x document into MCP tools.
//
// The package owns OpenAPI loading, tool schemas, request translation, and response
// formatting. The application owns the MCP server and its Streamable HTTP transport.
package openapi2mcp

import (
	"github.com/getkin/kin-openapi/openapi3"
)

// OpenAPIOperation is the normalized representation used to create one MCP tool.
type OpenAPIOperation struct {
	OperationID string
	Summary     string
	Description string
	Path        string
	Method      string
	Parameters  openapi3.Parameters
	RequestBody *openapi3.RequestBodyRef
	Tags        []string
	MCPReadOnly bool
	MCPBasePath string
}

// ToolGenOptions filters the operations exposed by a caller-created MCP server.
type ToolGenOptions struct {
	TagFilter []string
}
