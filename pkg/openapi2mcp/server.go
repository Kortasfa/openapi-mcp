package openapi2mcp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer creates an official MCP SDK server and registers all OpenAPI tools.
func NewServer(name, version string, doc *openapi3.T) *mcp.Server {
	return NewServerWithOps(name, version, doc, ExtractOpenAPIOperations(doc))
}

// NewServerWithOps creates an official MCP SDK server with the supplied operations.
func NewServerWithOps(name, version string, doc *openapi3.T, ops []OpenAPIOperation) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{Name: name, Version: version}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{},
		SchemaCache:  mcp.NewSchemaCache(),
	})
	fmt.Fprintf(os.Stderr, "[INFO] Registering %d operations for %s\n", len(ops), name)
	runtime.GC()
	RegisterOpenAPITools(srv, ops, doc, nil)
	runtime.GC()
	return srv
}

// ServeStdio runs the official SDK stdio transport.
func ServeStdio(server *mcp.Server) error {
	return server.Run(context.Background(), &mcp.StdioTransport{})
}

// ServeStreamableHTTP runs the latest stateless Streamable HTTP transport.
func ServeStreamableHTTP(server *mcp.Server, addr string, basePath string) error {
	mux := http.NewServeMux()
	mux.Handle(normalizeBasePath(basePath), HandlerForStreamableHTTP(server, basePath))
	return http.ListenAndServe(addr, mux)
}

// ServeHTTP is retained for callers of the old API and now runs stateless Streamable HTTP.
func ServeHTTP(server *mcp.Server, addr string, basePath string) error {
	return ServeStreamableHTTP(server, addr, basePath)
}

// HandlerForStreamableHTTP returns the official stateless Streamable HTTP handler.
func HandlerForStreamableHTTP(server *mcp.Server, basePath string) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		PropagateRequestCancellation: true,
	})
}

// HandlerForBasePath is retained as a source-compatible alias during migration.
// SSE is intentionally no longer exposed; all HTTP traffic uses stateless Streamable HTTP.
func HandlerForBasePath(server *mcp.Server, basePath string) http.Handler {
	return HandlerForStreamableHTTP(server, basePath)
}

func normalizeBasePath(basePath string) string {
	if strings.TrimSpace(basePath) == "" {
		return "/mcp"
	}
	return "/" + strings.Trim(basePath, "/")
}

func GetStreamableHTTPURL(addr, basePath string) string {
	if basePath == "" {
		basePath = "/mcp"
	}
	host := strings.TrimSpace(addr)
	if host == "" {
		host = "localhost"
	} else if strings.HasPrefix(host, ":") {
		host = "localhost" + host
	}
	return "http://" + host + "/" + strings.Trim(basePath, "/")
}
