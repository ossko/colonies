// Package dberr defines the semantic error contract for database drivers.
//
// Every database driver implementation must classify its native errors into
// the kinds defined here and wrap them so callers can test identity with
// errors.Is. The human-readable message text of database errors is NOT a
// stable interface: consumers must use errors.Is or the Is* helpers, never
// string matching on error messages.
package dberr

import (
	"errors"
	"fmt"
)

// Semantic error kinds that every driver must honor.
var (
	// ErrNotFound means a requested entity does not exist. Note that Get*
	// methods conventionally return nil results (not this error) when no
	// row was found; mutating methods return ErrNotFound when they act on
	// a missing entity. See CONTRACT.md.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists means an entity with the same unique key already
	// exists (e.g. colony name, executor name within a colony).
	ErrAlreadyExists = errors.New("already exists")

	// ErrConflict means the operation is illegal in the current state of
	// the targeted entity, e.g. assigning an already assigned process or
	// marking a successful process as failed.
	ErrConflict = errors.New("conflict")

	// ErrNotImplemented means the driver does not implement the method,
	// typically because the engine cannot support the required semantics
	// or the feature is engine-specific (e.g. TimescaleDB acceleration).
	ErrNotImplemented = errors.New("not implemented")
)

// DBError carries a semantic kind (one of the sentinel errors above) and a
// human-readable message. Message text is driver-defined and must not be
// interpreted by callers; use errors.Is / the Is* helpers instead.
//
// DBError unwraps to both its kind (e.g. ErrNotFound) and, when constructed
// with one of the *Wrap helpers, to the original engine error, so that
// errors.Is works against either.
type DBError struct {
	kind  error
	msg   string
	cause error
}

func (e *DBError) Error() string {
	if e.msg == "" {
		return e.kind.Error()
	}
	return e.msg
}

// Unwrap returns multiple targets when a cause is present; errors.Is and
// errors.As examine every element (Go 1.20+ multi-unwrap semantics).
func (e *DBError) Unwrap() []error {
	if e.cause == nil {
		return []error{e.kind}
	}
	return []error{e.kind, e.cause}
}

func newDBError(kind error, msg string) error {
	return &DBError{kind: kind, msg: msg}
}

// NotFound returns an error matching errors.Is(err, ErrNotFound).
// An empty msg produces the bare message "not found".
func NotFound(msg string) error {
	return newDBError(ErrNotFound, msg)
}

// NotFoundf returns ErrNotFound with a formatted message.
func NotFoundf(format string, args ...interface{}) error {
	return newDBError(ErrNotFound, fmt.Sprintf(format, args...))
}

// AlreadyExists returns an error matching errors.Is(err, ErrAlreadyExists).
func AlreadyExists(msg string) error {
	return newDBError(ErrAlreadyExists, msg)
}

// AlreadyExistsf returns ErrAlreadyExists with a formatted message.
func AlreadyExistsf(format string, args ...interface{}) error {
	return newDBError(ErrAlreadyExists, fmt.Sprintf(format, args...))
}

// Conflict returns an error matching errors.Is(err, ErrConflict).
func Conflict(msg string) error {
	return newDBError(ErrConflict, msg)
}

// Conflictf returns ErrConflict with a formatted message.
func Conflictf(format string, args ...interface{}) error {
	return newDBError(ErrConflict, fmt.Sprintf(format, args...))
}

// NotImplemented returns an error matching errors.Is(err, ErrNotImplemented).
func NotImplemented(msg string) error {
	return newDBError(ErrNotImplemented, msg)
}

// NotFoundWrap wraps err with kind ErrNotFound while preserving the wrapped error chain.
func NotFoundWrap(err error) error {
	if err == nil {
		return nil
	}
	return &DBError{kind: ErrNotFound, msg: err.Error(), cause: err}
}

// AlreadyExistsWrap wraps err with kind ErrAlreadyExists while preserving the wrapped error chain.
func AlreadyExistsWrap(err error) error {
	if err == nil {
		return nil
	}
	return &DBError{kind: ErrAlreadyExists, msg: err.Error(), cause: err}
}

// ConflictWrap wraps err with kind ErrConflict while preserving the wrapped error chain.
func ConflictWrap(err error) error {
	if err == nil {
		return nil
	}
	return &DBError{kind: ErrConflict, msg: err.Error(), cause: err}
}

// NotImplementedWrap wraps err with kind ErrNotImplemented while preserving the wrapped error chain.
func NotImplementedWrap(err error) error {
	if err == nil {
		return nil
	}
	return &DBError{kind: ErrNotImplemented, msg: err.Error(), cause: err}
}

// IsNotFound reports whether err (or anything it wraps) has kind ErrNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsAlreadyExists reports whether err (or anything it wraps) has kind ErrAlreadyExists.
func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}

// IsConflict reports whether err (or anything it wraps) has kind ErrConflict.
func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

// IsNotImplemented reports whether err (or anything it wraps) has kind ErrNotImplemented.
func IsNotImplemented(err error) bool {
	return errors.Is(err, ErrNotImplemented)
}
