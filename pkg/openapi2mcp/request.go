package openapi2mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

type upstreamRequest struct {
	Request *http.Request
	Body    []byte
	URL     string
}

func buildUpstreamRequest(ctx context.Context, doc *openapi3.T, baseURLs []string, operation OpenAPIOperation, args map[string]any) (*upstreamRequest, error) {
	if len(baseURLs) == 0 {
		return nil, fmt.Errorf("at least one upstream base URL is required")

	}
	path := operation.Path
	query := url.Values{}
	for _, parameterRef := range operation.Parameters {
		if parameterRef == nil || parameterRef.Value == nil {
			continue
		}
		parameter := parameterRef.Value
		value, present := getParameterValue(args, parameter.Name)
		if !present {
			continue
		}
		integer := parameter.Schema != nil && parameter.Schema.Value != nil && parameter.Schema.Value.Type != nil && parameter.Schema.Value.Type.Is("integer")
		switch parameter.In {
		case "path":
			path = strings.ReplaceAll(path, "{"+parameter.Name+"}", formatParameterValue(value, integer))
		case "query":
			addQueryParameter(query, parameter.Name, value, integer)
		}
	}

	baseURL := baseURLForOperation(baseURLs[rand.Intn(len(baseURLs))], operation.MCPBasePath)
	fullURL := baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	body, contentType, err := marshalRequestBody(operation.RequestBody, args)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, strings.ToUpper(operation.Method), fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if len(body) > 0 && contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	request.Header.Set("Accept", "application/json, application/vnd.api+json")
	applyHeaderParameters(request, doc, operation.Parameters, args)
	applyCookieParameters(request, operation.Parameters, args)
	return &upstreamRequest{Request: request, Body: body, URL: fullURL}, nil
}

func addQueryParameter(query url.Values, name string, value any, integer bool) {
	if values, ok := value.([]any); ok {
		for _, item := range values {
			query.Add(name, formatParameterValue(item, integer))
		}
		return
	}
	query.Set(name, formatParameterValue(value, integer))
}

func marshalRequestBody(requestBody *openapi3.RequestBodyRef, args map[string]any) ([]byte, string, error) {
	if requestBody == nil || requestBody.Value == nil {
		return nil, "", nil
	}
	mediaType := getContentByType(requestBody.Value.Content, "application/json")
	contentType := "application/json"
	if mediaType == nil {
		mediaType = getContentByType(requestBody.Value.Content, "application/vnd.api+json")
		contentType = "application/vnd.api+json"
	}
	if mediaType == nil || mediaType.Schema == nil || mediaType.Schema.Value == nil {
		return nil, "", nil
	}
	body, present := args["requestBody"]
	if !present || body == nil {
		return nil, contentType, nil
	}
	encoded, err := json.Marshal(body)
	return encoded, contentType, err
}

func applyHeaderParameters(request *http.Request, doc *openapi3.T, parameters openapi3.Parameters, args map[string]any) {
	headerNames := make(map[string]struct{})
	for _, parameterRef := range parameters {
		if parameterRef == nil || parameterRef.Value == nil || parameterRef.Value.In != "header" {
			continue
		}
		parameter := parameterRef.Value
		headerNames[parameter.Name] = struct{}{}
		if value, present := getParameterValue(args, parameter.Name); present {
			integer := parameter.Schema != nil && parameter.Schema.Value != nil && parameter.Schema.Value.Type != nil && parameter.Schema.Value.Type.Is("integer")
			request.Header.Set(parameter.Name, formatParameterValue(value, integer))
		}
	}
	if doc != nil && doc.Components != nil {
		for _, securityRef := range doc.Components.SecuritySchemes {
			if securityRef != nil && securityRef.Value != nil && securityRef.Value.Type == "apiKey" && securityRef.Value.In == "header" {
				headerNames[securityRef.Value.Name] = struct{}{}
			}
		}
	}
	for argumentName, value := range args {
		if _, isHeader := headerNames[argumentName]; isHeader {
			request.Header.Set(argumentName, fmt.Sprint(value))
		}
	}
}

func applyCookieParameters(request *http.Request, parameters openapi3.Parameters, args map[string]any) {
	var pairs []string
	for _, parameterRef := range parameters {
		if parameterRef == nil || parameterRef.Value == nil || parameterRef.Value.In != "cookie" {
			continue
		}
		parameter := parameterRef.Value
		value, present := getParameterValue(args, parameter.Name)
		if !present {
			continue
		}
		integer := parameter.Schema != nil && parameter.Schema.Value != nil && parameter.Schema.Value.Type != nil && parameter.Schema.Value.Type.Is("integer")
		pairs = append(pairs, fmt.Sprintf("%s=%s", parameter.Name, formatParameterValue(value, integer)))
	}
	if len(pairs) > 0 {
		request.Header.Set("Cookie", strings.Join(pairs, "; "))
	}
}
