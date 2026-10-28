package sqlite

import (
	"path/filepath"
	"strings"

	"github.com/colonyos/colonies/pkg/database"
	_ "modernc.org/sqlite"
)

// Driver registration for the registry-based factory. Binaries embed the
// driver by blank-importing this package:
//
//	_ "github.com/colonyos/colonies/pkg/database/sqlite"
//
// nolint:unparam — the factory never fails for SQLite; open issues surface
// on Connect().
func init() {
	database.RegisterDriver(database.SQLite, newSQLiteDatabaseFromConfig)
}

// newSQLiteDatabaseFromConfig builds an unconnected SQLite database from a
// DatabaseConfig.
//
// The storage file is resolved in this order:
//   - config.DSN (verbatim open path; ":memory:" supported)
//   - Options["path"]
//   - filepath.Join(config.DataDir, config.Name + ".db") when DataDir set
//   - config.Name interpreted as a file path when it looks like one
//   - ":memory:" otherwise (the conformance suite sets an explicit path)
//
// Engine-specific features are Options: e.g. "journal" = WAL (default) |
// "off"; busy timeout via Options["busyTimeoutMs"].
func newSQLiteDatabaseFromConfig(config database.DatabaseConfig) (database.Database, error) {
	path := config.DSN
	if path == "" {
		path, _ = config.GetOption("path")
	}
	if path == "" && config.DataDir != "" {
		path = filepath.Join(config.DataDir, config.Name+".db")
	}
	if path == "" {
		if strings.ContainsRune(config.Name, '/') || strings.HasSuffix(config.Name, ".db") {
			path = config.Name
		}
	}
	if path == "" {
		path = ":memory:"
	}

	db := CreateSQLiteDatabase(path, 0, "", "", "", config.Prefix, false)
	db.path = path

	if journal, ok := config.GetOption("journal"); ok {
		db.journal = journal
	}
	if busyMs, ok := config.GetOption("busyTimeoutMs"); ok {
		db.busyTimeoutMs = busyMs
	}

	return db, nil
}
