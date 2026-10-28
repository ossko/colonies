package sqlite

import (
	"testing"

	"github.com/colonyos/colonies/pkg/database"
	"github.com/colonyos/colonies/pkg/database/contract"
)

// Activates the driver-agnostic conformance suite (pkg/database/contract)
// for this driver. TestConformance runs all behavioral tests against the
// SQLite storage backend; per-test selection:
//
//	go test ./pkg/database/sqlite/ -run 'TestConformance/<TestName>'
//
// Behaviors that depend on internals the interface deliberately does not
// expose are covered in driverinternal_test.go.
func init() {
	contract.RegisterConformanceFactory(func(prefix string) (database.Database, error) {
		return PrepareTestsWithPrefix(prefix)
	})
}

// TestConformance runs the full behavioral suite for this driver.
func TestConformance(t *testing.T) {
	contract.RunSuite(t)
}
