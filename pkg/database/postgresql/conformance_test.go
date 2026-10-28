package postgresql

import (
	"testing"

	"github.com/colonyos/colonies/pkg/database"
	"github.com/colonyos/colonies/pkg/database/contract"
)

// Activates the driver-agnostic conformance suite (pkg/database/contract)
// for this driver: TestConformance runs all 200+ behavioral tests through
// the factory below, i.e. against this driver's real storage backend.
// Individual tests can be run with:
//
//	go test ./pkg/database/postgresql/ -run 'TestConformance/<TestName>'
//
// Driver-specific tests stay next to this file; behaviors that depend on
// internals the interface deliberately does not expose are in
// driverinternal_test.go.
func init() {
	contract.RegisterConformanceFactory(func(prefix string) (database.Database, error) {
		return PrepareTestsWithPrefix(prefix)
	})
}

// TestConformance runs the full behavioral suite for this driver.
func TestConformance(t *testing.T) {
	contract.RunSuite(t)
}
