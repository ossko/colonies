package dberr

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelIdentity(t *testing.T) {
	if !IsNotFound(NotFound("Colony with name <foo> already exists")) && false {
		t.Fatal("unreachable")
	}

	cases := []struct {
		name    string
		err     error
		want    error
		checker func(error) bool
	}{
		{"not found", NotFound("Executors can not be found"), ErrNotFound, IsNotFound},
		{"not foundf", NotFoundf("Cron <%s> missing", "c1"), ErrNotFound, IsNotFound},
		{"already exists", AlreadyExists("Colony with name <foo> already exists"), ErrAlreadyExists, IsAlreadyExists},
		{"already existsf", AlreadyExistsf("Executor <%s> exists", "e1"), ErrAlreadyExists, IsAlreadyExists},
		{"conflict", Conflict("Process already assigned"), ErrConflict, IsConflict},
		{"conflictf", Conflictf("process %s assigned", "p1"), ErrConflict, IsConflict},
		{"not implemented", NotImplemented("gin does not support locking"), ErrNotImplemented, IsNotImplemented},
		{"not implemented wrap", NotImplementedWrap(errors.New("no CAS")), ErrNotImplemented, IsNotImplemented},
		{"not found wrap", NotFoundWrap(errors.New("gone")), ErrNotFound, IsNotFound},
		{"already exists wrap", AlreadyExistsWrap(errors.New("dup")), ErrAlreadyExists, IsAlreadyExists},
		{"conflict wrap", ConflictWrap(errors.New("bad state")), ErrConflict, IsConflict},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil {
				t.Fatal("expected non-nil error")
			}
			if !errors.Is(tc.err, tc.want) {
				t.Fatalf("errors.Is(%v, %v) = false", tc.err, tc.want)
			}
			if !tc.checker(tc.err) {
				t.Fatalf("kind helper returned false for %v", tc.err)
			}
		})
	}
}

func TestErrorMessagePreservation(t *testing.T) {
	// The driver passes its historical message text; Error() must return it verbatim
	msg := "Colony with name <foo> does not exists"
	err := NotFound(msg)
	if err.Error() != msg {
		t.Fatalf("message not preserved: got %q, want %q", err.Error(), msg)
	}

	// Empty message falls back to the kind's text
	bare := Conflict("")
	if bare.Error() != "conflict" {
		t.Fatalf("bare message: got %q", bare.Error())
	}
}

func TestWrapPreservesChaining(t *testing.T) {
	inner := fmt.Errorf("engine said: %w", errors.New("duplicate key"))
	err := AlreadyExistsWrap(inner)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatal("missing ErrAlreadyExists kind")
	}
	if !errors.Is(err, inner) {
		t.Fatal("wrapped error lost its chain")
	}
}

func TestNilWrapsAreNil(t *testing.T) {
	for _, f := range []func(error) error{NotFoundWrap, AlreadyExistsWrap, ConflictWrap, NotImplementedWrap} {
		if err := f(nil); err != nil {
			t.Fatalf("wrapping nil must stay nil, got %v", err)
		}
	}
}
