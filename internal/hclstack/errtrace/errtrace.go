// Package errtrace provides error wrapping with stack trace capture.
package errtrace

import (
	"fmt"
	"runtime"
	"strings"
)

// WithStackTrace wraps an error with stack trace information.
func WithStackTrace(err error) error {
	if err == nil {
		return nil
	}
	return &traced{err: err, stack: captureStack(2)}
}

// WithStackTraceAndPrefix wraps an error with a formatted message prefix and stack trace.
func WithStackTraceAndPrefix(err error, format string, args ...interface{}) error {
	prefix := fmt.Sprintf(format, args...)
	if err == nil {
		return nil
	}
	return &traced{err: fmt.Errorf("%s: %w", prefix, err), stack: captureStack(2)}
}

type traced struct {
	err   error
	stack string
}

func (t *traced) Error() string {
	return t.err.Error()
}

func (t *traced) Unwrap() error {
	return t.err
}

func captureStack(skip int) string {
	var pcs [32]uintptr
	n := runtime.Callers(skip+1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	var sb strings.Builder
	for {
		frame, more := frames.Next()
		fmt.Fprintf(&sb, "  %s:%d\n", frame.File, frame.Line)
		if !more {
			break
		}
	}
	return sb.String()
}

// Errorf creates a new formatted error with stack trace.
func Errorf(format string, args ...interface{}) error {
	return WithStackTrace(fmt.Errorf(format, args...))
}

// New wraps an existing error with stack trace.
func New(err error) error {
	return WithStackTrace(err)
}

// PrintErrorWithEncoder prints an error using the given encoder.
func PrintErrorWithEncoder(err error, encoder func(v interface{}) error) {
	_ = encoder(err.Error())
}

// IsError checks if two errors are the same type.
func IsError(err error, target error) bool {
	return fmt.Sprintf("%T", err) == fmt.Sprintf("%T", target)
}

// Recover creates an error from a recovered panic value.
func Recover(r interface{}) error {
	if r == nil {
		return nil
	}
	if err, ok := r.(error); ok {
		return WithStackTrace(err)
	}
	return WithStackTrace(fmt.Errorf("%v", r))
}
