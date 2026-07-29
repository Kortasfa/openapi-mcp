package openapi2mcp

import "github.com/getkin/kin-openapi/openapi3"

func getContentByType(content openapi3.Content, contentType string) *openapi3.MediaType {
	if mediaType := content[contentType]; mediaType != nil {
		return mediaType
	}
	for name, mediaType := range content {
		if mediaType != nil && len(name) >= len(contentType) && name[:len(contentType)] == contentType {
			return mediaType
		}
	}
	return nil
}
