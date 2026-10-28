package database

import (
	log "github.com/sirupsen/logrus"
)

// DatabaseType identifies a registered database driver.
type DatabaseType string

const (
	PostgreSQL DatabaseType = "postgresql"
)

// DatabaseConfig carries the connection settings for a database driver.
//
// The Host/Port/User/Password/Name fields are the classic SQL server
// settings and are supported by every driver of that kind. Drivers with
// other addressing schemes (embedded stores, clustered stores, connection
// strings) read them from DSN or Options instead:
//
//   - DSN: a free-form connection string / locator used by the driver when
//     set; drivers that only support DSN-style addressing ignore the
//     Host/Port settings and vice versa.
//   - Options: driver-specific key/value settings (e.g. "timescale" for the
//     PostgreSQL driver, "bucket" or "dataDir" for embedded drivers).
//
// Engine-specific features MUST be expressed as Options, never as shared
// struct fields.
type DatabaseConfig struct {
	Type     DatabaseType
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	Prefix   string

	DSN     string
	Options map[string]string

	DataDir string // Future use
}

// GetOption returns the value for key and whether it was present.
func (config *DatabaseConfig) GetOption(key string) (string, bool) {
	if config.Options == nil {
		return "", false
	}
	value, ok := config.Options[key]
	return value, ok
}

// GetOptionDefault returns the value for key, or defaultValue when absent.
func (config *DatabaseConfig) GetOptionDefault(key string, defaultValue string) string {
	if value, ok := config.GetOption(key); ok {
		return value
	}
	return defaultValue
}

// SetOption sets a driver-specific option, creating the options map on demand.
func (config *DatabaseConfig) SetOption(key string, value string) {
	if config.Options == nil {
		config.Options = make(map[string]string)
	}
	config.Options[key] = value
}

func CreateDatabase(config DatabaseConfig) (Database, error) {
	log.WithFields(log.Fields{
		"DatabaseType": config.Type,
		"Host":         config.Host,
		"Port":         config.Port,
		"Name":         config.Name,
		"Prefix":       config.Prefix,
		"Options":      config.Options,
		"DataDir":      config.DataDir,
	}).Info("Creating database connection")

	db, err := createDatabaseFromRegistry(config)
	if err != nil {
		log.WithField("DatabaseType", config.Type).Error("Unsupported database type requested")
		return nil, err
	}

	return db, nil
}
