package openapi2mcp

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func HandlerForStreamableHTTP(server *mcp.Server, _ string) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		PropagateRequestCancellation: true,
	})
}
