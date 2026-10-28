package database

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// DriverFactory instantiates a Database from a configuration. The returned
// database is not yet connected; callers must call Connect() (see
// ConnectWithRetry) before use.
type DriverFactory func(config DatabaseConfig) (Database, error)

var (
	registryMu        sync.RWMutex
	registeredDrivers = make(map[DatabaseType]DriverFactory)
)

// RegisterDriver registers a database driver under a type name. Driver
// packages call it from init() and production binaries blank-import driver
// packages to trigger registration. Registering the same type twice is a
// programming error and panics.
func RegisterDriver(databaseType DatabaseType, factory DriverFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if factory == nil {
		panic("database: RegisterDriver called with nil factory")
	}
	if _, exists := registeredDrivers[databaseType]; exists {
		panic("database: driver already registered: " + string(databaseType))
	}
	registeredDrivers[databaseType] = factory
}

// RegisteredDrivers returns the sorted names of all registered drivers,
// mainly for diagnostics and error messages.
func RegisteredDrivers() []DatabaseType {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]DatabaseType, 0, len(registeredDrivers))
	for name := range registeredDrivers {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}

// createDatabaseFromRegistry instantiates the driver registered for
// config.Type. It expects the caller to have checked the map; an unknown type
// yields a descriptive error.
func createDatabaseFromRegistry(config DatabaseConfig) (Database, error) {
	registryMu.RLock()
	factory, exists := registeredDrivers[config.Type]
	registryMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported database type: %s (registered drivers: %s)",
			config.Type, strings.Join(registerDriverNames(), ", "))
	}

	return factory(config)
}

// registerDriverNames snapshots the registered driver names.
func registerDriverNames() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(registeredDrivers))
	for name := range registeredDrivers {
		names = append(names, string(name))
	}
	sort.Strings(names)
	return names
}
