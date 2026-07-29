package openapimcp

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// LoadOpenAPISpec loads and validates an OpenAPI YAML or JSON file.
func LoadOpenAPISpec(path string) (*openapi3.T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read OpenAPI spec %q: %w", path, err)
	}
	doc, err := LoadOpenAPISpecFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("load OpenAPI spec %q: %w", path, err)
	}
	return doc, nil
}

func LoadOpenAPISpecFromString(data string) (*openapi3.T, error) {
	return LoadOpenAPISpecFromBytes([]byte(data))
}

func LoadOpenAPISpecFromBytes(data []byte) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("parse OpenAPI spec: %w", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validate OpenAPI spec: %w", err)
	}
	return doc, nil
}

// ExtractOpenAPIOperations returns MCP-enabled operations in deterministic order.
func ExtractOpenAPIOperations(doc *openapi3.T) []OpenAPIOperation {
	if doc == nil || doc.Paths == nil {
		return nil
	}
	paths := make([]string, 0, len(doc.Paths.Map()))
	for path := range doc.Paths.Map() {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var operations []OpenAPIOperation
	for _, path := range paths {
		item := doc.Paths.Find(path)
		if item == nil {
			continue
		}
		for _, method := range []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"} {
			operation := item.GetOperation(strings.ToUpper(method))
			if operation == nil || !mcpEnabled(doc, operation) {
				continue
			}
			operations = append(operations, OpenAPIOperation{
				OperationID: operationID(method, path, operation.OperationID),
				Summary:     operation.Summary,
				Description: operation.Description,
				Path:        path,
				Method:      method,
				Parameters:  mergeParameters(item.Parameters, operation.Parameters),
				RequestBody: operation.RequestBody,
				Tags:        operation.Tags,
				MCPReadOnly: extensionEnabled(operation.Extensions, "x-mcp-read-only"),
				MCPBasePath: extensionString(operation.Extensions, "x-mcp-base-path"),
			})
		}
	}
	return operations
}

func operationID(method, path, id string) string {
	if id != "" {
		return id
	}
	name := strings.Trim(strings.NewReplacer("/", "_", "{", "", "}", "").Replace(path), "_")
	if name == "" {
		return method
	}
	return method + "_" + name
}

func mergeParameters(pathParameters, operationParameters openapi3.Parameters) openapi3.Parameters {
	parameters := append(openapi3.Parameters(nil), pathParameters...)
	indexes := make(map[string]int, len(parameters))
	for index, parameter := range parameters {
		if parameter != nil && parameter.Value != nil {
			indexes[parameterKey(parameter.Value)] = index
		}
	}
	for _, parameter := range operationParameters {
		if parameter == nil || parameter.Value == nil {
			continue
		}
		key := parameterKey(parameter.Value)
		if index, exists := indexes[key]; exists {
			parameters[index] = parameter
			continue
		}
		indexes[key] = len(parameters)
		parameters = append(parameters, parameter)
	}
	return parameters
}

func parameterKey(parameter *openapi3.Parameter) string {
	return parameter.In + "\x00" + parameter.Name
}

func mcpEnabled(doc *openapi3.T, operation *openapi3.Operation) bool {
	if extensionEnabled(operation.Extensions, "x-mcp-disabled") {
		return false
	}
	return !extensionEnabled(doc.Extensions, "x-mcp-curated") || extensionEnabled(operation.Extensions, "x-mcp-enabled")
}

func extensionEnabled(extensions map[string]any, name string) bool {
	enabled, ok := extensions[name].(bool)
	return ok && enabled
}

func extensionString(extensions map[string]any, name string) string {
	value, _ := extensions[name].(string)
	return value
}
