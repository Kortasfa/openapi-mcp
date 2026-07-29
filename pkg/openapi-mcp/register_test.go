package openapimcp

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
	request.Header.Set("Accept", "application/json, text/event-stream")
	handler.ServeHTTP(recorder, request)
	if recorder.Code >= 500 {
		t.Fatalf("tools/list failed: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestRegisterOpenAPIToolsFiltersByTag(t *testing.T) {
	doc := &openapi3.T{
		Info:  &openapi3.Info{Title: "Test API", Version: "1.0.0"},
		Paths: openapi3.NewPaths(),
	}
	doc.Paths.Set("/users", &openapi3.PathItem{Get: &openapi3.Operation{OperationID: "listUsers", Tags: []string{"users"}}})
	doc.Paths.Set("/groups", &openapi3.PathItem{Get: &openapi3.Operation{OperationID: "listGroups", Tags: []string{"groups"}}})

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	RegisterOpenAPITools(server, ExtractOpenAPIOperations(doc), doc, &ToolGenOptions{TagFilter: []string{"users"}})
	handler := testStreamableHandler(server)

	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("tools/list failed: %d %s", recorder.Code, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"name":"listUsers"`)) {
		t.Fatalf("filtered tool missing: %s", recorder.Body.String())
	}
	if bytes.Contains(recorder.Body.Bytes(), []byte(`"name":"listGroups"`)) {
		t.Fatalf("unfiltered tool was registered: %s", recorder.Body.String())
	}
}

func TestRegisterOpenAPIToolsForwardsMCPBearerToken(t *testing.T) {
	const token = "client-access-token"
	t.Setenv("BEARER_TOKEN", "server-fallback-token")
	type upstreamRequest struct {
		authorization string
		accept        string
		departments   []string
		path          string
	}
	upstreamRequests := make(chan upstreamRequest, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		upstreamRequests <- upstreamRequest{
			authorization: request.Header.Get("Authorization"),
			accept:        request.Header.Get("Accept"),
			departments:   request.URL.Query()["departments[]"],
			path:          request.URL.Path,
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	doc := &openapi3.T{
		Info:    &openapi3.Info{Title: "Test API", Version: "1.0.0"},
		Servers: openapi3.Servers{{URL: upstream.URL + "/api/v3"}},
		Components: &openapi3.Components{SecuritySchemes: openapi3.SecuritySchemes{
			"bearerAuth": {Value: &openapi3.SecurityScheme{Type: "http", Scheme: "bearer"}},
		}},
		Paths: openapi3.NewPaths(),
	}
	doc.Paths.Set("/hello", &openapi3.PathItem{Get: &openapi3.Operation{
		OperationID: "getHello",
		Extensions:  map[string]any{"x-mcp-base-path": "/api/v2"},
		Parameters: openapi3.Parameters{&openapi3.ParameterRef{Value: &openapi3.Parameter{
			Name:   "departments[]",
			In:     "query",
			Schema: &openapi3.SchemaRef{Value: openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())},
		}}},
	}})

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	RegisterOpenAPITools(server, ExtractOpenAPIOperations(doc), doc, nil)
	handler := testStreamableHandler(server)

	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "getHello",
			"arguments": map[string]any{"departments__": []string{"department-one", "department-two"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("tools/call failed: %d %s", recorder.Code, recorder.Body.String())
	}
	if bytes.Contains(recorder.Body.Bytes(), []byte(`"partial":true`)) {
		t.Fatalf("tools/call returned an unexpected partial response: %s", recorder.Body.String())
	}
	var response struct {
		Result struct {
			StructuredContent map[string]any `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode MCP response: %v", err)
	}
	data, ok := response.Result.StructuredContent["data"].(map[string]any)
	if !ok || data["ok"] != true {
		t.Fatalf("structuredContent.data = %#v, want {ok: true}", response.Result.StructuredContent["data"])
	}
	requestDetails := <-upstreamRequests
	if requestDetails.authorization != "Bearer "+token {
		t.Fatalf("upstream Authorization = %q, want %q", requestDetails.authorization, "Bearer "+token)
	}
	if requestDetails.accept != "application/json, application/vnd.api+json" {
		t.Fatalf("upstream Accept = %q", requestDetails.accept)
	}
	if got := requestDetails.departments; len(got) != 2 || got[0] != "department-one" || got[1] != "department-two" {
		t.Fatalf("upstream departments[] = %#v", got)
	}
	if requestDetails.path != "/api/v2/hello" {
		t.Fatalf("upstream path = %q, want %q", requestDetails.path, "/api/v2/hello")
	}
}

func TestWriteOperationRequiresConfirmationBeforeRequest(t *testing.T) {
	upstreamCalled := make(chan struct{}, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		upstreamCalled <- struct{}{}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	doc := &openapi3.T{
		Info:    &openapi3.Info{Title: "Test API", Version: "1.0.0"},
		Servers: openapi3.Servers{{URL: upstream.URL}},
		Paths:   openapi3.NewPaths(),
	}
	doc.Paths.Set("/users", &openapi3.PathItem{Post: &openapi3.Operation{OperationID: "createUser"}})

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	RegisterOpenAPITools(server, ExtractOpenAPIOperations(doc), doc, nil)
	handler := testStreamableHandler(server)

	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "createUser",
			"arguments": map[string]any{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("tools/call failed: %d %s", recorder.Code, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte("CONFIRMATION REQUIRED")) {
		t.Fatalf("tools/call did not request confirmation: %s", recorder.Body.String())
	}
	select {
	case <-upstreamCalled:
		t.Fatal("write operation reached upstream without confirmation")
	default:
	}
}

func testStreamableHandler(server *mcp.Server) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{
		Stateless: true, JSONResponse: true, PropagateRequestCancellation: true,
	})
}
