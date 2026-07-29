package openapi2mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegisterOpenAPIToolsWithOfficialSDK(t *testing.T) {
	doc := &openapi3.T{
		Info:  &openapi3.Info{Title: "Test API", Version: "1.0.0"},
		Paths: openapi3.NewPaths(),
	}
	doc.Paths.Set("/hello", &openapi3.PathItem{Get: &openapi3.Operation{OperationID: "getHello"}})

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	RegisterOpenAPITools(server, ExtractOpenAPIOperations(doc), doc, nil)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": map[string]any{}})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	handler.ServeHTTP(recorder, request)
	if recorder.Code >= 500 {
		t.Fatalf("tools/list failed: %d %s", recorder.Code, recorder.Body.String())
	}
}
