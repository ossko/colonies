package postgresql

import (
	"github.com/colonyos/colonies/pkg/database"
)

// Driver registration for the registry-based factory. Binaries embed the
// driver by blank-importing this package:
//
//	_ "github.com/colonyos/colonies/pkg/database/postgresql"
func init() {
	database.RegisterDriver(database.PostgreSQL, newPQDatabaseFromConfig)
}

// newPQDatabaseFromConfig builds an unconnected PostgreSQL database from a
// DatabaseConfig. TimescaleDB acceleration is enabled via the "timescale"
// option (e.g. COLONIES_DB_TIMESCALEDB=true; see internal/cli/db.go).
//
// nolint:unparam — the factory never fails for PostgreSQL; connection
// failures surface on Connect().
func newPQDatabaseFromConfig(config database.DatabaseConfig) (database.Database, error) {
	return CreatePQDatabase(
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Name,
		config.Prefix,
		config.GetOptionDefault("timescale", "") == "true",
	), nil
}
