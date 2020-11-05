package context

import (
	gtx "context"

	"github.com/google/uuid"
)

type keyID struct{}

func ID(ctx gtx.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(keyID{}); v != nil {
		return v.(string)
	}
	return ""
}

func WithID(ctx gtx.Context, id string) gtx.Context {
	return gtx.WithValue(ctx, keyID{}, id)
}

func NewID(ctx gtx.Context) gtx.Context {
	return WithID(ctx, uuid.New().String())
}
