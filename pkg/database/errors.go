package database

import "github.com/colonyos/colonies/pkg/database/dberr"

// Semantic error kinds returned by all database drivers. These re-export the
// sentinels defined in pkg/database/dberr (imported by drivers to avoid an
// import cycle with the factory in this package).
//
// Callers must compare using errors.Is, never by matching message text.
var (
	ErrNotFound       = dberr.ErrNotFound
	ErrAlreadyExists  = dberr.ErrAlreadyExists
	ErrConflict       = dberr.ErrConflict
	ErrNotImplemented = dberr.ErrNotImplemented
)

// NotFound creates a database error with kind ErrNotFound.
func NotFound(msg string) error { return dberr.NotFound(msg) }

// AlreadyExists creates a database error with kind ErrAlreadyExists.
func AlreadyExists(msg string) error { return dberr.AlreadyExists(msg) }

// Conflict creates a database error with kind ErrConflict.
func Conflict(msg string) error { return dberr.Conflict(msg) }

// NotImplemented creates a database error with kind ErrNotImplemented.
func NotImplemented(msg string) error { return dberr.NotImplemented(msg) }

// IsNotFound reports whether err has kind ErrNotFound.
func IsNotFound(err error) bool { return dberr.IsNotFound(err) }

// IsAlreadyExists reports whether err has kind ErrAlreadyExists.
func IsAlreadyExists(err error) bool { return dberr.IsAlreadyExists(err) }

// IsConflict reports whether err has kind ErrConflict.
func IsConflict(err error) bool { return dberr.IsConflict(err) }

// IsNotImplemented reports whether err has kind ErrNotImplemented.
func IsNotImplemented(err error) bool { return dberr.IsNotImplemented(err) }
