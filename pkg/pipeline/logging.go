package pipeline

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity level of log messages
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Field represents a structured logging field
type Field struct {
	Key   string
	Value interface{}
}

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an integer field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a boolean field
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Duration creates a duration field
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

// Error creates an error field
func Error(err error) Field {
	return Field{Key: "error", Value: err.Error()}
}

// Any creates a field with any value
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
}

// StructuredLogger implements the Logger interface with structured logging
type StructuredLogger struct {
	level      LogLevel
	format     string
	output     io.Writer
	fields     map[string]interface{}
	mu         sync.RWMutex
	enableCaller bool
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config LoggingConfig) *StructuredLogger {
	level := parseLogLevel(config.Level)
	
	var output io.Writer = os.Stdout
	if config.Output != "" && config.Output != "stdout" {
		if config.Output == "stderr" {
			output = os.Stderr
		} else {
			// For file output, we would implement file rotation here
			// For now, default to stdout
			output = os.Stdout
		}
	}
	
	return &StructuredLogger{
		level:        level,
		format:       config.Format,
		output:       output,
		fields:       make(map[string]interface{}),
		enableCaller: true,
	}
}

// parseLogLevel converts string log level to LogLevel
func parseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "fatal":
		return FatalLevel
	default:
		return InfoLevel
	}
}

// Debug logs a debug message
func (l *StructuredLogger) Debug(msg string, fields ...Field) {
	if l.level <= DebugLevel {
		l.log(DebugLevel, msg, fields...)
	}
}

// Info logs an info message
func (l *StructuredLogger) Info(msg string, fields ...Field) {
	if l.level <= InfoLevel {
		l.log(InfoLevel, msg, fields...)
	}
}

// Warn logs a warning message
func (l *StructuredLogger) Warn(msg string, fields ...Field) {
	if l.level <= WarnLevel {
		l.log(WarnLevel, msg, fields...)
	}
}

// Error logs an error message
func (l *StructuredLogger) Error(msg string, fields ...Field) {
	if l.level <= ErrorLevel {
		l.log(ErrorLevel, msg, fields...)
	}
}

// Fatal logs a fatal message and exits
func (l *StructuredLogger) Fatal(msg string, fields ...Field) {
	l.log(FatalLevel, msg, fields...)
	os.Exit(1)
}

// With creates a new logger with additional fields
func (l *StructuredLogger) With(fields ...Field) Logger {
	l.mu.RLock()
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	l.mu.RUnlock()
	
	for _, field := range fields {
		newFields[field.Key] = field.Value
	}
	
	return &StructuredLogger{
		level:        l.level,
		format:       l.format,
		output:       l.output,
		fields:       newFields,
		enableCaller: l.enableCaller,
	}
}

// log performs the actual logging
func (l *StructuredLogger) log(level LogLevel, msg string, fields ...Field) {
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level.String(),
		Message:   msg,
		Fields:    make(map[string]interface{}),
	}
	
	// Add persistent fields
	l.mu.RLock()
	for k, v := range l.fields {
		entry.Fields[k] = v
	}
	l.mu.RUnlock()
	
	// Add message fields
	for _, field := range fields {
		entry.Fields[field.Key] = field.Value
	}
	
	// Add caller information if enabled
	if l.enableCaller {
		if pc, file, line, ok := runtime.Caller(2); ok {
			if fn := runtime.FuncForPC(pc); fn != nil {
				entry.Caller = fmt.Sprintf("%s:%d", file, line)
			}
		}
	}
	
	// Format and write the log entry
	var output string
	switch l.format {
	case "json":
		if data, err := json.Marshal(entry); err == nil {
			output = string(data) + "\n"
		} else {
			output = fmt.Sprintf("ERROR: Failed to marshal log entry: %v\n", err)
		}
	case "text", "console":
		output = l.formatText(entry)
	default:
		output = l.formatText(entry)
	}
	
	l.output.Write([]byte(output))
}

// formatText formats log entry as human-readable text
func (l *StructuredLogger) formatText(entry LogEntry) string {
	var builder strings.Builder
	
	// Timestamp and level
	builder.WriteString(entry.Timestamp.Format("2006-01-02 15:04:05.000"))
	builder.WriteString(" [")
	builder.WriteString(entry.Level)
	builder.WriteString("] ")
	
	// Message
	builder.WriteString(entry.Message)
	
	// Fields
	if len(entry.Fields) > 0 {
		builder.WriteString(" {")
		first := true
		for k, v := range entry.Fields {
			if !first {
				builder.WriteString(", ")
			}
			builder.WriteString(k)
			builder.WriteString("=")
			builder.WriteString(fmt.Sprintf("%v", v))
			first = false
		}
		builder.WriteString("}")
	}
	
	// Caller
	if entry.Caller != "" {
		builder.WriteString(" (")
		builder.WriteString(entry.Caller)
		builder.WriteString(")")
	}
	
	builder.WriteString("\n")
	return builder.String()
}

// SetLevel sets the minimum log level
func (l *StructuredLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current log level
func (l *StructuredLogger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// EnableCaller enables or disables caller information in logs
func (l *StructuredLogger) EnableCaller(enable bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enableCaller = enable
}

// DefaultLogger creates a default logger for the pipeline
func DefaultLogger() Logger {
	config := LoggingConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	}
	return NewStructuredLogger(config)
}

// NullLogger creates a logger that discards all output (useful for testing)
func NullLogger() Logger {
	config := LoggingConfig{
		Level:  "fatal", // Only log fatal messages
		Format: "json",
		Output: "stdout",
	}
	logger := NewStructuredLogger(config)
	logger.output = io.Discard
	return logger
}

// LoggerFromStdLog creates a structured logger that wraps the standard log package
type StdLogAdapter struct {
	logger Logger
}

// NewStdLogAdapter creates a new adapter for the standard log package
func NewStdLogAdapter(logger Logger) *StdLogAdapter {
	return &StdLogAdapter{logger: logger}
}

// Write implements io.Writer to capture standard log output
func (a *StdLogAdapter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		a.logger.Info(msg)
	}
	return len(p), nil
}

// SetAsStdLogger sets this adapter as the output for the standard log package
func (a *StdLogAdapter) SetAsStdLogger() {
	log.SetOutput(a)
	log.SetFlags(0) // Remove standard log formatting since we handle it
}