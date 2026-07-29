package openapi2mcp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func formatUpstreamResponse(operation OpenAPIOperation, inputSchemaJSON []byte, inputSchema map[string]any, args map[string]any, requestURL string, response *http.Response, body []byte) *mcp.CallToolResult {
	contentType := response.Header.Get("Content-Type")
	isJSON := strings.HasPrefix(contentType, "application/json") || strings.HasPrefix(contentType, "application/vnd.api+json")
	isText := strings.HasPrefix(contentType, "text/")
	isBinary := !isJSON && !isText
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return formatUpstreamError(operation, inputSchemaJSON, inputSchema, args, requestURL, response, body, isBinary)
	}
	if isBinary {
		result := fileResponseObject(operation, response, body)
		return structuredResult(result)
	}

	text := fmt.Sprintf("HTTP %s %s\nStatus: %d\nResponse:\n%s", operation.Method, requestURL, response.StatusCode, body)
	if args["stream"] == true {
		return partialResult(text, "stream-"+fmt.Sprintf("%d", rand.Intn(1000)))
	}
	if resumeToken, ok := args["resume_token"].(string); ok && resumeToken != "" {
		return partialResult(text, resumeToken)
	}
	if isJSON {
		var data any
		if json.Unmarshal(body, &data) == nil {
			return structuredResult(apiResponseObject(operation, response.StatusCode, data))
		}
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func formatUpstreamError(operation OpenAPIOperation, inputSchemaJSON []byte, inputSchema map[string]any, args map[string]any, requestURL string, response *http.Response, body []byte, binary bool) *mcp.CallToolResult {
	suggestion := "Check the tool arguments and the client Bearer token, then retry."
	if binary {
		result := fileResponseObject(operation, response, body)
		result["error"] = map[string]any{
			"code":        "http_error",
			"http_status": response.StatusCode,
			"message":     fmt.Sprintf("%s (HTTP %d)", http.StatusText(response.StatusCode), response.StatusCode),
			"suggestion":  suggestion,
		}
		callResult := structuredResult(result)
		callResult.IsError = true
		return callResult
	}
	summary := operation.Summary
	if summary == "" {
		summary = operation.Description
	}
	text := fmt.Sprintf("HTTP %s %s\nError: %s (HTTP %d)", operation.Method, requestURL, http.StatusText(response.StatusCode), response.StatusCode)
	if len(body) > 0 {
		text += "\nDetails: " + string(body)
	}
	return newToolResultError(text+"\nSuggestion: "+suggestion+fmt.Sprintf("\nOperation: %s (%s)", operation.OperationID, summary), inputSchema, args, []any{args}, "call <tool> <json-args>", []string{"list", "schema <tool>"})
}

func apiResponseObject(operation OpenAPIOperation, status int, data any) map[string]any {
	return map[string]any{"type": "api_response", "http_status": status, "operation": operationMetadata(operation), "data": data}
}

func fileResponseObject(operation OpenAPIOperation, response *http.Response, body []byte) map[string]any {
	fileName := "file"
	if disposition := response.Header.Get("Content-Disposition"); disposition != "" {
		if parts := strings.Split(disposition, "filename="); len(parts) > 1 {
			fileName = strings.Trim(parts[1], `"`)
		}
	}
	return map[string]any{"type": "api_response", "http_status": response.StatusCode, "mime_type": response.Header.Get("Content-Type"), "file_base64": base64.StdEncoding.EncodeToString(body), "file_name": fileName, "operation": operationMetadata(operation)}
}

func operationMetadata(operation OpenAPIOperation) map[string]any {
	return map[string]any{"id": operation.OperationID, "summary": operation.Summary, "description": operation.Description}
}

func structuredResult(value map[string]any) *mcp.CallToolResult {
	encoded, _ := json.MarshalIndent(value, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(encoded)}}, StructuredContent: value}
}

func partialResult(text, resumeToken string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}, StructuredContent: map[string]any{"partial": true, "resume_token": resumeToken}}
}
