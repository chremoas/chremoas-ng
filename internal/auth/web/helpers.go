package web

import (
	"context"
	"net/http"

	"github.com/antihax/goesi"
)

type key int

const authenticatorKey key = 1
const apiClientKey key = 2

// pull the SSO Authenticator pointer from the context.
func authenticatorFromContext(ctx context.Context) *goesi.SSOAuthenticator {
	return ctx.Value(authenticatorKey).(*goesi.SSOAuthenticator)
}

// Add SSO Authenticator pointer to the context.
func contextWithAuthenticator(ctx context.Context, a *goesi.SSOAuthenticator) context.Context {
	return context.WithValue(ctx, authenticatorKey, a)
}

// pull the API Client pointer from the context.
func apiClientFromContext(ctx context.Context) *goesi.APIClient {
	return ctx.Value(apiClientKey).(*goesi.APIClient)
}

// Add API Client pointer to the context.
func contextWithAPIClient(ctx context.Context, a *goesi.APIClient) context.Context {
	return context.WithValue(ctx, apiClientKey, a)
}

// Add custom middleware for SSO Authenticator
func middleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ctx := contextWithAuthenticator(req.Context(), authenticator)
		ctx = contextWithAPIClient(ctx, apiClient)
		next.ServeHTTP(rw, req.WithContext(ctx))
	})
}
