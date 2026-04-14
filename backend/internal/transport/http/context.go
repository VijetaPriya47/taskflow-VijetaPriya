package httpapi

import "context"

type ctxKey string

const (
	ctxKeyRequestID ctxKey = "request_id"
	ctxKeyUserID    ctxKey = "user_id"
	ctxKeyUserEmail ctxKey = "user_email"
	ctxKeyReqInfo   ctxKey = "req_info"
)

type requestInfo struct {
	RequestID string
	UserID    string
	UserEmail string
}

func withRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, requestID)
}

func requestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyRequestID).(string)
	return v
}

func withUser(ctx context.Context, userID, email string) context.Context {
	ctx = context.WithValue(ctx, ctxKeyUserID, userID)
	ctx = context.WithValue(ctx, ctxKeyUserEmail, email)
	return ctx
}

func userFromContext(ctx context.Context) (userID, email string, ok bool) {
	userID, _ = ctx.Value(ctxKeyUserID).(string)
	email, _ = ctx.Value(ctxKeyUserEmail).(string)
	return userID, email, userID != ""
}

func withRequestInfo(ctx context.Context, info *requestInfo) context.Context {
	return context.WithValue(ctx, ctxKeyReqInfo, info)
}

func requestInfoFromContext(ctx context.Context) *requestInfo {
	v, _ := ctx.Value(ctxKeyReqInfo).(*requestInfo)
	return v
}
