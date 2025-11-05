package errors

import (
	"errors"
	"fmt"
	"testing"
)

var (
	baseErr = errors.New("base error")
	result  error
)

// Benchmark our optimized Wrap
func BenchmarkWrap(b *testing.B) {
	b.ReportAllocs()
	var err error
	for i := 0; i < b.N; i++ {
		err = Wrap(baseErr, "operation failed")
	}
	result = err
}

// Benchmark stdlib fmt.Errorf for comparison
func BenchmarkStdlibFmtErrorf(b *testing.B) {
	b.ReportAllocs()
	var err error
	for i := 0; i < b.N; i++ {
		err = fmt.Errorf("operation failed: %w", baseErr)
	}
	result = err
}

// Benchmark our optimized Wrapf
func BenchmarkWrapf(b *testing.B) {
	b.ReportAllocs()
	var err error
	for i := 0; i < b.N; i++ {
		err = Wrapf(baseErr, "operation failed with code %d", 500)
	}
	result = err
}

// Benchmark stdlib fmt.Errorf with formatting
func BenchmarkStdlibFmtErrorfWithFormat(b *testing.B) {
	b.ReportAllocs()
	var err error
	for i := 0; i < b.N; i++ {
		err = fmt.Errorf("operation failed with code %d: %w", 500, baseErr)
	}
	result = err
}

// Benchmark Error() method calls - our optimized version
func BenchmarkWrapError(b *testing.B) {
	err := Wrap(baseErr, "operation failed")
	b.ReportAllocs()
	b.ResetTimer()
	var s string
	for i := 0; i < b.N; i++ {
		s = err.Error()
	}
	_ = s
}

// Benchmark Error() method calls - stdlib version
func BenchmarkStdlibError(b *testing.B) {
	err := fmt.Errorf("operation failed: %w", baseErr)
	b.ReportAllocs()
	b.ResetTimer()
	var s string
	for i := 0; i < b.N; i++ {
		s = err.Error()
	}
	_ = s
}

// Benchmark AppError creation
func BenchmarkNewAppError(b *testing.B) {
	b.ReportAllocs()
	var err error
	for i := 0; i < b.N; i++ {
		err = NewAppError("INTERNAL_ERROR", "operation failed", 500, baseErr)
	}
	result = err
}

// Benchmark AppError.Error() calls
func BenchmarkAppErrorError(b *testing.B) {
	err := NewAppError("INTERNAL_ERROR", "operation failed", 500, baseErr)
	b.ReportAllocs()
	b.ResetTimer()
	var s string
	for i := 0; i < b.N; i++ {
		s = err.Error()
	}
	_ = s
}

// Benchmark chained wrapping (realistic scenario)
func BenchmarkChainedWrap(b *testing.B) {
	b.ReportAllocs()
	var err error
	for i := 0; i < b.N; i++ {
		err = baseErr
		err = Wrap(err, "repository error")
		err = Wrap(err, "service error")
		err = Wrap(err, "controller error")
	}
	result = err
}

// Benchmark chained wrapping with stdlib
func BenchmarkChainedStdlib(b *testing.B) {
	b.ReportAllocs()
	var err error
	for i := 0; i < b.N; i++ {
		err = baseErr
		err = fmt.Errorf("repository error: %w", err)
		err = fmt.Errorf("service error: %w", err)
		err = fmt.Errorf("controller error: %w", err)
	}
	result = err
}

// Benchmark Unwrap
func BenchmarkUnwrap(b *testing.B) {
	wrapped := Wrap(baseErr, "operation failed")
	b.ReportAllocs()
	b.ResetTimer()
	var err error
	for i := 0; i < b.N; i++ {
		err = Unwrap(wrapped)
	}
	result = err
}

// Benchmark Is check
func BenchmarkIs(b *testing.B) {
	wrapped := Wrap(baseErr, "operation failed")
	b.ReportAllocs()
	b.ResetTimer()
	var ok bool
	for i := 0; i < b.N; i++ {
		ok = Is(wrapped, baseErr)
	}
	_ = ok
}
