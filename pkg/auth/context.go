package auth

import (
	"context"
	"net/http"
	"strings"
)

type AuthContext struct {
	Token           string
	AuthType        string
	OriginalRequest *http.Request
}

type contextKey string

const authContextKey contextKey = "auth"

func CreateBearerAuthContext(request *http.Request) *AuthContext {
	authCtx := &AuthContext{AuthType: "bearer", OriginalRequest: request}
	if request == nil {
		return authCtx
	}

	authorization := request.Header.Get("Authorization")
	if strings.HasPrefix(authorization, "Bearer ") {
		authCtx.Token = strings.TrimPrefix(authorization, "Bearer ")
	}
	return authCtx
}

func WithAuthContext(ctx context.Context, authCtx *AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey, authCtx)
}

func FromContext(ctx context.Context) (*AuthContext, bool) {
	authCtx, ok := ctx.Value(authContextKey).(*AuthContext)
	return authCtx, ok
}
