package openapi2mcp

import (
	"reflect"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func typesPtr(types ...string) *openapi3.Types {
	value := openapi3.Types(types)
	return &value
}

func schemaRef(schema *openapi3.Schema) *openapi3.SchemaRef {
	return &openapi3.SchemaRef{Value: schema}
}

func TestBuildInputSchemaIncludesRequiredParameters(t *testing.T) {
	schema := BuildInputSchema(openapi3.Parameters{&openapi3.ParameterRef{Value: &openapi3.Parameter{
		Name:     "userId",
		In:       "path",
		Required: true,
		Schema:   schemaRef(&openapi3.Schema{Type: typesPtr("string")}),
	}}}, nil)

	properties := schema["properties"].(map[string]any)
	if properties["userId"].(map[string]any)["type"] != "string" {
		t.Fatalf("unexpected parameter schema: %#v", properties["userId"])
	}
	if !reflect.DeepEqual(schema["required"], []string{"userId"}) {
		t.Fatalf("unexpected required fields: %#v", schema["required"])
	}
}

func TestBuildInputSchemaMarksOptionalQueryParametersAsFilters(t *testing.T) {
	schema := BuildInputSchema(openapi3.Parameters{&openapi3.ParameterRef{Value: &openapi3.Parameter{
		Name:   "departmentId",
		In:     "query",
		Schema: schemaRef(&openapi3.Schema{Type: typesPtr("string")}),
	}}}, nil)

	property := schema["properties"].(map[string]any)["departmentId"].(map[string]any)
	if property["description"] != "Optional query filter. Omit it to return unfiltered results." {
		t.Fatalf("unexpected filter description: %#v", property["description"])
	}
}

func TestBuildInputSchemaEscapesBracketedParameterNames(t *testing.T) {
	schema := BuildInputSchema(openapi3.Parameters{&openapi3.ParameterRef{Value: &openapi3.Parameter{
		Name:     "filter[department]",
		In:       "query",
		Required: true,
		Schema:   schemaRef(&openapi3.Schema{Type: typesPtr("string")}),
	}}}, nil)

	if _, exists := schema["properties"].(map[string]any)["filter_department_"]; !exists {
		t.Fatalf("escaped property is absent: %#v", schema["properties"])
	}
}

func TestBuildInputSchemaExcludesConfiguredAuthenticationHeader(t *testing.T) {
	doc := &openapi3.T{Components: &openapi3.Components{SecuritySchemes: openapi3.SecuritySchemes{
		"bearerAuth": &openapi3.SecuritySchemeRef{Value: &openapi3.SecurityScheme{Type: "http", Scheme: "bearer"}},
	}}}
	parameters := openapi3.Parameters{&openapi3.ParameterRef{Value: &openapi3.Parameter{
		Name:     "Authorization",
		In:       "header",
		Required: true,
		Schema:   schemaRef(&openapi3.Schema{Type: typesPtr("string")}),
	}}}

	schema := BuildInputSchemaWithContext(parameters, nil, doc)
	if len(schema["properties"].(map[string]any)) != 0 {
		t.Fatalf("authentication header must not be an MCP argument: %#v", schema)
	}
}

func TestBuildInputSchemaConvertsJSONRequestBody(t *testing.T) {
	body := &openapi3.RequestBodyRef{Value: &openapi3.RequestBody{
		Required: true,
		Content: openapi3.Content{"application/json": &openapi3.MediaType{Schema: schemaRef(&openapi3.Schema{
			Type: typesPtr("object"),
			Properties: openapi3.Schemas{
				"name": schemaRef(&openapi3.Schema{Type: typesPtr("string")}),
			},
			Required: []string{"name"},
		})}},
	}}
	schema := BuildInputSchema(nil, body)

	requestBody := schema["properties"].(map[string]any)["requestBody"].(map[string]any)
	if requestBody["properties"].(map[string]any)["name"].(map[string]any)["type"] != "string" {
		t.Fatalf("unexpected request body schema: %#v", requestBody)
	}
	if !reflect.DeepEqual(schema["required"], []string{"requestBody"}) {
		t.Fatalf("unexpected required fields: %#v", schema["required"])
	}
}

func TestSchemaForPreservesCompositions(t *testing.T) {
	value := schemaFor(schemaRef(&openapi3.Schema{
		OneOf: openapi3.SchemaRefs{
			schemaRef(&openapi3.Schema{Type: typesPtr("string")}),
			schemaRef(&openapi3.Schema{Type: typesPtr("integer")}),
		},
	}), make(map[*openapi3.Schema]bool))

	variants, ok := value["oneOf"].([]any)
	if !ok || len(variants) != 2 {
		t.Fatalf("unexpected oneOf: %#v", value["oneOf"])
	}
	if variants[0].(map[string]any)["type"] != "string" || variants[1].(map[string]any)["type"] != "integer" {
		t.Fatalf("unexpected variants: %#v", variants)
	}
}
