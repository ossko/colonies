package sqlite

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Driver-internal tests: these exercise behavior that depends on internals
// the database.Database interface deliberately does not expose (e.g. the
// ingestion time of a log row via the driver's addHistoricalLog helper,
// which controls the ADDED column; AddLog always stamps the current time).
// The interface-level counterpart of these behaviors is covered by the
// conformance suite (pkg/database/contract).

// TestSearchLogsOverIngestionTime verifies that SearchLogs filters by the
// ingestion time (ADDED), not by the log's own timestamp. This is the
// driver-side companion of the contract suite's SearchLogs coverage
// (CONTRACT.md, LogDatabase).
func TestSearchLogsOverIngestionTime(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)

	defer db.Close()

	timestamp := time.Now().UTC()
	err = db.addHistoricalLog("test_processid1", "test_colony", "test_executor_name1", timestamp.UnixNano(), "1", timestamp)
	assert.Nil(t, err)
	timestamp = time.Now().UTC()
	err = db.addHistoricalLog("test_processid2", "test_colony", "test_executor_name1", timestamp.UnixNano(), "test", timestamp)
	assert.Nil(t, err)
	timestamp = time.Now().UTC()
	err = db.addHistoricalLog("test_processid2", "test_colony", "test_executor_name1", timestamp.UnixNano(), "test", timestamp)
	assert.Nil(t, err)
	timestamp = time.Now().Add(-24*time.Hour*3 - time.Hour).UTC()
	err = db.addHistoricalLog("test_processid3", "test_colony", "test_executor_name2", timestamp.UnixNano(), "error", timestamp)
	assert.Nil(t, err)

	logs, err := db.SearchLogs("test_colony", "", -1, 10)
	assert.Nil(t, err)
	assert.Len(t, logs, 0)

	logs, err = db.SearchLogs("test_colony", "error", -1, 10)
	assert.Nil(t, err)
	assert.Len(t, logs, 0)

	logs, err = db.SearchLogs("test_colony", "error", 0, 10)
	assert.Nil(t, err)
	assert.Len(t, logs, 0)

	logs, err = db.SearchLogs("test_colony", "error", 1, 10)
	assert.Nil(t, err)
	assert.Len(t, logs, 0)

	logs, err = db.SearchLogs("test_colony", "error", 2, 10)
	assert.Nil(t, err)
	assert.Len(t, logs, 0)

	logs, err = db.SearchLogs("test_colony", "error", 3, 10)
	assert.Nil(t, err)
	assert.Len(t, logs, 0)

	logs, err = db.SearchLogs("test_colony", "error", 4, 10)
	assert.Nil(t, err)
	assert.Len(t, logs, 1)
	for _, log := range logs {
		assert.Equal(t, log.ProcessID, "test_processid3")
		assert.Equal(t, log.Message, "error")
	}

	logs, err = db.SearchLogs("test_colony", "test", 3, 10)
	assert.Nil(t, err)
	assert.Len(t, logs, 2)
	for _, log := range logs {
		assert.Equal(t, log.Message, "test")
		assert.Equal(t, log.ExecutorName, "test_executor_name1")
	}
}
