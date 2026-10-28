package contract

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAddGetLogsByProcessID(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)

	defer db.Close()

	err = db.AddLog("test_processid", "test_colony", "test_executor_name", time.Now().UTC().UnixNano(), "1")
	assert.Nil(t, err)
	err = db.AddLog("test_processid", "test_colony", "test_executor_name", time.Now().UTC().UnixNano(), "2")
	assert.Nil(t, err)

	logs, err := db.GetLogsByProcessID("test_processid", 100)
	assert.Nil(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, logs[0].ProcessID, "test_processid")
	assert.Equal(t, logs[0].ColonyName, "test_colony")
	assert.Equal(t, logs[0].ExecutorName, "test_executor_name")
}

func TestAddGetLogsByExecutor(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)

	defer db.Close()

	err = db.AddLog("test_processid", "test_colony", "test_executor_name1", time.Now().UTC().UnixNano(), "1")
	assert.Nil(t, err)
	err = db.AddLog("test_processid", "test_colony", "test_executor_name2", time.Now().UTC().UnixNano(), "2")
	assert.Nil(t, err)

	logs, err := db.GetLogsByExecutor("test_executor_name2", 100)
	assert.Nil(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, logs[0].ExecutorName, "test_executor_name2")
}

func TestAddGetLogsByProcessIDSince(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)

	defer db.Close()

	timestamp1 := time.Now().UTC().UnixNano()
	time.Sleep(2 * time.Second)
	timestamp2 := time.Now().UTC().UnixNano()

	err = db.AddLog("test_processid", "test_colony", "test_executor_name", timestamp1, "1")
	assert.Nil(t, err)
	err = db.AddLog("test_processid", "test_colony", "test_executor_name", timestamp2, "2")
	assert.Nil(t, err)

	logs, err := db.GetLogsByProcessIDSince("test_processid", 100, timestamp1)
	assert.Nil(t, err)
	assert.Len(t, logs, 1)
}

func TestAddGetLogsByExecutorSince(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)

	defer db.Close()

	timestamp1 := time.Now().UTC().UnixNano()
	time.Sleep(2 * time.Second)
	timestamp2 := time.Now().UTC().UnixNano()

	err = db.AddLog("test_processid", "test_colony", "test_executor_name", timestamp1, "1")
	assert.Nil(t, err)
	err = db.AddLog("test_processid", "test_colony", "test_executor_name", timestamp2, "2")
	assert.Nil(t, err)

	logs, err := db.GetLogsByExecutorSince("test_executor_name", 100, timestamp1)
	assert.Nil(t, err)
	assert.Len(t, logs, 1)
}

func TestRemoveLogsByColonyName(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)

	defer db.Close()

	err = db.AddLog("test_processid1", "test_colony1", "test_executor_name", time.Now().UTC().UnixNano(), "1")
	assert.Nil(t, err)
	err = db.AddLog("test_processid2", "test_colony2", "test_executor_name", time.Now().UTC().UnixNano(), "2")
	assert.Nil(t, err)

	logs, err := db.GetLogsByProcessID("test_processid1", 100)
	assert.Nil(t, err)
	assert.Len(t, logs, 1)

	err = db.RemoveLogsByColonyName("test_colony1")
	assert.Nil(t, err)

	logs, err = db.GetLogsByProcessID("test_processid1", 100)
	assert.Nil(t, err)
	assert.Len(t, logs, 0)

	logs, err = db.GetLogsByProcessID("test_processid2", 100)
	assert.Nil(t, err)
	assert.Len(t, logs, 1)
}

// TestSearchLogs is intentionally NOT part of the conformance suite: it
// asserts the retention-window semantics of SearchLogs over the ingestion
// time (ADDED). The interface has no way to backdate ingestion (AddLog
// always stamps the current time), so the test lives with the driver, which
// may control ingestion time through driver-internal means: see
// pkg/database/postgresql/driverinternal_test.go (TestSearchLogsOverIngestionTime).
