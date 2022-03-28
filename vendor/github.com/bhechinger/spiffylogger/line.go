package spiffylogger

import (
	"fmt"
	"time"

	"github.com/go-stack/stack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogLine represents a single log line
type LogLine struct {
	Level         zapcore.Level `json:"lev"`
	Timestamp     string        `json:"tim"`
	Name          string        `json:"nam"`
	CorrelationID string        `json:"cid"`
	SpanID        string        `json:"sid"`
	File          string        `json:"fil"`
	LineNumber    string        `json:"lin"`
	Message       string        `json:"msg"`
	Fields        []zap.Field   `json:"fie"`
}

// NewLine populates a log Line with values and returns it.
func NewLine(lev zapcore.Level, s *Span, msg string, c *stack.Call, fields ...zap.Field) *LogLine {
	return &LogLine{
		Level:         lev,
		Name:          s.name,
		CorrelationID: s.cID,
		SpanID:        s.sID,
		File:          fmt.Sprintf("%#s", c),
		LineNumber:    fmt.Sprintf("%d", c),
		Message:       msg,
		Timestamp:     time.Now().String(),
		Fields:        fields,
	}
}
