//go:build apicompat

// Package apicompat freezes the ColonyOS wire RPC API. See README.md for the
// rules. The suite talks to a real server binary over HTTP through pkg/client
// only; it must keep passing, unmodified, across all internal refactoring.
package apicompat

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/colonyos/colonies/pkg/client"
	"github.com/colonyos/colonies/pkg/core"
	"github.com/colonyos/colonies/pkg/security/crypto"
	"github.com/colonyos/colonies/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const executorType = "apicompat_executor"

var (
	apiClient      *client.ColoniesClient
	serverPrvKey   string
	colonyName     string
	colonyPrvKey   string
	userPrvKey     string
	userID         string
	executorPrvKey string
	executorName   string
)

func TestMain(m *testing.M) {
	code, err := run(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, "apicompat setup failed:", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func run(m *testing.M) (int, error) {
	binary := os.Getenv("COLONIES_SERVER_BINARY")
	if binary == "" {
		built, err := buildServerBinary()
		if err != nil {
			return 0, err
		}
		binary = built
	}

	keys := crypto.CreateCrypto()
	var err error
	serverPrvKey, err = keys.GeneratePrivateKey()
	if err != nil {
		return 0, err
	}
	serverID, err := keys.GenerateID(serverPrvKey)
	if err != nil {
		return 0, err
	}
	colonyPrvKey, err = keys.GeneratePrivateKey()
	if err != nil {
		return 0, err
	}
	userPrvKey, err = keys.GeneratePrivateKey()
	if err != nil {
		return 0, err
	}
	userID, err = keys.GenerateID(userPrvKey)
	if err != nil {
		return 0, err
	}

	ports, err := utils.FreePorts(5)
	if err != nil {
		return 0, err
	}
	apiPort, etcdClientPort, etcdPeerPort, relayPort, monitorPort := ports[0], ports[1], ports[2], ports[3], ports[4]

	etcdDataDir, err := os.MkdirTemp("", "apicompat-etcd-")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(etcdDataDir)

	env := append(os.Environ(),
		"TZ=UTC",
		"COLONIES_SERVER_HOST=localhost",
		fmt.Sprintf("COLONIES_SERVER_PORT=%d", apiPort),
		fmt.Sprintf("COLONIES_MONITOR_PORT=%d", monitorPort),
		"COLONIES_MONITOR_INTERVAL=1",
		"COLONIES_SERVER_ID="+serverID,
		"COLONIES_SERVER_PRVKEY="+serverPrvKey,
		"COLONIES_COLONY_NAME=apicompat",
		"COLONIES_COLONY_PRVKEY="+colonyPrvKey,
		"COLONIES_PRVKEY="+userPrvKey,
		"COLONIES_ID="+userID,
		"COLONIES_EXECUTOR_TYPE=cli",
		"COLONIES_TLS=false",
		"COLONIES_VERBOSE=false",
		"COLONIES_CRON_CHECKER_PERIOD=1000",
		"COLONIES_GENERATOR_CHECKER_PERIOD=1000",
		"COLONIES_EXCLUSIVE_ASSIGN=false",
		"COLONIES_ALLOW_EXECUTOR_REREGISTER=false",
		"COLONIES_RETENTION=false",
		"COLONIES_RETENTION_POLICY=200",
	)
	if os.Getenv("COLONIES_DB_HOST") == "" {
		env = append(env, "COLONIES_DB_HOST=localhost")
	}
	if os.Getenv("COLONIES_DB_PORT") == "" {
		env = append(env, "COLONIES_DB_PORT=5432")
	}
	if os.Getenv("COLONIES_DB_USER") == "" {
		env = append(env, "COLONIES_DB_USER=postgres")
	}
	if os.Getenv("COLONIES_DB_PASSWORD") == "" {
		env = append(env, "COLONIES_DB_PASSWORD=postgres")
	}

	// Reset the server's tables so --initdb starts from a clean slate
	dropCmd := exec.Command(binary, "database", "drop")
	dropCmd.Env = env
	dropCmd.Stdin = strings.NewReader("YES\n")
	dropCmd.Run() // Ignore errors: tables may not exist yet

	server := exec.Command(binary, "server", "start",
		"--initdb",
		"--insecure",
		"--port", fmt.Sprintf("%d", apiPort),
		"--etcdname", "apicompat",
		"--etcdhost", "localhost",
		"--etcdclientport", fmt.Sprintf("%d", etcdClientPort),
		"--etcdpeerport", fmt.Sprintf("%d", etcdPeerPort),
		"--relayport", fmt.Sprintf("%d", relayPort),
		"--etcddatadir", etcdDataDir,
	)
	server.Env = env
	server.Stdout = os.Stderr
	server.Stderr = os.Stderr
	if err := server.Start(); err != nil {
		return 0, err
	}
	defer func() {
		server.Process.Kill()
		server.Wait()
	}()

	apiClient = client.CreateColoniesClient("localhost", apiPort, true, false)
	if err := waitForServer(apiClient, 30*time.Second); err != nil {
		return 0, err
	}

	return m.Run(), nil
}

func buildServerBinary() (string, error) {
	dir, err := os.MkdirTemp("", "apicompat-bin-")
	if err != nil {
		return "", err
	}
	binary := filepath.Join(dir, "colonies")
	build := exec.Command("go", "build", "-o", binary, "./cmd/main.go")
	build.Dir = "../.."
	out, err := build.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("building server binary: %v\n%s", err, out)
	}
	return binary, nil
}

func waitForServer(c *client.ColoniesClient, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		_, _, lastErr = c.Version()
		if lastErr == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("server did not become ready: %v", lastErr)
}

// TestA_Version must run first (tests run alphabetically within a file):
// later tests depend on the colony and executor it creates.
func TestA_Version(t *testing.T) {
	// The build/version strings depend on ldflags, so only the call contract
	// is asserted
	_, _, err := apiClient.Version()
	assert.Nil(t, err)
}

func TestB_ColonySetup(t *testing.T) {
	keys := crypto.CreateCrypto()
	colonyID, err := keys.GenerateID(colonyPrvKey)
	require.Nil(t, err)

	colonyName = core.GenerateRandomID()
	colony := core.CreateColony(colonyID, colonyName)
	addedColony, err := apiClient.AddColony(colony, serverPrvKey)
	require.Nil(t, err)
	assert.Equal(t, colonyName, addedColony.Name)

	user := core.CreateUser(colonyName, userID, "apicompat-user", "user@apicompat.test", "")
	_, err = apiClient.AddUser(user, colonyPrvKey)
	require.Nil(t, err)

	executor, prvKey, err := utils.CreateTestExecutorWithKey(colonyName)
	require.Nil(t, err)
	executor.Type = executorType
	executorPrvKey = prvKey
	executorName = executor.Name

	_, err = apiClient.AddExecutor(executor, colonyPrvKey)
	require.Nil(t, err)
	err = apiClient.ApproveExecutor(colonyName, executor.Name, colonyPrvKey)
	require.Nil(t, err)
}

func newFuncSpec() *core.FunctionSpec {
	funcSpec := core.CreateEmptyFunctionSpec()
	funcSpec.Conditions.ColonyName = colonyName
	funcSpec.Conditions.ExecutorType = executorType
	funcSpec.FuncName = "apicompat_func"
	funcSpec.MaxExecTime = -1
	funcSpec.MaxWaitTime = -1
	return funcSpec
}

func TestC_SubmitAssignCloseSuccess(t *testing.T) {
	submitted, err := apiClient.Submit(newFuncSpec(), userPrvKey)
	require.Nil(t, err)
	assert.Equal(t, core.WAITING, submitted.State)

	fetched, err := apiClient.GetProcess(submitted.ID, executorPrvKey)
	require.Nil(t, err)
	assert.Equal(t, submitted.ID, fetched.ID)

	assigned, err := apiClient.Assign(colonyName, -1, "", "", executorPrvKey)
	require.Nil(t, err)
	assert.Equal(t, submitted.ID, assigned.ID)
	assert.Equal(t, core.RUNNING, assigned.State)

	err = apiClient.AddLog(assigned.ID, "apicompat log line", executorPrvKey)
	require.Nil(t, err)
	logs, err := apiClient.GetLogsByProcess(colonyName, assigned.ID, 10, executorPrvKey)
	require.Nil(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, "apicompat log line", logs[0].Message)

	err = apiClient.Close(assigned.ID, executorPrvKey)
	require.Nil(t, err)

	closed, err := apiClient.GetProcess(assigned.ID, executorPrvKey)
	require.Nil(t, err)
	assert.Equal(t, core.SUCCESS, closed.State)
}

func TestD_SubmitAssignFail(t *testing.T) {
	submitted, err := apiClient.Submit(newFuncSpec(), userPrvKey)
	require.Nil(t, err)

	assigned, err := apiClient.Assign(colonyName, -1, "", "", executorPrvKey)
	require.Nil(t, err)
	assert.Equal(t, submitted.ID, assigned.ID)

	err = apiClient.Fail(assigned.ID, []string{"apicompat failure"}, executorPrvKey)
	require.Nil(t, err)

	failed, err := apiClient.GetProcess(assigned.ID, executorPrvKey)
	require.Nil(t, err)
	assert.Equal(t, core.FAILED, failed.State)
	require.NotEmpty(t, failed.Errors)
	assert.Equal(t, "apicompat failure", failed.Errors[0])
}

func TestE_WorkflowDAG(t *testing.T) {
	//    task1
	//    /   \
	// task2  task3
	//    \   /
	//    task4
	workflowSpec := core.CreateWorkflowSpec(colonyName)
	names := []string{"task1", "task2", "task3", "task4"}
	specs := make(map[string]*core.FunctionSpec)
	for _, name := range names {
		spec := newFuncSpec()
		spec.NodeName = name
		specs[name] = spec
	}
	specs["task2"].AddDependency("task1")
	specs["task3"].AddDependency("task1")
	specs["task4"].AddDependency("task2")
	specs["task4"].AddDependency("task3")
	for _, name := range names {
		workflowSpec.AddFunctionSpec(specs[name])
	}

	graph, err := apiClient.SubmitWorkflowSpec(workflowSpec, userPrvKey)
	require.Nil(t, err)
	require.NotNil(t, graph)

	// Tasks become assignable in dependency order
	for i := 0; i < len(names); i++ {
		assigned, err := apiClient.Assign(colonyName, 10, "", "", executorPrvKey)
		require.Nil(t, err)
		err = apiClient.Close(assigned.ID, executorPrvKey)
		require.Nil(t, err)
	}

	deadline := time.Now().Add(30 * time.Second)
	for {
		finished, err := apiClient.GetProcessGraph(graph.ID, executorPrvKey)
		require.Nil(t, err)
		if finished.State == core.SUCCESS {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("workflow did not reach SUCCESS, state=%d", finished.State)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestF_SubscribeProcess(t *testing.T) {
	submitted, err := apiClient.Submit(newFuncSpec(), userPrvKey)
	require.Nil(t, err)

	subscription, err := apiClient.SubscribeProcess(colonyName, submitted.ID, executorType, core.RUNNING, 30, executorPrvKey)
	require.Nil(t, err)

	received := make(chan error, 1)
	go func() {
		select {
		case <-subscription.ProcessChan:
			received <- nil
		case err := <-subscription.ErrChan:
			received <- err
		}
	}()

	assigned, err := apiClient.Assign(colonyName, 10, "", "", executorPrvKey)
	require.Nil(t, err)
	assert.Equal(t, submitted.ID, assigned.ID)

	select {
	case err := <-received:
		assert.Nil(t, err)
	case <-time.After(30 * time.Second):
		t.Fatal("no realtime event received for RUNNING transition")
	}

	err = apiClient.Close(assigned.ID, executorPrvKey)
	require.Nil(t, err)
}

func TestG_Statistics(t *testing.T) {
	// Statistics require colony membership, so the executor key is used
	stats, err := apiClient.ColonyStatistics(colonyName, executorPrvKey)
	require.Nil(t, err)
	assert.True(t, stats.SuccessfulProcesses >= 1)
	assert.True(t, stats.FailedProcesses >= 1)
}
