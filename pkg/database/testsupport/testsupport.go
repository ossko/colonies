// Package testsupport gives test packages a driver-neutral way to obtain a
// connected, empty database. Tests should depend on database.Database via
// these helpers instead of importing driver packages directly, so that the
// same suites can run against any driver.
//
// The backend is selected with COLONIES_TEST_DB (default "postgresql";
// "sqlite" is supported). Both drivers are always linked in.
package testsupport

import (
	"fmt"
	"os"

	"github.com/colonyos/colonies/pkg/database"
	"github.com/colonyos/colonies/pkg/database/postgresql"
	"github.com/colonyos/colonies/pkg/database/sqlite"
)

// backend prepares a connected, empty database for the given prefix.
type backend func(prefix string) (database.Database, error)

var testBackends = map[database.DatabaseType]backend{
	database.PostgreSQL: func(prefix string) (database.Database, error) {
		return postgresql.PrepareTestsWithPrefix(prefix)
	},
	database.SQLite: func(prefix string) (database.Database, error) {
		return sqlite.PrepareTestsWithPrefix(prefix)
	},
}

// testDatabaseType resolves the backend used for test databases from
// COLONIES_TEST_DB (default "postgresql").
func testDatabaseType() (database.DatabaseType, error) {
	selected := os.Getenv("COLONIES_TEST_DB")
	if selected == "" {
		selected = string(database.PostgreSQL)
	}
	if _, ok := testBackends[database.DatabaseType(selected)]; !ok {
		return "", fmt.Errorf("unknown COLONIES_TEST_DB backend %q (supported: %v)", selected, supportedBackends())
	}
	return database.DatabaseType(selected), nil
}

func supportedBackends() []string {
	names := make([]string, 0, len(testBackends))
	for name := range testBackends {
		names = append(names, string(name))
	}
	return names
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
	backendType, err := testDatabaseType()
	if err != nil {
		return nil, err
	}
	return testBackends[backendType](prefix)
}
