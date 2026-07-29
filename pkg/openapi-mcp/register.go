package openapimcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xeipuuv/gojsonschema"
)

type ToolRegistrar struct {
	server   *mcp.Server
	doc      *openapi3.T
	opts     *ToolGenOptions
	baseURLs []string
}

func RegisterOpenAPITools(server *mcp.Server, operations []OpenAPIOperation, doc *openapi3.T, opts *ToolGenOptions) []string {
	registrar := &ToolRegistrar{server: server, doc: doc, opts: opts, baseURLs: resolveBaseURLs(doc)}
	return registrar.register(operations)
}

func resolveBaseURLs(doc *openapi3.T) []string {
	if value := os.Getenv("OPENAPI_BASE_URL"); value != "" {
		return []string{value}
	}
	var values []string
	if doc != nil {
		for _, server := range doc.Servers {
			if server != nil && server.URL != "" {
				values = append(values, server.URL)
			}
		}
	}
	if len(values) == 0 {
		return []string{"http://localhost:8080"}
	}
	return values
}

func (r *ToolRegistrar) register(operations []OpenAPIOperation) []string {
	var names []string
	for _, operation := range operations {
		if !r.includes(operation) {
			continue
		}
		name, schema, description := r.toolDefinition(operation)
		operationCopy := operation
		tool := newRawTool(name, description, schema)
		tool.Annotations = &mcp.ToolAnnotations{Title: operationTitle(operationCopy)}
		r.server.AddTool(tool, r.handler(name, schema, operationCopy))
		names = append(names, name)
	}
	return names
}

func (r *ToolRegistrar) includes(operation OpenAPIOperation) bool {
	if r.opts == nil || len(r.opts.TagFilter) == 0 {
		return true
	}
	for _, tag := range operation.Tags {
		for _, wanted := range r.opts.TagFilter {
			if tag == wanted {
				return true
			}
		}
	}
	return false
}

func (r *ToolRegistrar) toolDefinition(operation OpenAPIOperation) (string, []byte, string) {
	schema, _ := json.Marshal(BuildInputSchemaWithContext(operation.Parameters, operation.RequestBody, r.doc))
	description := operation.Description
	if description == "" {
		description = operation.Summary
	}
	if !operation.MCPReadOnly && isWriteMethod(operation.Method) {
		description += "\n\nCONFIRMATION REQUIRED: ask the user, then include __confirmed: true."
	}
	return operation.OperationID, schema, description
}

func operationTitle(operation OpenAPIOperation) string {
	if len(operation.Tags) == 0 {
		return operation.OperationID
	}
	return strings.Join(operation.Tags, ", ")
}

func (r *ToolRegistrar) handler(name string, schema []byte, operation OpenAPIOperation) func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, call *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := toolArguments(call)
		if err := validateToolArguments(schema, args); err != nil {
			return newToolResultError(err.Error()), nil
		}
		if !operation.MCPReadOnly && isWriteMethod(operation.Method) {
			if confirmed, _ := args["__confirmed"].(bool); !confirmed {
				return confirmationResult(name), nil
			}
		}
		request, err := buildUpstreamRequest(ctx, r.doc, r.baseURLs, operation, args)
		if err != nil {
			return nil, err
		}
		authorization := ""
		if call.Extra != nil {
			authorization = call.Extra.Header.Get("Authorization")
		}
		response, err := executeUpstreamRequest(ctx, request.Request, authorization)
		if err != nil {
			return nil, err
		}
		return formatUpstreamResponse(operation, request.URL, response.Response, response.Body), nil
	}
}

func validateToolArguments(schema []byte, args map[string]any) error {
	encoded, err := json.Marshal(args)
	if err != nil {
		return err
	}
	result, err := gojsonschema.Validate(gojsonschema.NewBytesLoader(schema), gojsonschema.NewBytesLoader(encoded))
	if err != nil {
		return err
	}
	if result.Valid() {
		return nil
	}
	var messages []string
	for _, validationError := range result.Errors() {
		messages = append(messages, validationError.String())
	}
	return fmt.Errorf("invalid tool arguments: %s", strings.Join(messages, "; "))
}

func confirmationResult(name string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("CONFIRMATION REQUIRED: retry %s with __confirmed: true after user approval.", name)}}}
}

func isWriteMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		return true
	}
	return false
}

func newRawTool(name, description string, schema []byte) *mcp.Tool {
	return &mcp.Tool{Name: name, Description: description, InputSchema: json.RawMessage(schema)}
}

func toolArguments(request *mcp.CallToolRequest) map[string]any {
	if request == nil || len(request.Params.Arguments) == 0 {
		return map[string]any{}
	}
	var args map[string]any
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil || args == nil {
		return map[string]any{}
	}
	return args
}

func newToolResultError(message string) *mcp.CallToolResult {
	result := &mcp.CallToolResult{}
	result.SetError(errorString(message))
	return result
}

type errorString string

func (value errorString) Error() string { return string(value) }

func getParameterValue(args map[string]any, name string) (any, bool) {
	value, ok := args[escapeParameterName(name)]
	if ok {
		return value, true
	}
	value, ok = args[name]
	return value, ok
}
func formatParameterValue(value any, integer bool) string {
	if integer {
		switch value := value.(type) {
		case float64:
			return fmt.Sprintf("%d", int64(value))
		case float32:
			return fmt.Sprintf("%d", int64(value))
		}
	}
	return fmt.Sprint(value)
}
func baseURLForOperation(baseURL, basePath string) string {
	if basePath == "" {
		return strings.TrimRight(baseURL, "/")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" {
		return strings.TrimRight(baseURL, "/")
	}
	parsed.Path = "/" + strings.Trim(basePath, "/")
	parsed.RawQuery = ""
	return strings.TrimRight(parsed.String(), "/")
}
