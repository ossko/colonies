package postgresql

import (
	"testing"
	"time"

	"github.com/colonyos/colonies/pkg/core"
	"github.com/colonyos/colonies/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// Driver-internal tests: these exercise behavior that depends on internals
// the database.Database interface deliberately does not expose (e.g. the
// ingestion time of a log row via the driver's addHistoricalLog helper, or
// internal selection helpers). The interface-level counterpart of these
// behaviors is covered by the conformance suite (pkg/database/contract).

// TestSearchLogsOverIngestionTime verifies that SearchLogs filters by the
// ingestion time (ADDED), not by the log's own timestamp. This is the
// driver-side companion of the contract suite's SearchLogs coverage.
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
func TestSelectCandiate(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)

	defer db.Close()

	var processes []*core.Process

	selectedProcess := db.selectCandidate(processes)
	assert.Nil(t, selectedProcess)

	process1 := utils.CreateTestProcess(core.GenerateRandomID())
	process2 := utils.CreateTestProcess(core.GenerateRandomID())
	processes = append(processes, process1)
	processes = append(processes, process2)

	selectedProcess = db.selectCandidate(processes)
	assert.NotNil(t, selectedProcess)
	assert.Equal(t, selectedProcess.ID, process1.ID)
}

func TestRemoveAllProcessesInProcessGraphsByColonyNameWithState(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)

	defer db.Close()

	colonyName := core.GenerateRandomID()
	processGraphID1 := core.GenerateRandomID()
	processGraphID2 := core.GenerateRandomID()
	processGraphID3 := core.GenerateRandomID()
	process1 := utils.CreateTestProcess(colonyName)
	process1.ProcessGraphID = processGraphID1
	err = db.AddProcess(process1)
	assert.Nil(t, err)
	attribute1 := core.CreateAttribute(process1.ID, colonyName, processGraphID1, core.IN, "test_key1", "test_value1")
	err = db.AddAttribute(attribute1)
	assert.Nil(t, err)

	process2 := utils.CreateTestProcess(colonyName)
	process2.ProcessGraphID = processGraphID2
	err = db.AddProcess(process2)
	assert.Nil(t, err)
	attribute2 := core.CreateAttribute(process2.ID, colonyName, processGraphID2, core.IN, "test_key1", "test_value1")
	err = db.AddAttribute(attribute2)
	assert.Nil(t, err)

	process3 := utils.CreateTestProcess(colonyName)
	process3.ProcessGraphID = processGraphID3
	err = db.AddProcess(process3)
	assert.Nil(t, err)
	attribute3 := core.CreateAttribute(process3.ID, colonyName, processGraphID3, core.IN, "test_key1", "test_value1")
	err = db.AddAttribute(attribute3)
	assert.Nil(t, err)

	process4 := utils.CreateTestProcess(colonyName)
	err = db.AddProcess(process4)
	assert.Nil(t, err)
	attribute4 := core.CreateAttribute(process4.ID, colonyName, "", core.IN, "test_key1", "test_value1")
	err = db.AddAttribute(attribute4)
	assert.Nil(t, err)

	err = db.SetProcessState(process1.ID, core.FAILED)
	assert.Nil(t, err)

	err = db.SetProcessState(process2.ID, core.FAILED)
	assert.Nil(t, err)

	err = db.SetProcessState(process3.ID, core.RUNNING)
	assert.Nil(t, err)

	err = db.SetProcessState(process4.ID, core.RUNNING)
	assert.Nil(t, err)

	runningProcesses, err := db.CountFailedProcesses()
	assert.Nil(t, err)
	assert.Equal(t, runningProcesses, 2)

	err = db.RemoveAllProcessesInProcessGraphsByColonyNameWithState(colonyName, core.FAILED)
	assert.Nil(t, err)

	runningProcesses, err = db.CountFailedProcesses()
	assert.Nil(t, err)
	assert.Equal(t, runningProcesses, 0)

	err = db.RemoveAllProcessesInProcessGraphsByColonyNameWithState(colonyName, core.RUNNING)
	assert.Nil(t, err)

	runningProcesses, err = db.CountRunningProcesses()
	assert.Nil(t, err)
	assert.Equal(t, runningProcesses, 1)
}

func TestSetWaitDeadline(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)

	defer db.Close()

	colony := core.CreateColony(core.GenerateRandomID(), "test_colony_name")
	process := utils.CreateTestProcess(colony.Name)
	err = db.AddProcess(process)
	assert.Nil(t, err)
	assert.Equal(t, process.ExecDeadline, time.Time{})

	err = db.SetWaitDeadline(process, time.Now())
	assert.Nil(t, err)

	processFromDB, err := db.GetProcessByID(process.ID)
	assert.Nil(t, err)
	assert.NotEqual(t, processFromDB.WaitDeadline, time.Time{})
}

func TestSetExecDeadline(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)

	defer db.Close()

	colony := core.CreateColony(core.GenerateRandomID(), "test_colony_name")
	process := utils.CreateTestProcess(colony.Name)
	err = db.AddProcess(process)
	assert.Nil(t, err)
	assert.Equal(t, process.ExecDeadline, time.Time{})

	err = db.SetExecDeadline(process, time.Now())
	assert.Nil(t, err)

	processFromDB, err := db.GetProcessByID(process.ID)
	assert.Nil(t, err)
	assert.NotEqual(t, processFromDB.ExecDeadline, time.Time{})
}
