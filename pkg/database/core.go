package database

// DatabaseCore defines the lifecycle contract shared by all drivers.
// Connect establishes the connection (or opens the storage). Healthy reports
// whether the backend is reachable and must be cheap enough for periodic checks.
type DatabaseCore interface {
	Connect() error
	Healthy() bool
	Close()
	Initialize() error
	Drop() error
	ApplyRetentionPolicy(retentionPeriod int64) error
}