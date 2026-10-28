package postgresql

import (
	"testing"

	"github.com/colonyos/colonies/pkg/database"
	"github.com/stretchr/testify/assert"
)

func TestNewPQDatabaseFromConfig(t *testing.T) {
	// Without the option, TimescaleDB acceleration must stay off
	db, err := newPQDatabaseFromConfig(database.DatabaseConfig{
		Type:   database.PostgreSQL,
		Host:   "nohost",
		Port:   1,
		Prefix: "SOMEPREFIX_",
	})
	assert.Nil(t, err)
	pq, ok := db.(*PQDatabase)
	assert.True(t, ok)
	assert.False(t, pq.timescaleDB)
	assert.Equal(t, "SOMEPREFIX_", pq.dbPrefix)

	// The "timescale" driver option opts in to hypertables
	db, err = newPQDatabaseFromConfig(database.DatabaseConfig{
		Type:    database.PostgreSQL,
		Options: map[string]string{"timescale": "true"},
	})
	assert.Nil(t, err)
	pq, ok = db.(*PQDatabase)
	assert.True(t, ok)
	assert.True(t, pq.timescaleDB)

	// Any other value must not enable it
	db, err = newPQDatabaseFromConfig(database.DatabaseConfig{
		Type:    database.PostgreSQL,
		Options: map[string]string{"timescale": "false"},
	})
	assert.Nil(t, err)
	pq, ok = db.(*PQDatabase)
	assert.True(t, ok)
	assert.False(t, pq.timescaleDB)
}
