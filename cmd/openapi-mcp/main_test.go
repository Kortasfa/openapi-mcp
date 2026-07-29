package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	openapimcp "github.com/Kortasfa/openapi-mcp/pkg/openapi-mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPHandlerIsStatelessStreamableHTTP(t *testing.T) {
	doc, err := openapimcp.LoadOpenAPISpecFromString(`openapi: 3.0.0
info:
  title: Test
  version: "1"
paths: {}`)
	if err != nil {
		t.Fatal(err)
	}
	server := createServerWithOptions("test", "1.0.0", doc, nil, "", false)
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{
		Stateless: true, JSONResponse: true, PropagateRequestCancellation: true,
	})
	recorder := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "server/discover", "params": map[string]any{},
	})
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	handler.ServeHTTP(recorder, request)
	if recorder.Code >= 500 {
		t.Fatalf("stateless handler returned %d: %s", recorder.Code, recorder.Body.String())
	}
}
