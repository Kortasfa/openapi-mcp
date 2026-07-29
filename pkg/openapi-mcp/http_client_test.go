package openapimcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExecuteUpstreamRequestForwardsBearerToken(t *testing.T) {
	var authorization string
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		authorization = request.Header.Get("Authorization")
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := executeUpstreamRequest(context.Background(), request, "Bearer client-token")
	if err != nil {
		t.Fatal(err)
	}
	if authorization != "Bearer client-token" {
		t.Fatalf("Authorization = %q", authorization)
	}
	if result.Response.StatusCode != http.StatusOK || string(result.Body) != `{"ok":true}` {
		t.Fatalf("unexpected response: %d %s", result.Response.StatusCode, result.Body)
	}
}
