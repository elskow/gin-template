package errors

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// AppError represents an application-level error with additional context
type AppError struct {
	Code       string
	Message    string
	StatusCode int
	Err        error
	cachedMsg  string // cached error message for zero-allocation Error() calls
}

func (e *AppError) Error() string {
	if e.cachedMsg != "" {
		return e.cachedMsg
	}
	if e.Err != nil {
		e.cachedMsg = e.Message + ": " + e.Err.Error()
	} else {
		e.cachedMsg = e.Message
	}
	return e.cachedMsg
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new application error with status code
func NewAppError(code, message string, statusCode int, err error) *AppError {
	ae := &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
	}
	// Pre-compute the error message for zero-allocation Error() calls
	if err != nil {
		ae.cachedMsg = message + ": " + err.Error()
	} else {
		ae.cachedMsg = message
	}
	return ae
}

// wrapError is an allocation-efficient error wrapper
type wrapError struct {
	msg string
	err error
}

func (e *wrapError) Error() string {
	return e.msg
}

func (e *wrapError) Unwrap() error {
	return e.err
}

// RecordError records an error in an OpenTelemetry span
func RecordError(span trace.Span, err error) {
	if err == nil || span == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetSpanError sets the error status on an OpenTelemetry span
func SetSpanError(span trace.Span, err error) {
	if err == nil || span == nil {
		return
	}
	span.SetStatus(codes.Error, err.Error())
}

// Re-export standard library functions for convenience
var (
	New    = errors.New
	Is     = errors.Is
	As     = errors.As
	Unwrap = errors.Unwrap
	Join   = errors.Join
)

// Wrap wraps an error with additional context
// This is allocation-efficient: it pre-computes the error message at wrap time,
// so Error() method calls are zero-allocation (just returns the cached string)
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return &wrapError{
		msg: message + ": " + err.Error(),
		err: err,
	}
}

// Wrapf wraps an error with a formatted message
// Uses fmt.Sprintf for formatting, then stores the result for zero-allocation Error() calls
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	msg := fmt.Sprintf(format, args...)
	return &wrapError{
		msg: msg + ": " + err.Error(),
		err: err,
	}
}
