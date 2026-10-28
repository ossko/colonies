package contract

import (
	"sync"
	"testing"

	"github.com/colonyos/colonies/pkg/core"
	"github.com/colonyos/colonies/pkg/database/dberr"
	"github.com/colonyos/colonies/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// The tests in this file assert contract-defining semantics that go beyond a
// single API call: the nil-result convention (CONTRACT.md §4) and the atomic
// claim primitive (CONTRACT.md §6, ProcessDatabase).

// TestNilResultConvention verifies that Get* methods return (nil, nil) when
// the entity does not exist, while mutating methods return ErrNotFound, and
// invalid transitions return ErrConflict.
func TestNilResultConvention(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)

	defer db.Close()

	colonyName := core.GenerateRandomID()

	// Get*, the nil-result convention: no error, no result
	colony, err := db.GetColonyByName(colonyName)
	assert.Nil(t, err)
	assert.Nil(t, colony)

	colony, err = db.GetColonyByID(colonyName)
	assert.Nil(t, err)
	assert.Nil(t, colony)

	executor, err := db.GetExecutorByName(colonyName, "no_such_executor")
	assert.Nil(t, err)
	assert.Nil(t, executor)

	user, err := db.GetUserByName(colonyName, "no_such_user")
	assert.Nil(t, err)
	assert.Nil(t, user)

	process, err := db.GetProcessByID(colonyName)
	assert.Nil(t, err)
	assert.Nil(t, process)

	// Clean-slate list semantics: empty means empty, no error
	executors, err := db.GetExecutorsByColonyName(colonyName)
	assert.Nil(t, err)
	assert.Len(t, executors, 0)

	// Mutations on a missing entity: ErrNotFound for existence-checking
	// mutations (CONTRACT.md §4); Remove* methods are idempotent and return
	// nil - verified below
	err = db.RemoveColonyByName(colonyName)
	assert.True(t, dberr.IsNotFound(err), err)

	err = db.MarkFailed(colonyName, []string{"err"})
	assert.True(t, dberr.IsNotFound(err), err)

	err = db.RemoveCronByID(colonyName)
	assert.Nil(t, err) // idempotent remove of a missing cron
}

// TestErrorSentinels verifies the ErrAlreadyExists/ErrConflict kinds of
// CONTRACT.md §3 through the public API.
func TestErrorSentinels(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)

	defer db.Close()

	colony := core.CreateColony(core.GenerateRandomID(), core.GenerateRandomID())
	err = db.AddColony(colony)
	assert.Nil(t, err)

	// Duplicate colony name: pre-check path
	err = db.AddColony(colony)
	assert.True(t, dberr.IsAlreadyExists(err), err)

	executor := utils.CreateTestExecutor(colony.Name)
	err = db.AddExecutor(executor)
	assert.Nil(t, err)

	// Assigning a process twice: conflict with the state, not an error kind
	// of its own - the second Assign must fail with ErrConflict
	process := utils.CreateTestProcess(colony.Name)
	err = db.AddProcess(process)
	assert.Nil(t, err)

	err = db.Assign(executor.ID, process)
	assert.Nil(t, err)

	err = db.Assign(executor.ID, process)
	assert.True(t, dberr.IsConflict(err), err)

	// Closing a running process twice: MarkSuccessful is idempotent on an
	// already terminal process, but MarkFailed must refuse
	_, _, err = db.MarkSuccessful(process.ID)
	assert.Nil(t, err)

	_, _, err = db.MarkSuccessful(process.ID)
	assert.Nil(t, err)

	err = db.MarkFailed(process.ID, []string{"err"})
	assert.True(t, dberr.IsConflict(err), err)
}

// TestSelectAndAssignClaimRace verifies the atomicity guarantee of the claim
// primitive: N concurrent callers competing for one waiting process must
// yield exactly one winner and N-1 empty results - no double assignment, no
// lost claims (CONTRACT.md §6, the most concurrency-sensitive contract).
func TestSelectAndAssignClaimRace(t *testing.T) {
	db, err := PrepareTestsWithPrefix("TEST_CONTRACT_CLAIM_RACE")
	assert.Nil(t, err)

	defer db.Close()

	colony := core.CreateColony(core.GenerateRandomID(), core.GenerateRandomID())
	err = db.AddColony(colony)
	assert.Nil(t, err)

	executor := utils.CreateTestExecutor(colony.Name)
	err = db.AddExecutor(executor)
	assert.Nil(t, err)

	const claimants = 16

	processes := make([]*core.Process, claimants)
	for i := range processes {
		processes[i] = utils.CreateTestProcess(colony.Name)
		err := db.AddProcess(processes[i])
		assert.Nil(t, err)
	}

	winners := make([]*core.Process, claimants)
	errs := make([]error, claimants)

	var wg sync.WaitGroup
	for i := 0; i < claimants; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			winners[i], errs[i] = db.SelectAndAssign(colony.Name, executor.ID, executor.Name,
				executor.Type, executor.LocationName, 1, 1, 1, 1, 1, 1, 1)
		}(i)
	}
	wg.Wait()

	won := 0
	claimedIDs := make(map[string]bool)
	for i := 0; i < claimants; i++ {
		assert.Nil(t, errs[i])
		if winners[i] != nil {
			won++
			assert.False(t, claimedIDs[winners[i].ID], "process claimed twice: %s", winners[i].ID)
			claimedIDs[winners[i].ID] = true
			assert.Equal(t, core.RUNNING, winners[i].State)
			assert.True(t, winners[i].IsAssigned)
			assert.Equal(t, executor.ID, winners[i].AssignedExecutorID)
		}
	}

	// Every waiting process must be claimed by exactly one caller
	assert.Equal(t, claimants, won)

	running, err := db.FindAllRunningProcesses()
	assert.Nil(t, err)
	assert.Len(t, running, claimants)
	for _, p := range running {
		assert.Equal(t, executor.ID, p.AssignedExecutorID)
	}
}

// TestPrefixIsolation verifies that two prefix-scoped databases sharing one
// backend do not see each other's data (CONTRACT.md §7).
func TestPrefixIsolation(t *testing.T) {
	db1, err := PrepareTestsWithPrefix("TEST_CONTRACT_ISO_A")
	assert.Nil(t, err)
	defer db1.Close()

	db2, err := PrepareTestsWithPrefix("TEST_CONTRACT_ISO_B")
	assert.Nil(t, err)
	defer db2.Close()

	colonyA := core.CreateColony(core.GenerateRandomID(), core.GenerateRandomID())
	err = db1.AddColony(colonyA)
	assert.Nil(t, err)

	colonies, err := db2.GetColonies()
	assert.Nil(t, err)
	assert.Len(t, colonies, 0)

	colonies, err = db1.GetColonies()
	assert.Nil(t, err)
	assert.Len(t, colonies, 1)
}
