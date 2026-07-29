package openapimcp

import (
	"context"
	"io"
	"net/http"
)

type upstreamResponse struct {
	Response *http.Response
	Body     []byte
}

func executeUpstreamRequest(ctx context.Context, request *http.Request, authorization string) (*upstreamResponse, error) {
	upstreamRequest := request
	if authorization != "" {
		upstreamRequest = request.Clone(ctx)
		upstreamRequest.Header.Set("Authorization", authorization)
	}
	response, err := http.DefaultClient.Do(upstreamRequest)
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
