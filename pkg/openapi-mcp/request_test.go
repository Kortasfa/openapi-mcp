package openapimcp

import (
	"context"
	"io"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestBuildUpstreamRequestMapsOpenAPIInputs(t *testing.T) {
	doc := &openapi3.T{Components: &openapi3.Components{SecuritySchemes: openapi3.SecuritySchemes{
		"apiKey": {Value: &openapi3.SecurityScheme{Type: "apiKey", In: "header", Name: "X-API-Key"}},
	}}}
	operation := OpenAPIOperation{
		Path:   "/users/{userId}",
		Method: "post",
		Parameters: openapi3.Parameters{
			{Value: &openapi3.Parameter{Name: "userId", In: "path", Schema: &openapi3.SchemaRef{Value: openapi3.NewIntegerSchema()}}},
			{Value: &openapi3.Parameter{Name: "departments[]", In: "query", Schema: &openapi3.SchemaRef{Value: openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())}}},
			{Value: &openapi3.Parameter{Name: "X-Trace-Id", In: "header", Schema: &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}}},
			{Value: &openapi3.Parameter{Name: "session", In: "cookie", Schema: &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}}},
		},
		RequestBody: &openapi3.RequestBodyRef{Value: &openapi3.RequestBody{Content: openapi3.Content{
			"application/json": &openapi3.MediaType{Schema: &openapi3.SchemaRef{Value: openapi3.NewObjectSchema()}},
		}}},
	}
	arguments := map[string]any{
		"userId":        42,
		"departments__": []any{"engineering", "sales"},
		"X-Trace-Id":    "trace-123",
		"X-API-Key":     "key-123",
		"session":       "session-123",
		"requestBody":   map[string]any{"name": "Ada"},
	}

	upstream, err := buildUpstreamRequest(context.Background(), doc, []string{"https://api.example.test/api/v3"}, operation, arguments)
	if err != nil {
		t.Fatal(err)
	}
	if upstream.URL != "https://api.example.test/api/v3/users/42?departments%5B%5D=engineering&departments%5B%5D=sales" {
		t.Fatalf("URL = %q", upstream.URL)
	}
	if upstream.Request.Method != "POST" || upstream.Request.Header.Get("Accept") != "application/json, application/vnd.api+json" {
		t.Fatalf("unexpected request metadata: %s %#v", upstream.Request.Method, upstream.Request.Header)
	}
	if upstream.Request.Header.Get("Content-Type") != "application/json" || upstream.Request.Header.Get("X-Trace-Id") != "trace-123" || upstream.Request.Header.Get("X-API-Key") != "key-123" {
		t.Fatalf("unexpected request headers: %#v", upstream.Request.Header)
	}
	if upstream.Request.Header.Get("Cookie") != "session=session-123" {
		t.Fatalf("Cookie = %q", upstream.Request.Header.Get("Cookie"))
	}
	body, err := io.ReadAll(upstream.Request.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"name":"Ada"}` {
		t.Fatalf("body = %s", body)
	}
}

func TestBuildUpstreamRequestUsesOperationBasePath(t *testing.T) {
	operation := OpenAPIOperation{Path: "/user/list", Method: "POST", MCPBasePath: "/api/v2"}
	upstream, err := buildUpstreamRequest(context.Background(), nil, []string{"https://tenant.example.test/api/v3"}, operation, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if upstream.URL != "https://tenant.example.test/api/v2/user/list" {
		t.Fatalf("URL = %q", upstream.URL)
	}
}
