// Command smokeclient exercises the core ColonyOS wire flows against a running
// server and exits non-zero on any failure.
//
// It is deliberately self-contained and restricted to long-stable client SDK
// calls, so it can be compiled against an OLDER release of this repository and
// run against a NEWER server binary. CI uses this to prove that existing
// executors and SDK clients keep working across server upgrades.
//
// Configuration via environment:
//
//	COLONIES_SERVER_HOST    server host (default localhost)
//	COLONIES_SERVER_PORT    server port (required)
//	COLONIES_SERVER_PRVKEY  server private key, used to create a colony (required)
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/colonyos/colonies/pkg/client"
	"github.com/colonyos/colonies/pkg/core"
	"github.com/colonyos/colonies/pkg/security/crypto"
)

func fail(step string, err error) {
	fmt.Fprintf(os.Stderr, "smokeclient: %s: %v\n", step, err)
	os.Exit(1)
}

func main() {
	host := os.Getenv("COLONIES_SERVER_HOST")
	if host == "" {
		host = "localhost"
	}
	portStr := os.Getenv("COLONIES_SERVER_PORT")
	serverPrvKey := os.Getenv("COLONIES_SERVER_PRVKEY")
	if portStr == "" || serverPrvKey == "" {
		fail("config", fmt.Errorf("COLONIES_SERVER_PORT and COLONIES_SERVER_PRVKEY must be set"))
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		fail("config", err)
	}

	c := client.CreateColoniesClient(host, port, true, false)

	// Wait for the server
	deadline := time.Now().Add(30 * time.Second)
	for {
		_, _, err = c.Version()
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			fail("version", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	keys := crypto.CreateCrypto()

	colonyPrvKey, err := keys.GeneratePrivateKey()
	if err != nil {
		fail("keygen", err)
	}
	colonyID, err := keys.GenerateID(colonyPrvKey)
	if err != nil {
		fail("keygen", err)
	}
	userPrvKey, err := keys.GeneratePrivateKey()
	if err != nil {
		fail("keygen", err)
	}
	userID, err := keys.GenerateID(userPrvKey)
	if err != nil {
		fail("keygen", err)
	}
	executorPrvKey, err := keys.GeneratePrivateKey()
	if err != nil {
		fail("keygen", err)
	}
	executorID, err := keys.GenerateID(executorPrvKey)
	if err != nil {
		fail("keygen", err)
	}

	colonyName := core.GenerateRandomID()
	colony := core.CreateColony(colonyID, colonyName)
	if _, err := c.AddColony(colony, serverPrvKey); err != nil {
		fail("addcolony", err)
	}

	user := core.CreateUser(colonyName, userID, "smoke-user", "smoke@test", "")
	if _, err := c.AddUser(user, colonyPrvKey); err != nil {
		fail("adduser", err)
	}

	executorType := "smoke_executor"
	now := time.Now()
	executor := core.CreateExecutor(executorID, executorType, "smoke-executor", colonyName, now, now)
	if _, err := c.AddExecutor(executor, colonyPrvKey); err != nil {
		fail("addexecutor", err)
	}
	if err := c.ApproveExecutor(colonyName, executor.Name, colonyPrvKey); err != nil {
		fail("approveexecutor", err)
	}

	funcSpec := core.CreateEmptyFunctionSpec()
	funcSpec.Conditions.ColonyName = colonyName
	funcSpec.Conditions.ExecutorType = executorType
	funcSpec.FuncName = "smoke_func"
	funcSpec.MaxExecTime = -1
	funcSpec.MaxWaitTime = -1

	submitted, err := c.Submit(funcSpec, userPrvKey)
	if err != nil {
		fail("submit", err)
	}

	assigned, err := c.Assign(colonyName, 10, "", "", executorPrvKey)
	if err != nil {
		fail("assign", err)
	}
	if assigned.ID != submitted.ID {
		fail("assign", fmt.Errorf("assigned process %s, expected %s", assigned.ID, submitted.ID))
	}

	if err := c.AddLog(assigned.ID, "smoke log line", executorPrvKey); err != nil {
		fail("addlog", err)
	}

	if err := c.Close(assigned.ID, executorPrvKey); err != nil {
		fail("close", err)
	}

	closed, err := c.GetProcess(assigned.ID, executorPrvKey)
	if err != nil {
		fail("getprocess", err)
	}
	if closed.State != core.SUCCESS {
		fail("getprocess", fmt.Errorf("state %d, expected SUCCESS", closed.State))
	}

	fmt.Println("smokeclient: all wire flows OK")
}
