package context

import (
	gtx "context"
)

type keyID struct{}

func WithID(ctx gtx.Context, id string) gtx.Context {
	return gtx.WithValue(ctx, keyID{}, id)
}

func ID(ctx gtx.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(keyID{}); v != nil {
		return v.(string)
	}
	return ""
}
