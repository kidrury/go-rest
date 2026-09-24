package auth

import (
	"context"
)

type identityContextKey string

var identityKey identityContextKey = "identity_key"

type Identity struct {
	UserID string
}

func WithIdentity(ctx context.Context, userID string) context.Context {
	var identity = Identity{
		UserID: userID,
	}
	return context.WithValue(ctx, identityKey, identity)
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey).(Identity)
	return identity, ok
}
