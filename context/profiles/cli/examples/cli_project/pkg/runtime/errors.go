package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ExitCode is a typed process exit code.
type ExitCode int

// Exit codes follow sysexits(3) where one fits.
const (
	ExitOK          ExitCode = 0
	ExitNotFound    ExitCode = 1  // the requested thing does not exist
	ExitUsage       ExitCode = 2  // command-line usage error
	ExitUnavailable ExitCode = 69 // service unavailable, or interrupted
	ExitSoftware    ExitCode = 70 // internal software error
	ExitPartial     ExitCode = 71 // mixed results: some operations failed, not all
	ExitTempFail    ExitCode = 75 // transient upstream failure: a rerun may succeed
	ExitConfig      ExitCode = 78 // configuration error
)

// ExitError carries a semantic exit code alongside the underlying error.
type ExitError struct {
	Code ExitCode
	Err  error
	// Silent marks an outcome the command already reported through its own
	// output (denial, "no results"): handlers skip error rendering and let
	// the exit code alone carry the outcome.
	Silent  bool
	Message string
}

func (e *ExitError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err == nil {
		return fmt.Sprintf("exit %d", e.Code)
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }

// Exit wraps err with a semantic exit code.
func Exit(code ExitCode, err error) *ExitError { return &ExitError{Code: code, Err: err} }

// Exitf formats a message and keeps the last error argument as the cause,
// so errors.Is and errors.As still traverse the chain.
func Exitf(code ExitCode, format string, args ...any) *ExitError {
	var cause error
	for _, a := range args {
		if e, ok := a.(error); ok && e != nil {
			cause = e
		}
	}
	return &ExitError{Code: code, Err: cause, Message: fmt.Sprintf(format, args...)}
}

// ExitSilent sets the exit code without rendering an error.
func ExitSilent(code ExitCode, message string) *ExitError {
	return &ExitError{Code: code, Message: message, Silent: true}
}

// IsSilentExit reports whether err is an ExitError marked Silent.
func IsSilentExit(err error) bool {
	var ee *ExitError
	return errors.As(err, &ee) && ee.Silent
}

// MultiError aggregates the errors of a batch operation.
type MultiError struct {
	Errors []error
}

// NewMultiError collects the non-nil errs.
func NewMultiError(errs ...error) *MultiError {
	m := &MultiError{}
	for _, e := range errs {
		if e != nil {
			m.Errors = append(m.Errors, e)
		}
	}
	return m
}

func (m *MultiError) Error() string {
	msgs := make([]string, len(m.Errors))
	for i, e := range m.Errors {
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "\n")
}

func (m *MultiError) Unwrap() []error { return m.Errors }

// Append adds err and returns the receiver for chaining.
func (m *MultiError) Append(err error) *MultiError {
	m.Errors = append(m.Errors, err)
	return m
}

// ErrorOrNil returns nil when nothing was collected.
func (m *MultiError) ErrorOrNil() error {
	if len(m.Errors) == 0 {
		return nil
	}
	return m
}

// ExitCode returns the highest code among the collected errors; an error
// without a code counts as ExitSoftware.
func (m *MultiError) ExitCode() ExitCode {
	var maxCode ExitCode
	for _, e := range m.Errors {
		code := ExitSoftware
		if ee, ok := errors.AsType[*ExitError](e); ok {
			code = ee.Code
		}
		maxCode = max(maxCode, code)
	}
	return maxCode
}

// ExitCodeFromError maps an error to a process exit code. An error that
// carries a code always reports it; only a bare cancellation of ctx (the
// user pressed Ctrl-C and the command stopped cleanly) maps to ExitOK.
// Unwrapping first matters: a coded failure whose chain happens to include
// the cancellation must not exit 0.
func ExitCodeFromError(ctx context.Context, err error) ExitCode {
	if err == nil {
		return ExitOK
	}
	if me, ok := errors.AsType[*MultiError](err); ok {
		return me.ExitCode()
	}
	if ee, ok := errors.AsType[*ExitError](err); ok {
		return ee.Code
	}
	if ctx.Err() != nil {
		if errors.Is(err, ctx.Err()) {
			return ExitOK
		}
		return ExitUnavailable
	}
	return ExitSoftware
}
