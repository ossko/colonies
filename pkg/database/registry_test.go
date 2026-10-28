package database

import (
	"strings"
	"testing"
)

// driverStub satisfies the fat Database interface by embedding it; calls
// would panic on the nil embedded interface, which is fine since the registry
// tests never use the returned database beyond type-asserting it.
type driverStub struct {
	Database
}

func (d *driverStub) Connect() error                                   { return nil }
func (d *driverStub) Healthy() bool                                    { return true }
func (d *driverStub) Close()                                           {}
func (d *driverStub) Initialize() error                                { return nil }
func (d *driverStub) Drop() error                                      { return nil }
func (d *driverStub) ApplyRetentionPolicy(retentionPeriod int64) error { return nil }

func TestRegisterAndCreate(t *testing.T) {
	// Register a driver under a private type name so the global registry is
	// only touched at a key this test owns
	registryMu.Lock()
	_, existed := registeredDrivers["stub-test"]
	registryMu.Unlock()
	if existed {
		t.Fatal("stub-test must not be pre-registered")
	}

	RegisterDriver("stub-test", func(config DatabaseConfig) (Database, error) {
		if config.Type != "stub-test" {
			t.Fatalf("config.Type = %q, want stub-test", config.Type)
		}
		return &driverStub{}, nil
	})

	db, err := CreateDatabase(DatabaseConfig{Type: "stub-test"})
	if err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	if _, ok := db.(*driverStub); !ok {
		t.Fatalf("factory returned %T", db)
	}
}

func TestRegisterTwicePanics(t *testing.T) {
	RegisterDriver("stub-double", func(config DatabaseConfig) (Database, error) {
		return &driverStub{}, nil
	})

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on double registration")
		}
	}()
	RegisterDriver("stub-double", func(config DatabaseConfig) (Database, error) {
		return &driverStub{}, nil
	})
}

func TestUnknownTypeError(t *testing.T) {
	_, err := CreateDatabase(DatabaseConfig{Type: DatabaseType("no-such-driver")})
	if err == nil {
		t.Fatal("expected error for unregistered type")
	}
	// Registration is link-time: which drivers the test binary has depends on
	// its imports, so only the prefix is asserted
	if !strings.HasPrefix(err.Error(), "unsupported database type: no-such-driver") {
		t.Fatalf("unexpected error text: %q", err.Error())
	}
}

func TestOptionsHelpers(t *testing.T) {
	config := DatabaseConfig{Type: PostgreSQL}

	if _, ok := config.GetOption("timescale"); ok {
		t.Fatal("nil options must yield false")
	}
	if got := config.GetOptionDefault("timescale", "fallback"); got != "fallback" {
		t.Fatalf("default option = %q", got)
	}

	config.SetOption("timescale", "true")
	value, ok := config.GetOption("timescale")
	if !ok || value != "true" {
		t.Fatalf("option read-back: %q %v", value, ok)
	}
	if config.GetOptionDefault("timescale", "fallback") != "true" {
		t.Fatal("present option must win over default")
	}
}
