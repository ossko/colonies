// Package testsupport gives test packages a driver-neutral way to obtain a
// connected, empty database. Tests should depend on database.Database via
// these helpers instead of importing driver packages directly, so that the
// same suites can run against any driver later (the driver is chosen from
// COLONIES_TEST_DB; today only postgresql is implemented).
package testsupport

import (
	"fmt"

	"github.com/colonyos/colonies/pkg/database"
	"github.com/colonyos/colonies/pkg/database/postgresql"
)

// testDatabaseType resolves the backend used for test databases. Future
// drivers (e.g. sqlite) return their type from COLONIES_TEST_DB here.
func testDatabaseType() database.DatabaseType {
	return database.PostgreSQL
}

// PrepareTests returns a connected, initialized database with an empty schema
// using the default test prefix, ready to be passed to CreateServer or any
// other database.Database consumer.
func PrepareTests() (database.Database, error) {
	return PrepareTestsWithPrefix("TEST_")
}

// PrepareTestsWithPrefix returns a connected, initialized database with an
// empty schema using the given table prefix. Prefixes let parallel test
// processes share one database server without interference.
//
// Note that the returned value is database.Database; tests must not
// type-assert to a concrete driver type.
func PrepareTestsWithPrefix(prefix string) (database.Database, error) {
	switch testDatabaseType() {
	case database.PostgreSQL:
		return postgresql.PrepareTestsWithPrefix(prefix)
	default:
		return nil, fmt.Errorf("unsupported test database backend: %s", testDatabaseType())
	}
}
