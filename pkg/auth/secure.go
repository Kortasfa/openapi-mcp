package auth

import "context"

type SecureAuthProvider interface {
	GetAuthHeaders(ctx context.Context) map[string]string
}

type contextAuthProvider struct{}

func NewSecureAuthProvider() SecureAuthProvider {
	return &contextAuthProvider{}
}

func (p *contextAuthProvider) GetAuthHeaders(ctx context.Context) map[string]string {
	authCtx, ok := FromContext(ctx)
	if !ok || authCtx.Token == "" || authCtx.AuthType != "bearer" {
		return nil
	}
	return map[string]string{"Authorization": "Bearer " + authCtx.Token}
}
