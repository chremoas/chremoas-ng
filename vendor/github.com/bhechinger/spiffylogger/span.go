package spiffylogger

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/go-stack/stack"
	"github.com/pkg/errors"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type (
	spanKey struct{}
)

// Span is our implementation of a Spanner
type Span struct {
	name   string
	start  time.Time
	cID    string
	sID    string
	logger *zap.Logger
	fields []zap.Field
}

// OpenSpan configures and returns a Span from a context, creating a child span if one exists in the current context
func OpenSpan(ctx context.Context) (context.Context, *Span) {
	caller := "unknown"
	pc, _, _, ok := runtime.Caller(1)
	if ok {
		d := runtime.FuncForPC(pc)
		if d != nil {
			n := strings.Split(d.Name(), "/")
			caller = n[len(n)-1] // get just the filename + function for our span's name
		}
	}
	return openNamedSpan(ctx, caller, 1, "")
}

// OpenCorrelatedSpan configures and returns a span from a context but uses a provided Correlation ID
// instead of generating one.
func OpenCorrelatedSpan(ctx context.Context, correlationID string) (context.Context, *Span) {
	caller := "unknown"
	pc, _, _, ok := runtime.Caller(1)
	if ok {
		d := runtime.FuncForPC(pc)
		if d != nil {
			n := strings.Split(d.Name(), "/")
			caller = n[len(n)-1] // get just the filename + function for our span's name
		}
	}
	return openNamedSpan(ctx, caller, 1, correlationID)
}

// OpenCustomSpan configures and returns a Span from a context, creating a child span if one exists in the current context
// "custom" only if we want a custom name for this span
func OpenCustomSpan(ctx context.Context, name string) (context.Context, *Span) {
	return openNamedSpan(ctx, name, 1, "")
}

// openNamedSpan contains the common code for OpenSpan and OpenCustomSpan
// with the appropriate log depth of 3
func openNamedSpan(ctx context.Context, name string, depth int, correlationID string) (context.Context, *Span) {
	depth++
	var newSpan *Span
	s, ok := spanFromContext(ctx)
	if ok && correlationID == "" {
		newSpan = openChildSpan(s, name, depth)
	} else {
		l, ok := loggerFromContext(ctx)
		if !ok {
			// if we don't get a logger, make sure we're at least logging to stderr
			l = NewLogger(zapcore.InfoLevel)
		}
		newSpan = openNewSpan(name, l, depth, correlationID)
		if !ok {
			newSpan.printToLog(zapcore.InfoLevel, "failed to find logger in context; defaulting to stderr logger", depth)
		}
	}

	return CtxWithSpan(ctx, newSpan), newSpan
}

// openNew returns a child span of this span, keeping the same context and CID
func openChildSpan(s *Span, childName string, depth int) *Span {
	depth++
	ns := &Span{
		name:   fmt.Sprintf("%s|%s", s.name, childName), // semi-stacktrace naming
		start:  time.Now(),
		cID:    s.cID,
		logger: s.logger,
	}
	ns.sID = ns.newID(depth)

	// ns.printToLog(zapcore.DebugLevel, "span opened (child)", depth)
	return ns
}

// spanFromContext pulls a span out of a context
func spanFromContext(ctx context.Context) (s *Span, found bool) {
	key, ok := ctx.Value(spanKey{}).(*Span)
	if ok {
		return key, true
	}
	return nil, false
}

// CtxWithSpan allows for injecting a span into a context
func CtxWithSpan(ctx context.Context, s *Span) context.Context {
	return context.WithValue(ctx, spanKey{}, s)
}

// openNew returns a brand new span with a new CID
func openNewSpan(name string, l *zap.Logger, depth int, correlationID string) *Span {
	depth++
	s := &Span{
		name:   name,
		start:  time.Now(),
		logger: l,
	}

	if correlationID == "" {
		s.cID = s.newID(depth)
	} else {
		s.cID = correlationID
	}

	s.sID = s.newID(depth)
	// s.printToLog(zapcore.DebugLevel, "span opened", 1)
	return s
}

func (s *Span) newID(depth int) string {
	depth++
	id, err := ksuid.NewRandom()
	if err != nil {
		s.printToLog(zapcore.ErrorLevel, errors.Wrap(err, "Failed to generate id.").Error(), depth)
		return "ERRID"
	}
	return id.String()
}

// GetCorrelationID ...
func (s Span) GetCorrelationID() string {
	return s.cID
}

// Close .
func (s *Span) Close() {
	// TODO: close/end OT span
	// TODO: add timing metric to OT
	// s.printToLog(zapcore.DebugLevel, "span closed", 1)
}

// This is neat but unfortunately it breaks the zap interface expectations
// // Error .
// func (s *Span) Error(err error, fields ...zapcore.Field) {
// 	if s.ll.Level >= zapcore.ErrorLevel {
// 		s.printToLog(zapcore.ErrorLevel, withStacktrace(err), 1, fields...)
// 	}
// }
//
// func withStacktrace(err error) string {
// 	// %+v gives us the error message plus a full stack trace for the error, as long as it was constructed with the "github.com/pkg/errors" package
// 	// we should strive to use `errors.New`, `errors.Errorf`, and `errors.Wrap` wherever we create a new error or get one from an external source
// 	return fmt.Sprintf("%+v", err)
// }

// Error .
func (s *Span) Error(msg string, fields ...zapcore.Field) {
	s.printToLog(zapcore.ErrorLevel, msg, 1, fields...)
}

// Warn .
func (s *Span) Warn(msg string, fields ...zapcore.Field) {
	s.printToLog(zapcore.WarnLevel, msg, 1, fields...)
}

// Info .
func (s *Span) Info(msg string, fields ...zapcore.Field) {
	s.printToLog(zapcore.InfoLevel, msg, 1, fields...)
}

// Debug .
func (s *Span) Debug(msg string, fields ...zapcore.Field) {
	s.printToLog(zapcore.DebugLevel, msg, 1, fields...)
}

// printToLog is solely responsible for creating log lines and printing them to the logger
//
// NOTE about log levels: we want to check levels before calling this function
// to avoid string cacentation functions being called needlessly
//
// NOTE: we want to use printToLog explicitly in our logging functions to ensure the caller is captured correctly (exactly 2 function callers away)
//
// NOTE: depth is relative to the calls in this package. We always want depth to be equal to the call of these functions.
// Therefore, its important to be careful to not call spans's public-facing functions inside of span.
// Instead, each internal function should accept a depth value, and +1 that value for its own call.
func (s *Span) printToLog(level zapcore.Level, msg string, depth int, fields ...zapcore.Field) {
	depth++
	c := stack.Caller(depth)
	fields = append(fields, s.fields...) // Append our With() fields
	ll := NewLine(level, s, msg, &c, fields...)
	dur := time.Since(s.start).Nanoseconds()
	switch level {
	case zapcore.ErrorLevel:
		s.logger.Error(msg, ll.ZapFields(dur)...)
	case zapcore.WarnLevel:
		s.logger.Warn(msg, ll.ZapFields(dur)...)
	case zapcore.InfoLevel:
		s.logger.Info(msg, ll.ZapFields(dur)...)
	case zapcore.DebugLevel:
		s.logger.Debug(msg, ll.ZapFields(dur)...)
	}
}

func (s *Span) With(fields ...zap.Field) *Span {
	s.fields = append(s.fields, fields...)
	return s
}

func NewID() string {
	id, err := ksuid.NewRandom()
	if err != nil {
		return "ERRID"
	}
	return id.String()
}
