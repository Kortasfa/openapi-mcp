package openapimcp

import (
	"net/http"
	"testing"
)

func TestFormatUpstreamResponseReturnsStructuredJSON(t *testing.T) {
	response := &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}}
	result := formatUpstreamResponse(OpenAPIOperation{OperationID: "listUsers"}, "https://api.example.test/users", response, []byte(`{"users":[1]}`))
	data, ok := result.StructuredContent.(map[string]any)["data"].(map[string]any)
	if !ok || len(data["users"].([]any)) != 1 {
		t.Fatalf("unexpected structured content: %#v", result.StructuredContent)
	}
}

func TestFormatUpstreamResponseMarksHTTPError(t *testing.T) {
	response := &http.Response{StatusCode: http.StatusUnauthorized, Header: http.Header{"Content-Type": []string{"application/json"}}}
	result := formatUpstreamResponse(OpenAPIOperation{OperationID: "listUsers"}, "https://api.example.test/users", response, []byte(`{"error":"expired"}`))
	if !result.IsError || len(result.Content) == 0 {
		t.Fatalf("expected an MCP error result: %#v", result)
	}
}
