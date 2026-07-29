package openapi2mcp

import (
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newRawTool(name, description string, schema []byte) *mcp.Tool {
	return &mcp.Tool{
		Name:        name,
		Description: description,
		InputSchema: json.RawMessage(schema),
	}
}

func toolArguments(req *mcp.CallToolRequest) map[string]any {
	if req == nil || len(req.Params.Arguments) == 0 {
		return map[string]any{}
	}
	var args map[string]any
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil || args == nil {
		return map[string]any{}
	}
	return args
}

func newToolResultError(message string, _ ...any) *mcp.CallToolResult {
	result := &mcp.CallToolResult{}
	result.SetError(errorString(message))
	return result
}

type errorString string

func (e errorString) Error() string { return string(e) }
