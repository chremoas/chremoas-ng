package spiffylogger

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type loggerKey struct{}

// loggerFromContext pulls a logger from a context
func loggerFromContext(ctx context.Context) (l *zap.Logger, found bool) {
	value, ok := ctx.Value(loggerKey{}).(*zap.Logger)
	if ok {
		return value, true
	}
	return nil, false
}

// CtxWithLogger allows for injecting a logger into a context
func CtxWithLogger(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

// LogServer is intended to be implemented by all servers that want to inject a logger into the context before calling endpoint handlers
type LogServer interface {
	Logger() *zap.Logger
}

// LogInterceptor is used to inject a logger into the context
// This injector should be called FIRST, so that other injectors have logging capabilities
func LogInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	ls, ok := (info.Server).(LogServer)
	// If we're not dealing with a server with an injected logger, we have nothing to do: let the endpoint handle it (likely with a default, or error)
	if ok {
		ctx = CtxWithLogger(ctx, ls.Logger())
	}
	return handler(ctx, req)
}
