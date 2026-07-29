package openapimcp

import (
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestLoadOpenAPISpecFromBytesValidatesDocument(t *testing.T) {
	doc, err := LoadOpenAPISpecFromString(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /health:
    get:
      operationId: getHealth
      responses:
        '200':
          description: OK
`)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if doc.Info.Title != "Test API" {
		t.Fatalf("unexpected document: %#v", doc.Info)
	}

	_, err = LoadOpenAPISpecFromBytes([]byte(`openapi: 3.0.0
info: {title: Test, version: 1.0}
paths: invalid
`))
	if err == nil || !strings.Contains(err.Error(), "parse OpenAPI spec") {
		t.Fatalf("unexpected invalid-spec error: %v", err)
	}
}

func TestExtractOpenAPIOperationsHonorsMCPCurationExtensions(t *testing.T) {
	doc := &openapi3.T{Extensions: map[string]any{"x-mcp-curated": true}, Paths: openapi3.NewPaths()}
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

func TestExtractOpenAPIOperationsUsesDeterministicOrderAndParameterOverrides(t *testing.T) {
	pathParameter := &openapi3.ParameterRef{Value: &openapi3.Parameter{
		Name: "page", In: "query", Schema: schemaRef(&openapi3.Schema{Type: typesPtr("integer")}),
	}}
	operationParameter := &openapi3.ParameterRef{Value: &openapi3.Parameter{
		Name: "page", In: "query", Schema: schemaRef(&openapi3.Schema{Type: typesPtr("string")}),
	}}
	doc := &openapi3.T{Paths: openapi3.NewPaths()}
	doc.Paths.Set("/z", &openapi3.PathItem{Post: &openapi3.Operation{Parameters: openapi3.Parameters{pathParameter}}})
	doc.Paths.Set("/a", &openapi3.PathItem{
		Parameters: openapi3.Parameters{pathParameter},
		Get:        &openapi3.Operation{OperationID: "listA", Parameters: openapi3.Parameters{operationParameter}},
	})

	operations := ExtractOpenAPIOperations(doc)
	if len(operations) != 2 || operations[0].OperationID != "listA" || operations[1].OperationID != "post_z" {
		t.Fatalf("unexpected operation order: %#v", operations)
	}
	if len(operations[0].Parameters) != 1 || operations[0].Parameters[0] != operationParameter {
		t.Fatalf("operation parameter did not override path parameter: %#v", operations[0].Parameters)
	}
}

func TestExtractOpenAPIOperationsHandlesEmptyDocument(t *testing.T) {
	if operations := ExtractOpenAPIOperations(nil); operations != nil {
		t.Fatalf("operations = %#v, want nil", operations)
	}
}
