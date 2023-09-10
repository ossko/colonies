package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateEmptyFunctionSpec(t *testing.T) {
	funcSpec := CreateEmptyFunctionSpec()
	assert.NotNil(t, funcSpec)
}

func TestFunctionSpecJSON(t *testing.T) {
	colonyID := GenerateRandomID()
	executorType := "test_executor_type"
	executor1ID := GenerateRandomID()
	executor2ID := GenerateRandomID()
	maxWaitTime := -1
	maxExecTime := -1
	maxRetries := 3
	env := make(map[string]string)
	env["test_key"] = "test_value"

	args := make([]interface{}, 1)
	args[0] = "test_arg"
	kwargs := make(map[string]interface{}, 1)
	kwargs["0"] = "test_arg"

	var fs []*SyncDir
	syncDir1 := &SyncDir{Label: "test_label1", SnapshotID: "test_snapshotid1", Dir: "test_dir1"}
	syncDir2 := &SyncDir{Label: "test_label2", SnapshotID: "test_snapshotid2", Dir: "test_dir2"}
	fs = append(fs, syncDir1)
	fs = append(fs, syncDir2)

	funcSpec := CreateFunctionSpec("test_name", "test_func", args, kwargs, colonyID, []string{executor1ID, executor2ID}, executorType, maxWaitTime, maxExecTime, maxRetries, env, []string{"test_name2"}, 5, "test_label")

	funcSpec.Filesystem = fs

	funcSpec.Conditions.Nodes = 10
	funcSpec.Conditions.CPU = "1000m"
	funcSpec.Conditions.Processes = 10
	funcSpec.Conditions.ProcessesPerNode = 1
	funcSpec.Conditions.Memory = "10G"
	funcSpec.Conditions.Storage = "10G"
	funcSpec.Conditions.GPU = GPU{Name: "test_name1", Count: 1, Memory: "11G"}
	funcSpec.Conditions.WallTime = 1000

	jsonString, err := funcSpec.ToJSON()
	assert.Nil(t, err)

	funcSpec2, err := ConvertJSONToFunctionSpec(jsonString + "error")
	assert.NotNil(t, err)

	funcSpec2, err = ConvertJSONToFunctionSpec(jsonString)
	assert.Nil(t, err)

	assert.Equal(t, funcSpec.Conditions.ColonyID, funcSpec2.Conditions.ColonyID)
	assert.Equal(t, funcSpec.MaxExecTime, funcSpec2.MaxExecTime)
	assert.Equal(t, funcSpec.MaxRetries, funcSpec2.MaxRetries)
	assert.Equal(t, funcSpec.Conditions.ExecutorIDs, funcSpec2.Conditions.ExecutorIDs)
	assert.Contains(t, funcSpec.Conditions.ExecutorIDs, executor1ID)
	assert.Contains(t, funcSpec.Conditions.ExecutorIDs, executor2ID)
	assert.Equal(t, funcSpec.Conditions.ExecutorType, funcSpec2.Conditions.ExecutorType)
	assert.Equal(t, funcSpec.Env, funcSpec2.Env)
}

func TestFunctionSpecEquals(t *testing.T) {
	colonyID := GenerateRandomID()
	executorType := "test_executor_type"
	executor1ID := GenerateRandomID()
	executor2ID := GenerateRandomID()
	executor3ID := GenerateRandomID()
	maxWaitTime := -1
	maxExecTime := -1
	maxRetries := 3
	env := make(map[string]string)
	env["test_key"] = "test_value"

	env2 := make(map[string]string)
	env2["test_key2"] = "test_value2"

	args := make([]interface{}, 1)
	args[0] = "test_arg"
	kwargs := make(map[string]interface{}, 1)
	kwargs["0"] = "test_arg"

	var fs []*SyncDir
	syncDir1 := &SyncDir{Label: "test_label1", SnapshotID: "test_snapshotid1", Dir: "test_dir1"}
	syncDir2 := &SyncDir{Label: "test_label2", SnapshotID: "test_snapshotid2", Dir: "test_dir2"}
	fs = append(fs, syncDir1)
	fs = append(fs, syncDir2)

	functionSpec1 := CreateFunctionSpec("test_name", "test_func", args, kwargs, colonyID, []string{executor1ID, executor2ID}, executorType, maxWaitTime, maxExecTime, maxRetries, env, []string{}, 1, "test_label")

	functionSpec1.Filesystem = fs

	args = make([]interface{}, 1)
	args[0] = "test_arg2"

	functionSpec2 := CreateFunctionSpec("test_name", "test_func", args, kwargs, colonyID, []string{executor3ID}, executorType+"2", 200, 4, 2, env2, []string{}, 1, "test_label")

	assert.True(t, functionSpec1.Equals(functionSpec1))
	assert.False(t, functionSpec1.Equals(nil))
	assert.False(t, functionSpec1.Equals(functionSpec2))
}
