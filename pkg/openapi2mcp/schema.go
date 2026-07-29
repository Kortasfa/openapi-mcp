package openapi2mcp

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// escapeParameterName makes bracketed OpenAPI parameter names valid MCP fields.
func escapeParameterName(name string) string {
	return strings.NewReplacer("[", "_", "]", "_").Replace(name)
}

// buildParameterNameMapping is retained for the upstream request builder. Parameter
// lookup derives the escaped name directly, so no mapping is necessary.
func buildParameterNameMapping(openapi3.Parameters) map[string]string {
	return nil
}

func isAuthenticationHeader(parameter *openapi3.Parameter, doc *openapi3.T) bool {
	if parameter.In != "header" || doc == nil || doc.Components == nil {
		return false
	}
	for _, security := range doc.Components.SecuritySchemes {
		if security == nil || security.Value == nil {
			continue
		}
		scheme := security.Value
		if scheme.Type == "http" && strings.EqualFold(parameter.Name, "Authorization") {
			return true
		}
		if scheme.Type == "apiKey" && scheme.In == "header" && strings.EqualFold(scheme.Name, parameter.Name) {
			return true
		}
	}
	return false
}

// schemaFor converts the OpenAPI schema subset accepted by the JSON Schema tool validator.
// References have already been resolved by kin-openapi while loading the document.
func schemaFor(reference *openapi3.SchemaRef, visiting map[*openapi3.Schema]bool) map[string]any {
	if reference == nil || reference.Value == nil {
		return map[string]any{}
	}
	schema := reference.Value
	if visiting[schema] {
		return map[string]any{"type": "object"}
	}
	visiting[schema] = true
	defer delete(visiting, schema)

	result := map[string]any{}
	if types := schema.Type.Slice(); len(types) == 1 {
		result["type"] = types[0]
	} else if len(types) > 1 {
		result["type"] = types
	}
	copySchemaMetadata(result, schema)
	addCompositions(result, "allOf", schema.AllOf, visiting)
	addCompositions(result, "anyOf", schema.AnyOf, visiting)
	addCompositions(result, "oneOf", schema.OneOf, visiting)

	if len(schema.Properties) > 0 {
		properties := make(map[string]any, len(schema.Properties))
		for name, property := range schema.Properties {
			properties[name] = schemaFor(property, visiting)
		}
		result["properties"] = properties
		if len(schema.Required) > 0 {
			result["required"] = schema.Required
		}
	}
	if schema.Items != nil {
		result["items"] = schemaFor(schema.Items, visiting)
	}
	if schema.AdditionalProperties.Has != nil {
		result["additionalProperties"] = *schema.AdditionalProperties.Has
	} else if schema.AdditionalProperties.Schema != nil {
		result["additionalProperties"] = schemaFor(schema.AdditionalProperties.Schema, visiting)
	}
	return result
}

func copySchemaMetadata(result map[string]any, schema *openapi3.Schema) {
	if schema.Format != "" {
		result["format"] = schema.Format
	}
	if schema.Description != "" {
		result["description"] = schema.Description
	}
	if len(schema.Enum) > 0 {
		result["enum"] = schema.Enum
	}
	if schema.Default != nil {
		result["default"] = schema.Default
	}
	if schema.Example != nil {
		result["examples"] = []any{schema.Example}
	}
}

func addCompositions(result map[string]any, keyword string, references openapi3.SchemaRefs, visiting map[*openapi3.Schema]bool) {
	if len(references) == 0 {
		return
	}
	schemas := make([]any, 0, len(references))
	for _, reference := range references {
		schemas = append(schemas, schemaFor(reference, visiting))
	}
	result[keyword] = schemas
}

// BuildInputSchema converts OpenAPI parameters and a JSON request body to MCP input schema.
func BuildInputSchema(parameters openapi3.Parameters, requestBody *openapi3.RequestBodyRef) map[string]any {
	return BuildInputSchemaWithContext(parameters, requestBody, nil)
}

func BuildInputSchemaWithContext(parameters openapi3.Parameters, requestBody *openapi3.RequestBodyRef, doc *openapi3.T) map[string]any {
	properties := make(map[string]any)
	var required []string
	for _, reference := range parameters {
		if reference == nil || reference.Value == nil {
			continue
		}
		parameter := reference.Value
		if parameter.Schema == nil || isAuthenticationHeader(parameter, doc) {
			continue
		}
		property := schemaFor(parameter.Schema, make(map[*openapi3.Schema]bool))
		if parameter.Description != "" {
			property["description"] = parameter.Description
		} else if parameter.In == "query" && !parameter.Required {
			property["description"] = "Optional query filter. Omit it to return unfiltered results."
		}
		name := escapeParameterName(parameter.Name)
		properties[name] = property
		if parameter.Required {
			required = append(required, name)
		}
	}

	if requestBody != nil && requestBody.Value != nil {
		if mediaType := getContentByType(requestBody.Value.Content, "application/json"); mediaType != nil && mediaType.Schema != nil {
			properties["requestBody"] = schemaFor(mediaType.Schema, make(map[*openapi3.Schema]bool))
			if requestBody.Value.Required {
				required = append(required, "requestBody")
			}
		}
	}
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}
