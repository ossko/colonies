# Database Contract

This document defines the semantics every `database.Database` driver implementation
must honor. The interface in this package is the contract; this document is its
normative companion. Drivers live in their own packages (e.g. `pkg/database/postgresql`)
and must pass the shared behavioral test suite (`pkg/database/contract`) before being
registered in the factory: drivers activate it from a test file with
`contract.RegisterConformanceFactory(...)` plus a
`func TestConformance(t *testing.T) { contract.RunSuite(t) }` runner, e.g.
`pkg/database/postgresql/conformance_test.go`.

Drivers may be backed by SQL or non-SQL engines. The contract deliberately describes
*behavior*, not storage.

---

## 1. Scope and non-goals

- The fat `Database` interface (composed of the 16 sub-interfaces) is **frozen**.
  Drivers implement all of it. A method the engine cannot support returns an error
  of kind `ErrNotImplemented` — not a panic, not a silent no-op.
- Drivers must be safe for concurrent use from multiple goroutines; the Colonies
  server calls the interface freely from HTTP handlers and background workers.
- Multi-server deployments (several Colonies servers sharing one database) are
  valid. Any method documented as **atomic** below must remain correct under that
  concurrency. A driver that cannot guarantee the atomicity requirement (e.g. an
  embedded store without conditional writes) must refuse it (return
  `ErrNotImplemented` for the affected method) rather than approximate it, and is a
  **single-server engine**: it may only run with clustering disabled
  (single-node deployments). Server startup validates this pairing and refuses
  invalid combinations (see the deployment capability matrix in the abstraction
  plan).

## 2. Lifecycle

| Method | Contract |
|---|---|
| `Connect() error` | Establish the connection / open the storage. Idempotent or safe to call once; must not leak resources on repeated failure. Must return quickly on an unreachable backend (bounded timeout). |
| `Healthy() bool` | Non-blocking-ish, cheap reachability probe. Safe to call at any time, including before `Connect` or after `Close` (returns false then). |
| `Initialize() error` | Create tables/collections/indexes/structures. Idempotent: calling twice must succeed if the schema already exists. |
| `Drop() error` | Remove all driver-owned data structures. Missing structures may either be created implicitly or ignored. |
| `Close()` | Release resources. Subsequent operations may return driver errors; must not panic. |
| `ApplyRetentionPolicy(retentionPeriod int64)` | Delete finished, retained data older than `retentionPeriod` seconds: attributes, logs, successful processes and process graphs. See §8. |

## 3. Error contract

Drivers classify failures into the semantic kinds defined in `pkg/database/dberr`
(re-exported by `pkg/database`): `ErrNotFound`, `ErrAlreadyExists`, `ErrConflict`,
`ErrNotImplemented` — via the `dberr.NotFound/AlreadyExists/Conflict/NotImplemented`
constructors and the `IsNotFound/IsAlreadyExists/...` helpers.

Rules:

1. **Identity, not text**: consumers must test kinds with `errors.Is` / the `Is*`
   helpers. `err.Error()` text of dberr-wrapped errors is chosen by the driver.
2. **No leaking engine internals** as the *kind* signal; a raw driver error may be
   returned where no kind applies, but whenever the situation matches a defined
   kind, the driver must return a kind-classified error.
3. **Attention — error text crosses the RPC boundary.** Until server handlers own
   HTTP-status translation, driver messages are surfaced verbatim in ColoniesError
   responses, and some client/CI tests assert on substrings such as
   `"already exists"` / `"not found"`. Therefore: keep the *phrasing* of
   kind-classified errors stable ("\<entity\> ... already exists",
   "... not found") until handler-side translation exists.
4. `ErrNotImplemented` may wrap an explanation naming the engine limitation.

## 4. Not-found signaling convention

The repository uses a nil-result convention that drivers must preserve:

- **`Get*` single-entity methods return `(nil, nil)`** when the entity does not
  exist. They do NOT return `ErrNotFound`. (Exception: mid-population parse
  failures return driver errors.)
- **Mutating methods that act on a named/ID target** (`Assign`, `MarkFailed`,
  `SetAllocations`, `RemoveColonyByName`, snapshot getters, ...) return
  `ErrNotFound` when the target is missing.
- **`Find*`/list methods return an empty (possibly nil) slice**, no error.
- `Remove*` methods: the postgres driver splits into two families, and new
  drivers must do the same (conformance suite asserts both):
  - *existence-checking* — return `ErrNotFound` when the target is missing
    (e.g. `RemoveColonyByName`, `MarkFailed`);
  - *idempotent* — return `nil` regardless of prior existence
    (e.g. `RemoveCronByID`).
- **State transitions of `MarkSuccessful` are idempotent on terminal
  processes** (calling it on a `SUCCESS` process is a no-op returning their
  times), while `MarkFailed` refuses terminal transitions with `ErrConflict`.
- `SearchLogs` filters over the **ingestion time** of the row, not over the
  log's own `timestamp` field. Since `AddLog` always stamps ingestion time as
  now, the days-window semantics cannot be exercised through the interface;
  each driver covers this in its own internal tests
  (e.g. `TestSearchLogsOverIngestionTime`).
- **`Count*` methods return 0**, no error, on empty results.

## 5. Identity, naming and uniqueness

- All IDs (`colony.ID`, `executor.ID`, `process.ID`, ...) are strings generated
  by the server; drivers treat them opaquely.
- Colony names are globally unique; executor/user/blueprint/cron names are unique
  *within a colony*. Executor storage keys are `<colonyName>:<executorName>`.
- Violating a uniqueness rule must return `ErrAlreadyExists` — whether detected
  by pre-check or by a storage constraint (both paths must classify identically).

## 6. Per-interface semantics

### ColonyDatabase
- `AddColony` errors with `ErrAlreadyExists` on duplicate name (pre-check).
- `RenameColony`, `ChangeColonyID` (SecurityDatabase family) are plain updates;
  they do not verify existence unless stated in tests.

### UserDatabase
- `AddUser` errors with `ErrAlreadyExists` on duplicate name; user names are
  unique within a colony.

### ExecutorDatabase
- `AddExecutor` errors with `ErrAlreadyExists` for duplicate name **or** duplicate
  executor ID within the colony; both pre-check and constraint paths classify as
  `ErrAlreadyExists`.
- `ApproveExecutor`/`RejectExecutor`/`MarkAlive` update state/heartbeat columns.
- Metrics/capability fields are stored as JSON documents.

### FunctionDatabase
- Functions are keyed by `(colonyName, executorName, funcName)`;
  stats updates (`UpdateFunctionStats`) create or update the entry.

### ProcessDatabase — the most concurrency-sensitive interface
Process states: `WAITING → RUNNING → SUCCESS|FAILED` (see `pkg/core`).

- **`SelectAndAssign` is the atomic claim primitive.** Given executor attributes
  (type/limits/location) it must select *at most one* waiting, unassigned,
  parents-cleared process in the colony matching the executor-type and resource
  constraints, set it `RUNNING`, assigned, and return it. Under concurrent calls
  from N goroutines (possibly across servers), each waiting process must be
  claimed by exactly one caller — no double assignment, no lost claims. No
  candidate ⇒ return `(nil, nil)`.
  - The postgres driver implements this with a single
    `UPDATE ... WHERE id = (SELECT ... FOR UPDATE SKIP LOCKED ...)` statement.
  - Other engines must use equivalent primitives (conditional write /
    find-and-modify / transactions). Engines with none must return
    `ErrNotImplemented` for this method (and for `Assign`).
- `Ordering`: candidates are ordered by priority time (`PRIORITYTIME ASC`),
  oldest priority first.
- `Assign`: assigns a *specific* process to an executor; returns `ErrConflict`
  if already assigned.
- `Unassign` re-enters the process to `WAITING` (used on timeouts/retries).
- `MarkSuccessful` transitions `RUNNING → SUCCESS` and returns the process's
  waiting time and processing time in seconds. `ErrConflict` if the process is
  not in a valid state (`FAILED`, or `WAITING` never started).
- `MarkFailed` transitions to `FAILED`; `ErrNotFound` if missing; `ErrConflict`
  if already terminal.
- `SetInput/SetOutput/SetErrors/SetProcessState/SetParents/SetChildren/...` are
  field mutations by process ID; missing process ⇒ driver error (may be raw).
- Process/attribute coupling: marking a process terminal also updates the state
  of its attributes (`SetAttributeState`); retention deletes them together.

### AttributeDatabase
- Attributes address a `targetID` (usually a process ID) with type + key;
  `GetAttribute*` conflict-detect duplicates (`ErrConflict` when >1 found).
- Bulk removals by colony/process-graph/state are cascades for process deletion.

### ProcessGraphDatabase
- Mirrors process state transitions with the same `Find*`/`Count*` conventions;
  graphs carry an ordered `Roots` list.

### GeneratorDatabase
- Generators produce processes on triggers; `SetGeneratorLastRun` /
  `SetGeneratorFirstPack` are timestamp checkpoints. Args are ordered
  (`GetGeneratorArgs(generatorID, count)`).

### CronDatabase
- Unique `(colonyName, name)`; duplicate `AddCron` must yield
  `ErrAlreadyExists` even under parallel insert races (constraint-backed).
- `UpdateCron` persists scheduler checkpoints (nextRun/lastRun/prev graph).

### LogDatabase
- Append-only. `AddLog` never fails on duplicates. Read methods accept `limit`
  and `since`; latest-variants return the tail. `SearchLogs` is substring match
  over the last `days` days, up to `count` entries.

### FileDatabase
- File rows point to S3 objects (seaweedfs); the driver stores metadata only.
- A monotonically increasing per-database sequence number (`Seqnr`) is assigned
  per file; drivers may implement it as a sequence, counter document, or
  timestamp-derived value — it must be strictly increasing within a database.
- `GetLatestFileByName` etc. group by `(colonyName, label, name)`.

### SnapshotDatabase
- `CreateSnapshot` derives rows from existing file rows (label/name split);
  duplicate snapshot name ⇒ `ErrAlreadyExists`.
- Missing snapshot getters ⇒ `ErrNotFound`.

### BlueprintDatabase
- Definitions and blueprints both keyed by ID plus unique
  `(namespace=colonyName, name)`; duplicates ⇒ `ErrAlreadyExists`.
- `UpdateBlueprintStatus` merges a JSON status object.
- History rows are appended per generation; `GetBlueprintHistory` returns
  newest-first, up to `limit`.

### SecurityDatabase
- ID-rotation helpers (`ChangeColonyID`, `ChangeUserID`, `ChangeExecutorID`,
  `SetServerID`) perform cross-table updates; they must not create orphaned
  references.

### LocationDatabase
- Locations are per-colony named points; duplicates ⇒ `ErrAlreadyExists`.

## 7. Data-model conventions

- Table/collection names are prefixed with the configured `prefix` (e.g. `TEST_`).
  Prefixes are upper-case; two servers must be able to share one database using
  distinct prefixes without interference. The conformance suite relies on this.
- Time: UTC; driver must not depend on the server's `TZ` (the postgres driver
  currently uses it for the DSN — this is driver-internal and being phased out).
- Arrays-of-strings and JSON documents are first-class: drivers on engines
  without arrays must encode them as JSON text and decode transparently.

## 8. Retention

`ApplyRetentionPolicy(retentionPeriod)` deletes, in one call:
- attributes with `added < cutoff` and state `SUCCESS` (and their state updates),
- logs older than cutoff,
- successful processes / process graphs submitted before cutoff.

Partial progress (engine without cross-collection transactions) must favor
*deleting more rather than keeping stale data*; repeated calls converge.

## 9. Driver checklist (summary)

- [ ] Implements all 16 sub-interfaces; unsupported ⇒ `ErrNotImplemented`
- [ ] Kinds returned correctly (`dberr`); no consumer string-matching needed
- [ ] Declares whether it supports distributed claim semantics (drives the
      clustering pairing validation)
- [ ] `Get*` ⇒ `(nil, nil)` on missing; mutations ⇒ `ErrNotFound`
- [ ] Atomic `SelectAndAssign` (+ no-double-assign under distributed callers)
- [ ] Prefix isolation works (parallel test processes)
- [ ] Idempotent `Initialize`; `Drop` + `Initialize` round-trip is clean
- [ ] Passes `pkg/database/contract` conformance suite
