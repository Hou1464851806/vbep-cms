package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type OutputMode int

const (
	OutputModeJSON OutputMode = iota
	OutputModeConsole
)

const (
	fieldCaller  = "caller"
	fieldSource  = "source"
	fieldService = "service"
)

type structuredCore struct {
	base    zapcore.Core
	mode    OutputMode
	context []zap.Field
}

func NewStructuredCore(base zapcore.Core, mode OutputMode) zapcore.Core {
	return &structuredCore{base: base, mode: mode}
}

func (c *structuredCore) Enabled(level zapcore.Level) bool {
	return c.base.Enabled(level)
}

func (c *structuredCore) With(fields []zap.Field) zapcore.Core {
	clone := *c
	clone.context = append(append([]zap.Field{}, c.context...), fields...)
	return &clone
}

func (c *structuredCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c *structuredCore) Write(entry zapcore.Entry, fields []zap.Field) error {
	allFields := append(append([]zap.Field{}, c.context...), fields...)
	entry.Caller = resolveEntryCaller(allFields, entry.Caller)
	allFields = stripField(allFields, fieldCaller)

	switch c.mode {
	case OutputModeConsole:
		entry.Message = formatConsoleMessage(entry.Message, allFields)
		allFields = nil
	}

	return c.base.Write(entry, allFields)
}

func (c *structuredCore) Sync() error {
	return c.base.Sync()
}

func resolveEntryCaller(fields []zap.Field, fallback zapcore.EntryCaller) zapcore.EntryCaller {
	for _, field := range fields {
		if field.Key != fieldCaller {
			continue
		}
		value, ok := fieldValue(field)
		if !ok {
			break
		}
		return zapcore.EntryCaller{Defined: true, File: fmt.Sprint(value)}
	}
	return fallback
}

func stripField(fields []zap.Field, key string) []zap.Field {
	result := fields[:0]
	for _, field := range fields {
		if field.Key == key {
			continue
		}
		result = append(result, field)
	}
	return result
}

func formatConsoleMessage(msg string, fields []zap.Field) string {
	if len(fields) == 0 {
		return msg
	}

	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		value, ok := fieldValue(field)
		if !ok {
			continue
		}
		parts = append(parts, formatKeyValue(field.Key, value))
	}

	sort.Strings(parts)
	if len(parts) == 0 {
		return msg
	}
	if msg == "" {
		return strings.Join(parts, " ")
	}
	return msg + " " + strings.Join(parts, " ")
}

func formatKeyValue(key string, value interface{}) string {
	switch v := value.(type) {
	case string:
		if v == "" {
			return key + `=""`
		}
		if needsQuotes(v) {
			return key + "=" + strconv.Quote(v)
		}
		return key + "=" + v
	case time.Duration:
		return key + "=" + strconv.Quote(v.String())
	case time.Time:
		return key + "=" + strconv.Quote(v.Format(time.RFC3339Nano))
	case error:
		return key + "=" + strconv.Quote(v.Error())
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return key + "=" + strconv.Quote(fmt.Sprint(v))
		}
		if bytes.IndexFunc(raw, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n'
		}) >= 0 {
			return key + "=" + strconv.Quote(string(raw))
		}
		return key + "=" + string(raw)
	}
}

func needsQuotes(v string) bool {
	return strings.ContainsAny(v, " \t\n\r=\"")
}

func fieldValue(field zap.Field) (interface{}, bool) {
	enc := zapcore.NewMapObjectEncoder()
	field.AddTo(enc)
	value, ok := enc.Fields[field.Key]
	return value, ok
}
