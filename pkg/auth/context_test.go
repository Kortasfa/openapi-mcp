package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateBearerAuthContextReadsOnlyRequestHeader(t *testing.T) {
	t.Setenv("BEARER_TOKEN", "server-token")
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	request.Header.Set("Authorization", "Bearer client-token")

	if got := CreateBearerAuthContext(request).Token; got != "client-token" {
		t.Fatalf("token = %q, want client token", got)
	}

	request.Header.Del("Authorization")
	if got := CreateBearerAuthContext(request).Token; got != "" {
		t.Fatalf("token = %q, want empty token", got)
	}
}
