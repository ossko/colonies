package sqlite

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Test database lifecycle: one SQLite file per test process (in a temp
// directory), shared across prefixes, mirroring the per-process database
// model of the PostgreSQL driver. Every PrepareTestsWithPrefix call wipes the
// tables of its prefix so each test hands off an empty schema.
var (
	testPathOnce sync.Once
	testPath     string
	testPathErr  error
)

func prepareTestPath() (string, error) {
	testPathOnce.Do(func() {
		dir, err := os.MkdirTemp("", "colonies_sqlite_test_")
		if err != nil {
			testPathErr = err
			return
		}
		testPath = filepath.Join(dir, "colonies.db")
	})
	return testPath, testPathErr
}

// PrepareTests returns a connected, empty test database with the default
// prefix.
func PrepareTests() (*SQLiteDatabase, error) {
	return PrepareTestsWithPrefix("TEST_")
}

// PrepareTestsWithPrefix returns a connected database whose schema (tables
// for the given prefix) is empty.
func PrepareTestsWithPrefix(prefix string) (*SQLiteDatabase, error) {
	log.SetOutput(log.StandardLogger().Out) // keep the default logger behavior

	path, err := prepareTestPath()
	if err != nil {
		return nil, err
	}

	db := CreateSQLiteDatabase("", 0, "", "", "", prefix, false)
	db.path = path

	if err := db.Connect(); err != nil {
		return nil, err
	}

	if err := db.clearTestData(prefix); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// clearTestData drops and recreates every table of the prefix, guaranteeing
// an empty schema without paying for a new file.
func (db *SQLiteDatabase) clearTestData(prefix string) error {
	rows, err := db.postgresql.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name LIKE ?`, strings.ToLower(prefix)+"%")
	if err == nil {
		var dropped []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				dropped = append(dropped, name)
			}
		}
		rows.Close()

		for _, name := range dropped {
			if _, err := db.postgresql.Exec(`DROP TABLE IF EXISTS ` + name); err != nil {
				return err
			}
		}
	} else {
		rows.Close()
	}

	// Fall through: sqlite_master stores table names with the case they were
	// created in; the driver creates UPPER-case prefixed tables
	return db.Initialize()
}
