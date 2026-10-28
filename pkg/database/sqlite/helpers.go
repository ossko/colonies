package sqlite

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// SQLite cannot store Go time.Time values with comparable semantics unless
// the storage format is fixed-width and single-timezone. Every timestamp
// bound into the database goes through sqliteTS: UTC, fixed-width 9-digit
// nanoseconds. With that invariant, TEXT storage compares correctly with
// the same format produced for query parameters (e.g. retention cutoffs and
// SearchLogs windows), which modernc/sqlite then scans back into time.Time
// for DATETIME-declared columns.
const sqliteTimestampFormat = "2006-01-02T15:04:05.000000000Z"

// sqliteTS normalizes a timestamp for storage or parameter binding.
func sqliteTS(t time.Time) string {
	return t.UTC().Format(sqliteTimestampFormat)
}

// stringArray adapts Go []string values to SQLite storage: the PostgreSQL
// driver stores TEXT[] via lib/pq; SQLite stores a JSON array in a TEXT
// column, preserving the null semantics of the interface (nil -> NULL).
// Use stringArray(x) when binding and (*stringArray)(&x) when scanning.
type stringArray []string

func (a stringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return nil, nil // NULL, like an empty TEXT[] in PostgreSQL
	}
	data, err := json.Marshal([]string(a))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal string array: %w", err)
	}
	return string(data), nil
}

func (a *stringArray) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}
	var data string
	switch v := src.(type) {
	case string:
		data = v
	case []byte:
		data = string(v)
	default:
		return fmt.Errorf("unsupported type for string array scan: %T", src)
	}

	var values []string
	if err := json.Unmarshal([]byte(data), &values); err != nil {
		return fmt.Errorf("failed to unmarshal string array %q: %w", data, err)
	}
	*a = values
	return nil
}

// generalPoolsJSON is the JSON representation of the general-pool marker
// (TARGET_EXECUTOR_NAMES = ['*'] in PostgreSQL).
const generalPoolsJSON = `["*"]`

// sortStrings is a small helper shared by parse helpers
func sortStrings(values []string) {
	sort.Strings(values)
}

var _ = strings.TrimSpace // reserved for future parse helpers
