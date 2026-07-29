package openapi2mcp

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestExtractOpenAPIOperationsHonorsMCPCurationExtensions(t *testing.T) {
	doc := &openapi3.T{
		Extensions: map[string]any{"x-mcp-curated": true},
		Paths:      openapi3.NewPaths(),
	}
	doc.Paths.Set("/enabled", &openapi3.PathItem{Get: &openapi3.Operation{
		OperationID: "enabled",
		Extensions:  map[string]any{"x-mcp-enabled": true, "x-mcp-base-path": "/api/v2"},
	}})
	doc.Paths.Set("/disabled", &openapi3.PathItem{Get: &openapi3.Operation{
		OperationID: "disabled",
		Extensions:  map[string]any{"x-mcp-enabled": true, "x-mcp-disabled": true},
	}})
	doc.Paths.Set("/unmarked", &openapi3.PathItem{Get: &openapi3.Operation{OperationID: "unmarked"}})

	operations := ExtractOpenAPIOperations(doc)
	if len(operations) != 1 || operations[0].OperationID != "enabled" {
		t.Fatalf("curated operations = %#v, want only enabled", operations)
	}
	if operations[0].MCPBasePath != "/api/v2" {
		t.Fatalf("MCPBasePath = %q, want %q", operations[0].MCPBasePath, "/api/v2")
	}
}
