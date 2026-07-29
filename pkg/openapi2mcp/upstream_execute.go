package openapi2mcp

import (
	"context"
	"io"
	"net/http"

	"github.com/ubermorgenland/openapi-mcp/pkg/auth"
)

type upstreamResponse struct {
	Response *http.Response
	Body     []byte
}

func executeUpstreamRequest(ctx context.Context, request *http.Request, authorization string) (*upstreamResponse, error) {
	authRequest := request
	if authorization != "" {
		authRequest = request.Clone(ctx)
		authRequest.Header.Set("Authorization", authorization)
	}
	authContext := auth.CreateBearerAuthContext(authRequest)
	requestWithAuth := request.WithContext(auth.WithAuthContext(ctx, authContext))
	authProvider := auth.NewSecureAuthProvider()
	response, err := auth.NewSecureHTTPClientWrapper(http.DefaultClient, authProvider).Do(requestWithAuth)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return &upstreamResponse{Response: response, Body: body}, nil
}
