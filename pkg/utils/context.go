package utils

import (
	"context"
	"errors"
	"net/http"

	"jusha/mcp/pkg/model"

	httptransport "github.com/go-kit/kit/transport/http"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

// ContextKeyUser is the key used to store user information in the context.
const ContextKeyUser contextKey = "x-user"

func GetUserIDFromContext(ctx context.Context) (int64, error) {
	user := ctx.Value(ContextKeyUser).(*model.User)
	if user == nil {
		return -1, errors.New("user not found in context")
	}
	return user.UserId, nil
}

// UserContextServerBefore 是 go-kit 的 ServerBefore，用于在 http handler 层注入 user 到 context
func UserContextServerBefore() httptransport.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		user, err := GetUserFromHeader(r)
		if err == nil && user != nil {
			ctx = context.WithValue(ctx, ContextKeyUser, user)
		}
		return ctx
	}
}

// PopulateUserContext extracts user information from the request header and adds it to the context.
// This can be used by both Go-Kit ServerBefore and custom handlers.
func PopulateUserContext(ctx context.Context, r *http.Request) context.Context {
	// Assuming utils.GetUserFromHeader returns the user as a string and an error
	user, err := GetUserFromHeader(r) // Adjust if GetUserFromHeader returns a different type
	if err == nil && user != nil {    // Check for non-empty user
		ctx = context.WithValue(ctx, ContextKeyUser, user)
	}
	// Consider logging the error if err != nil and userID == ""
	return ctx
}
