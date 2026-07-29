package openapi2mcp

import "testing"

func TestGetStreamableHTTPURL(t *testing.T) {
	if got := GetStreamableHTTPURL(":8080", "/weather"); got != "http://localhost:8080/weather" {
		t.Fatalf("unexpected URL: %s", got)
	}
}
