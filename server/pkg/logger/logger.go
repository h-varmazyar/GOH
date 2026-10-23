package logger

import (
	"io"
	stdlog "log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/sirupsen/logrus"
)

var Logger = logrus.StandardLogger()

// Configure writes all logrus and standard-library logs as JSON lines to both
// stdout and path. The returned closer must remain open for the application lifetime.
func Configure(path string, level logrus.Level) (io.Closer, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, err
	}

	standard := logrus.StandardLogger()
	standard.SetFormatter(&logrus.JSONFormatter{TimestampFormat: "2006-01-02T15:04:05.000Z07:00"})
	standard.SetOutput(io.MultiWriter(os.Stdout, file))
	standard.SetLevel(level)
	Logger = standard

	stdlog.SetFlags(0)
	stdlog.SetOutput(standardLogWriter{})
	return file, nil
}

type standardLogWriter struct{}

func (standardLogWriter) Write(data []byte) (int, error) {
	message := strings.TrimSpace(string(data))
	if message != "" {
		component := "stdlib"
		if strings.HasPrefix(message, "[") {
			if end := strings.IndexByte(message, ']'); end > 1 {
				component = kebabCase(message[1:end])
			}
		}
		logrus.WithField("component", component).Info(message)
	}
	return len(data), nil
}

func kebabCase(value string) string {
	var result strings.Builder
	for i, r := range value {
		if unicode.IsUpper(r) && i > 0 {
			result.WriteByte('-')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

// GetLogger returns the global logger instance
func GetLogger() *logrus.Logger {
	return Logger
}

// WithError returns a logger entry with error field
func WithError(err error) *logrus.Entry {
	return Logger.WithError(err)
}

// WithField returns a logger entry with a single field
func WithField(key string, value interface{}) *logrus.Entry {
	return Logger.WithField(key, value)
}

// WithFields returns a logger entry with multiple fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Logger.WithFields(fields)
}

// InfoMessage logs an info level message
func Info(msg string) {
	Logger.Info(msg)
}

// ErrorMessage logs an error level message
func Error(msg string) {
	Logger.Error(msg)
}

// DebugMessage logs a debug level message
func Debug(msg string) {
	Logger.Debug(msg)
}

// WarnMessage logs a warning level message
func Warn(msg string) {
	Logger.Warn(msg)
}
