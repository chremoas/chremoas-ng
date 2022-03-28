package spiffylogger

import (
	"context"
	"log"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewCtxWithLogger is a wrapper around NewLogger that returns a context because that's
// generally what you want to do anyway.
func NewCtxWithLogger(level zapcore.Level, options ...zap.Option) context.Context {
	return CtxWithLogger(context.Background(), NewLogger(level, options...))
}

// NewLogger sets up a new logger for us to use
func NewLogger(level zapcore.Level, options ...zap.Option) *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.Level.SetLevel(level)
	cfg.EncoderConfig = zap.NewProductionEncoderConfig()
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = func(t time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(t.Format(time.RFC3339Nano))
	}

	opts := []zap.Option{
		zap.AddCallerSkip(2),
	}
	opts = append(opts, options...)

	zapLogger, err := cfg.Build(opts...)
	if err != nil {
		log.Fatalf("err creating logger: %v\n", err.Error())
	}

	return zapLogger
}

// ZapFields converts a LogLine to a slice of zapcore.Field.
//
// Zap already has built in fields for these log line information:
// - ll.Timestamp	=> ts
// - ll.File		=> caller
// - ll.LineNumber	=> caller
func (ll LogLine) ZapFields(duration int64) []zapcore.Field {
	zapFields := []zapcore.Field{
		zap.String("name", ll.Name),
		zap.String("correlation_id", ll.CorrelationID),
		zap.String("span_id", ll.SpanID),
		zap.Int64("duration", duration),
	}

	return append(zapFields, ll.Fields...)
}
