package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

type Postgresql interface {
	Begin() (*sql.Tx, error)
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	Close() error
	Conn(ctx context.Context) (*sql.Conn, error)
	Driver() driver.Driver
	Exec(query string, args ...any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Ping() error
	PingContext(ctx context.Context) error
	Prepare(query string) (*sql.Stmt, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	SetConnMaxIdleTime(d time.Duration)
	SetConnMaxLifetime(d time.Duration)
	SetMaxIdleConns(n int)
	SetMaxOpenConns(n int)
	Stats() sql.DBStats
}

type SQLiteDatabase struct {
	postgresql  Postgresql
	dbHost      string
	dbPort      int
	dbUser      string
	dbPassword  string
	dbName      string
	dbPrefix    string
	timescaleDB bool

	// SQLite-specific: path is the open target ('/path/colonies.db' or
	// ':memory:'), set by the driver factory.
	path          string
	journal       string // WAL (default) | OFF
	busyTimeoutMs string
}

func CreateSQLiteDatabase(dbHost string, dbPort int, dbUser string, dbPassword string, dbName string, dbPrefix string, timescaleDB bool) *SQLiteDatabase {
	return &SQLiteDatabase{dbHost: dbHost, dbPort: dbPort, dbUser: dbUser, dbPassword: dbPassword, dbName: dbName, dbPrefix: dbPrefix, timescaleDB: timescaleDB}
}

func (db *SQLiteDatabase) Connect() error {
	if db.path == "" {
		return errors.New("sqlite: storage path not set; configure the driver with a DSN or the \"path\" option")
	}

	dsn := db.path
	if db.path != ":memory:" {
		if strings.Contains(dsn, "?") {
			// The DSN may already carry pragmas
		} else {
			dsn = "file:" + dsn
		}
	}
	// SQLite is single-writer: serialize database access through one pooled
	// connection so concurrent goroutines queue instead of failing with
	// SQLITE_BUSY. WAL gives multi-process readers; busy_timeout covers
	// cross-process write contention.
	openDSN := dsn + dsnSeparator(dsn) + "_pragma=busy_timeout(" + busyTimeoutOr(db.busyTimeoutMs, "10000") + ")&_pragma=journal_mode(" + journalModeOr(db.journal) + ")&_pragma=foreign_keys(0)"

	sqliteDB, err := sql.Open("sqlite", openDSN)
	if err != nil {
		return err
	}
	db.postgresql = sqliteDB

	// Configure connection pool
	maxOpenConns := 1
	maxIdleConns := 1
	connMaxLifetime := 5 * time.Minute
	connMaxIdleTime := 1 * time.Minute

	if v := os.Getenv("COLONIES_DB_MAX_OPEN_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxOpenConns = n
			maxIdleConns = n
		}
	}
	if v := os.Getenv("COLONIES_DB_MAX_IDLE_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxIdleConns = n
		}
	}
	if v := os.Getenv("COLONIES_DB_CONN_MAX_LIFETIME"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			connMaxLifetime = time.Duration(n) * time.Second
		}
	}
	if v := os.Getenv("COLONIES_DB_CONN_MAX_IDLE_TIME"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			connMaxIdleTime = time.Duration(n) * time.Second
		}
	}

	db.postgresql.SetMaxOpenConns(maxOpenConns)
	db.postgresql.SetMaxIdleConns(maxIdleConns)
	db.postgresql.SetConnMaxLifetime(connMaxLifetime)
	db.postgresql.SetConnMaxIdleTime(connMaxIdleTime)

	log.WithFields(log.Fields{
		"Path":            db.path,
		"MaxOpenConns":    maxOpenConns,
		"MaxIdleConns":    maxIdleConns,
		"ConnMaxLifetime": connMaxLifetime,
	}).Info("SQLite database pool configured")

	err = db.postgresql.Ping()
	if err != nil {
		return err
	}

	return nil
}

// dsnSeparator returns the query separator given the DSN form (paths with an
// existing query string take '&', everything else takes '?').
func dsnSeparator(dsn string) string {
	if strings.Contains(dsn, "?") {
		return "&"
	}
	return "?"
}

func busyTimeoutOr(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func journalModeOr(journal string) string {
	if journal != "" {
		return journal
	}
	return "WAL"
}

func (db *SQLiteDatabase) Close() {
	db.postgresql.Close()
}

// Healthy reports whether the PostgreSQL server is reachable. It is safe to
// call at any time after Connect and is intended for periodic health checks.
func (db *SQLiteDatabase) Healthy() bool {
	if db.postgresql == nil {
		return false
	}
	return db.postgresql.Ping() == nil
}

func (db *SQLiteDatabase) dropUsersTable() error {
	sqlStatement := `DROP TABLE ` + db.dbPrefix + `USERS`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropColoniesTable() error {
	sqlStatement := `DROP TABLE ` + db.dbPrefix + `COLONIES`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropExecutorsTable() error {
	sqlStatement := `DROP TABLE ` + db.dbPrefix + `EXECUTORS`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropNodesTable() error {
	sqlStatement := `DROP TABLE ` + db.dbPrefix + `NODES`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropFunctionsTable() error {
	sqlStatement := `DROP TABLE ` + db.dbPrefix + `FUNCTIONS`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropProcessesTable() error {
	sqlStatement := `DROP TABLE ` + db.dbPrefix + `PROCESSES`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropLogTable() error {
	sqlStatement := `DROP TABLE ` + db.dbPrefix + `LOGS`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropFileTable() error {
	sqlStatement := `DROP TABLE IF EXISTS ` + db.dbPrefix + `FILES`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropSnapshotTable() error {
	sqlStatement := `DROP TABLE ` + db.dbPrefix + `SNAPSHOTS`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropAttributesTable() error {
	sqlStatement := `DROP TABLE ` + db.dbPrefix + `ATTRIBUTES`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropProcessGraphsTable() error {
	sqlStatement := `DROP TABLE ` + db.dbPrefix + `PROCESSGRAPHS`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropGeneratorsTable() error {
	sqlStatement := `DROP TABLE ` + db.dbPrefix + `GENERATORS`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropGeneratorArgsTable() error {
	sqlStatement := `DROP TABLE ` + db.dbPrefix + `GENERATORARGS`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropCronsTable() error {
	sqlStatement := `DROP TABLE ` + db.dbPrefix + `CRONS`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropBlueprintDefinitionsTable() error {
	sqlStatement := `DROP TABLE IF EXISTS ` + db.dbPrefix + `BLUEPRINTDEFINITIONS`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropBlueprintsTable() error {
	sqlStatement := `DROP TABLE IF EXISTS ` + db.dbPrefix + `BLUEPRINTS`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropBlueprintHistoryTable() error {
	sqlStatement := `DROP TABLE IF EXISTS ` + db.dbPrefix + `BLUEPRINT_HISTORY`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropLocationsTable() error {
	sqlStatement := `DROP TABLE IF EXISTS ` + db.dbPrefix + `LOCATIONS`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) dropServerTable() error {
	sqlStatement := `DROP TABLE ` + db.dbPrefix + `SERVER`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) Drop() error {
	err := db.dropUsersTable()
	if err != nil {
		return err
	}

	err = db.dropColoniesTable()
	if err != nil {
		return err
	}

	err = db.dropExecutorsTable()
	if err != nil {
		return err
	}

	err = db.dropNodesTable()
	if err != nil {
		return err
	}

	err = db.dropFunctionsTable()
	if err != nil {
		return err
	}

	err = db.dropProcessesTable()
	if err != nil {
		return err
	}

	err = db.dropLogTable()
	if err != nil {
		return err
	}

	err = db.dropFileTable()
	if err != nil {
		return err
	}

	err = db.dropSnapshotTable()
	if err != nil {
		return err
	}

	err = db.dropAttributesTable()
	if err != nil {
		return err
	}

	err = db.dropProcessGraphsTable()
	if err != nil {
		return err
	}

	err = db.dropGeneratorsTable()
	if err != nil {
		return err
	}

	err = db.dropGeneratorArgsTable()
	if err != nil {
		return err
	}

	err = db.dropCronsTable()
	if err != nil {
		return err
	}

	err = db.dropBlueprintDefinitionsTable()
	if err != nil {
		return err
	}

	err = db.dropBlueprintsTable()
	if err != nil {
		return err
	}

	err = db.dropBlueprintHistoryTable()
	if err != nil {
		return err
	}

	err = db.dropLocationsTable()
	if err != nil {
		return err
	}

	err = db.dropServerTable()
	if err != nil {
		return err
	}

	return nil
}

// createHypertables exists only in TimescaleDB deployments; the SQLite
// driver keeps plain indexes for the logs table instead.

func (db *SQLiteDatabase) createServerTable() error {
	sqlStatement := `CREATE TABLE IF NOT EXISTS ` + db.dbPrefix + `SERVER (SERVER_ID TEXT PRIMARY KEY NOT NULL)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createColoniesTable() error {
	sqlStatement := `CREATE TABLE IF NOT EXISTS ` + db.dbPrefix + `COLONIES (NAME TEXT PRIMARY KEY NOT NULL, COLONY_ID TEXT NOT NULL)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createUsersTable() error {
	sqlStatement := `CREATE TABLE IF NOT EXISTS ` + db.dbPrefix + `USERS (NAME TEXT PRIMARY KEY NOT NULL, USER_ID TEXT NOT NULL, COLONY_NAME TEXT NOT NULL, EMAIL TEXT NOT NULL, PHONE TEXT NOT NULL)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createExecutorsTable() error {
	sqlStatement := `CREATE TABLE IF NOT EXISTS ` + db.dbPrefix + `EXECUTORS (NAME TEXT PRIMARY KEY NOT NULL, EXECUTOR_TYPE TEXT NOT NULL, EXECUTOR_ID TEXT NOT NULL, COLONY_NAME TEXT NOT NULL, STATE INTEGER, REQUIRE_FUNC_REG NUMERIC, COMMISSIONTIME DATETIME, LASTHEARDFROM DATETIME, LONG REAL, LAT REAL, LOCNAME TEXT, LOCDESC TEXT, HARDWARE TEXT, SOFTWARE TEXT, ALLOCATIONS TEXT NOT NULL, NODE_ID TEXT, BLUEPRINT_ID TEXT, BLUEPRINT_GEN BIGINT, LOCATION_NAME TEXT)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createNodesTable() error {
	sqlStatement := `CREATE TABLE IF NOT EXISTS ` + db.dbPrefix + `NODES (ID TEXT PRIMARY KEY NOT NULL, NAME TEXT NOT NULL, COLONY_NAME TEXT NOT NULL, LOCATION TEXT, PLATFORM TEXT, ARCHITECTURE TEXT, CPU INTEGER, MEMORY BIGINT, GPU INTEGER, CAPABILITIES TEXT, LABELS TEXT, EXECUTORS TEXT, STATE TEXT, LAST_SEEN DATETIME, CREATED DATETIME)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}
	return nil
}

func (db *SQLiteDatabase) createFunctionsTable() error {
	sqlStatement := `CREATE TABLE ` + db.dbPrefix + `FUNCTIONS (FUNCTION_ID TEXT PRIMARY KEY NOT NULL, EXECUTOR_NAME TEXT NOT NULL, EXECUTOR_TYPE TEXT NOT NULL, COLONY_NAME TEXT NOT NULL, FUNCNAME TEXT NOT NULL, DESCRIPTION TEXT, ARGS TEXT, COUNTER INTEGER, MINWAITTIME FLOAT, MAXWAITTIME FLOAT, MINEXECTIME FLOAT, MAXEXECTIME FLOAT, AVGWAITTIME FLOAT, AVGEXECTIME FLOAT)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createProcessesTable() error {
	sqlStatement := `CREATE TABLE ` + db.dbPrefix + `PROCESSES (PROCESS_ID TEXT PRIMARY KEY NOT NULL, TARGET_COLONY_NAME TEXT NOT NULL, TARGET_EXECUTOR_NAMES TEXT, ASSIGNED_EXECUTOR_ID TEXT, STATE INTEGER, IS_ASSIGNED NUMERIC, EXECUTOR_TYPE TEXT, SUBMISSION_TIME DATETIME, START_TIME DATETIME, END_TIME DATETIME, WAIT_DEADLINE DATETIME, EXEC_DEADLINE DATETIME, ERRORS TEXT, NODENAME TEXT, FUNCNAME TEXT, ARGS TEXT, KWARGS TEXT, MAX_WAIT_TIME INTEGER, MAX_EXEC_TIME INTEGER, RETRIES INTEGER, MAX_RETRIES INTEGER, DEPENDENCIES TEXT, PRIORITY INTEGER, PRIORITYTIME BIGINT, WAIT_FOR_PARENTS NUMERIC, PARENTS TEXT, CHILDREN TEXT, PROCESSGRAPH_ID TEXT, INPUT TEXT, OUTPUT TEXT, LABEL TEXT, FS TEXT, NODES INTEGER, CPU BIGINT, PROCESSES INTEGER, PROCESSES_PER_NODE INTEGER, MEMORY BIGINT, STORAGE BIGINT, GPUNAME TEXT, GPUCOUNT TEXT, GPUMEM BIGINT, WALLTIME BIGINT, INITIATOR_ID TEXT NOT NULL, INITIATOR_NAME TEXT NOT NULL, BLUEPRINT TEXT, CHANNELS TEXT, LOCATION_NAME TEXT)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createLogTable() error {
	sqlStatement := `CREATE TABLE ` + db.dbPrefix + `LOGS (PROCESS_ID TEXT, COLONY_NAME TEXT NOT NULL, EXECUTOR_NAME TEXT NOT NULL, TS BIGINT, MSG TEXT NOT NULL, ADDED DATETIME)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createFileTable() error {
	sqlStatement := `CREATE TABLE ` + db.dbPrefix + `FILES (FILE_ID TEXT PRIMARY KEY NOT NULL, COLONY_NAME TEXT NOT NULL, LABEL TEXT NOT NULL, NAME TEXT NOT NULL, SIZE INTEGER, SEQNR INTEGER, CHECKSUM TEXT, CHECKSUM_ALG TEXT, ADDED DATETIME, PROTOCOL TEXT, S3_SERVER TEXT, S3_PORT INTEGER, S3_TLS NUMERIC, S3_ACCESSKEY TEXT, S3_SECRETKEY TEXT, S3_REGION TEXT, S3_ENCKEY TEXT, S3_ENCALG TEXT, S3_OBJ TEXT, S3_BUCKET TEXT)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createSnapshotTable() error {
	sqlStatement := `CREATE TABLE ` + db.dbPrefix + `SNAPSHOTS (SNAPSHOT_ID TEXT PRIMARY KEY NOT NULL, COLONY_NAME TEXT NOT NULL, LABEL TEXT NOT NULL, NAME TEXT NOT NULL UNIQUE, FILE_IDS TEXT, ADDED DATETIME)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createAttributesTable() error {
	sqlStatement := `CREATE TABLE ` + db.dbPrefix + `ATTRIBUTES (ATTRIBUTE_ID TEXT PRIMARY KEY NOT NULL, KEY TEXT NOT NULL, VALUE TEXT NOT NULL, ATTRIBUTE_TYPE INTEGER, TARGET_ID TEXT NOT NULL, TARGET_COLONY_NAME TEXT NOT NULL, PROCESSGRAPH_ID TEXT NOT NULL, ADDED DATETIME, STATE INTEGER)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createProcessGraphsTable() error {
	sqlStatement := `CREATE TABLE ` + db.dbPrefix + `PROCESSGRAPHS (PROCESSGRAPH_ID TEXT PRIMARY KEY NOT NULL, TARGET_COLONY_NAME TEXT NOT NULL, ROOTS TEXT, STATE INTEGER, SUBMISSION_TIME DATETIME, START_TIME DATETIME, END_TIME DATETIME, INITIATOR_ID TEXT NOT NULL, INITIATOR_NAME TEXT NOT NULL)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createGeneratorsTable() error {
	sqlStatement := `CREATE TABLE ` + db.dbPrefix + `GENERATORS (GENERATOR_ID TEXT PRIMARY KEY NOT NULL, COLONY_NAME TEXT NOT NULL, NAME TEXT NOT NULL, WORKFLOW_SPEC TEXT NOT NULL, TRIGGER INTEGER, TIMEOUT INTEGER, LASTRUN DATETIME, FIRSTPACK DATETIME, INITIATOR_ID TEXT NOT NULL, INITIATOR_NAME TEXT NOT NULL)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createGeneratorArgsTable() error {
	sqlStatement := `CREATE TABLE ` + db.dbPrefix + `GENERATORARGS (GENERATORARG_ID TEXT PRIMARY KEY NOT NULL, GENERATOR_ID TEXT NOT NULL, COLONY_NAME TEXT NOT NULL, ARG TEXT NOT NULL)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createCronsTable() error {
	sqlStatement := `CREATE TABLE ` + db.dbPrefix + `CRONS (CRON_ID TEXT PRIMARY KEY NOT NULL, COLONY_NAME TEXT NOT NULL, NAME TEXT NOT NULL, CRON_EXPR TEXT NOT NULL, INTERVAL INT, RANDOM NUMERIC, NEXT_RUN DATETIME, LAST_RUN DATETIME, WORKFLOW_SPEC TEXT NOT NULL, PREV_PROCESSGRAPH_ID TEXT NOT NULL, WAIT_FOR_PREV_PROCESSGRAPH NUMERIC, INITIATOR_ID TEXT NOT NULL, INITIATOR_NAME TEXT NOT NULL, UNIQUE(COLONY_NAME, NAME))`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createBlueprintDefinitionsTable() error {
	sqlStatement := `CREATE TABLE IF NOT EXISTS ` + db.dbPrefix + `BLUEPRINTDEFINITIONS (ID TEXT PRIMARY KEY NOT NULL, COLONY_NAME TEXT NOT NULL, NAME TEXT NOT NULL, API_GROUP TEXT NOT NULL, VERSION TEXT NOT NULL, KIND TEXT NOT NULL, DATA TEXT NOT NULL, UNIQUE(COLONY_NAME, NAME))`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createBlueprintsTable() error {
	sqlStatement := `CREATE TABLE IF NOT EXISTS ` + db.dbPrefix + `BLUEPRINTS (ID TEXT PRIMARY KEY NOT NULL, COLONY_NAME TEXT NOT NULL, NAME TEXT NOT NULL, KIND TEXT NOT NULL, DATA TEXT NOT NULL, UNIQUE(COLONY_NAME, NAME))`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createLocationsTable() error {
	sqlStatement := `CREATE TABLE IF NOT EXISTS ` + db.dbPrefix + `LOCATIONS (NAME TEXT PRIMARY KEY NOT NULL, LOCATION_ID TEXT NOT NULL, COLONY_NAME TEXT NOT NULL, DESCRIPTION TEXT, LONG REAL, LAT REAL)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createBlueprintHistoryTable() error {
	sqlStatement := `CREATE TABLE IF NOT EXISTS ` + db.dbPrefix + `BLUEPRINT_HISTORY (
		ID TEXT PRIMARY KEY NOT NULL,
		BLUEPRINT_ID TEXT NOT NULL,
		KIND TEXT NOT NULL,
		NAMESPACE TEXT NOT NULL,
		NAME TEXT NOT NULL,
		GENERATION BIGINT NOT NULL,
		SPEC TEXT NOT NULL,
		STATUS TEXT,
		TIMESTAMP TIMESTAMP NOT NULL,
		CHANGED_BY TEXT NOT NULL,
		CHANGE_TYPE TEXT NOT NULL
	)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	// Create index for faster queries by resource_id
	indexStatement := `CREATE INDEX IF NOT EXISTS ` + db.dbPrefix + `BLUEPRINT_HISTORY_INDEX1 ON ` + db.dbPrefix + `BLUEPRINT_HISTORY (BLUEPRINT_ID, TIMESTAMP DESC)`
	_, err = db.postgresql.Exec(indexStatement)
	return err
}

func (db *SQLiteDatabase) createProcessesIndex1() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `PROCESSES_INDEX1 ON ` + db.dbPrefix + `PROCESSES (TARGET_COLONY_NAME, STATE, SUBMISSION_TIME)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createProcessesIndex2() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `PROCESSES_INDEX2 ON ` + db.dbPrefix + `PROCESSES (TARGET_COLONY_NAME, STATE, START_TIME)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createProcessesIndex3() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `PROCESSES_INDEX3 ON ` + db.dbPrefix + `PROCESSES (TARGET_COLONY_NAME, STATE, END_TIME)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createProcessesIndex4() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `PROCESSES_INDEX4 ON ` + db.dbPrefix + `PROCESSES (IS_ASSIGNED, START_TIME, ASSIGNED_EXECUTOR_ID, STATE, PROCESS_ID)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createProcessesIndex6() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `PROCESSES_INDEX6 ON ` + db.dbPrefix + `PROCESSES (STATE, EXECUTOR_TYPE, IS_ASSIGNED, WAIT_FOR_PARENTS, TARGET_COLONY_NAME, LOCATION_NAME, PRIORITYTIME)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createProcessesIndex7() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `PROCESSES_INDEX7 ON ` + db.dbPrefix + `PROCESSES (TARGET_COLONY_NAME, STATE, PRIORITYTIME)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createProcessesIndex8() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `PROCESSES_INDEX8 ON ` + db.dbPrefix + `PROCESSES (TARGET_COLONY_NAME, STATE, EXECUTOR_TYPE, PRIORITYTIME)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createProcessesIndex9() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `PROCESSES_INDEX9 ON ` + db.dbPrefix + `PROCESSES (TARGET_COLONY_NAME, STATE, INITIATOR_NAME, PRIORITYTIME)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createProcessesIndex10() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `PROCESSES_INDEX10 ON ` + db.dbPrefix + `PROCESSES (TARGET_COLONY_NAME, STATE, LABEL, PRIORITYTIME)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createProcessesIndex11() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `PROCESSES_INDEX11 ON ` + db.dbPrefix + `PROCESSES (STATE, EXECUTOR_TYPE, IS_ASSIGNED, WAIT_FOR_PARENTS, TARGET_COLONY_NAME, LOCATION_NAME, TARGET_EXECUTOR_NAMES, CPU, MEMORY, GPUNAME, GPUMEM, GPUCOUNT, STORAGE, NODES, PROCESSES, PROCESSES_PER_NODE, PRIORITYTIME)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

// createProcessesAssignGIN creates a GIN index for efficient array membership searches
// on TARGET_EXECUTOR_NAMES. This optimizes the "$1 = ANY(TARGET_EXECUTOR_NAMES)" condition
// in SelectAndAssign. Partial index only includes waiting, unassigned processes.
func (db *SQLiteDatabase) createProcessesAssignGIN() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `PROCESSES_ASSIGN_GIN ON ` + db.dbPrefix + `PROCESSES (TARGET_EXECUTOR_NAMES) WHERE STATE = 0 AND IS_ASSIGNED = FALSE`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

// createProcessesAssignBTree creates a partial B-tree index optimized for the
// SelectAndAssign hot path. Only includes waiting, unassigned processes that
// are not waiting for parents.
func (db *SQLiteDatabase) createProcessesAssignBTree() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `PROCESSES_ASSIGN_BTREE ON ` + db.dbPrefix + `PROCESSES (TARGET_COLONY_NAME, EXECUTOR_TYPE, PRIORITYTIME) WHERE STATE = 0 AND IS_ASSIGNED = FALSE AND WAIT_FOR_PARENTS = FALSE`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createAttributesIndex1() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `ATTRIBUTES_INDEX1 ON ` + db.dbPrefix + `ATTRIBUTES (TARGET_ID, ATTRIBUTE_TYPE)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createAttributesIndex2() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `ATTRIBUTES_INDEX2 ON ` + db.dbPrefix + `ATTRIBUTES (TARGET_ID)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createRetentionIndex1() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `RETENTION_INDEX1 ON ` + db.dbPrefix + `ATTRIBUTES (ADDED, STATE)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createRetentionIndex2() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `RETENTION_INDEX2 ON ` + db.dbPrefix + `PROCESSES (SUBMISSION_TIME, STATE)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createRetentionIndex3() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `RETENTION_INDEX3 ON ` + db.dbPrefix + `PROCESSGRAPHS (SUBMISSION_TIME, STATE)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createRetentionIndex4() error {
	if !db.timescaleDB {
		sqlStatement := `CREATE INDEX ` + db.dbPrefix + `RETENTION_INDEX4 ON ` + db.dbPrefix + `FILES (ADDED)`
		_, err := db.postgresql.Exec(sqlStatement)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *SQLiteDatabase) createProcessesStateIndex() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `PROCESSES_STATE_INDEX ON ` + db.dbPrefix + `PROCESSES (STATE)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createProcessGraphsStateIndex() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `PROCESSGRAPHS_STATE_INDEX ON ` + db.dbPrefix + `PROCESSGRAPHS (STATE)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createFileIndex1() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `FILE_INDEX1 ON ` + db.dbPrefix + `FILES (COLONY_NAME, LABEL, NAME)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createFileIndex2() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `FILE_INDEX2 ON ` + db.dbPrefix + `FILES (COLONY_NAME, FILE_ID)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createFileIndex3() error {
	sqlStatement := `CREATE INDEX ` + db.dbPrefix + `FILE_INDEX3 ON ` + db.dbPrefix + `FILES (COLONY_NAME, LABEL)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createExecutorsIndex1() error {
	sqlStatement := `CREATE INDEX IF NOT EXISTS ` + db.dbPrefix + `EXECUTORS_INDEX1 ON ` + db.dbPrefix + `EXECUTORS (EXECUTOR_ID)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createUsersIndex1() error {
	sqlStatement := `CREATE INDEX IF NOT EXISTS ` + db.dbPrefix + `USERS_INDEX1 ON ` + db.dbPrefix + `USERS (COLONY_NAME, USER_ID)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createProcessesAssignedExecutorIndex() error {
	sqlStatement := `CREATE INDEX IF NOT EXISTS ` + db.dbPrefix + `PROCESSES_ASSIGNED_EXECUTOR_INDEX ON ` + db.dbPrefix + `PROCESSES (ASSIGNED_EXECUTOR_ID, STATE)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) createProcessGraphsIndex1() error {
	sqlStatement := `CREATE INDEX IF NOT EXISTS ` + db.dbPrefix + `PROCESSGRAPHS_INDEX1 ON ` + db.dbPrefix + `PROCESSGRAPHS (TARGET_COLONY_NAME, STATE, SUBMISSION_TIME)`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) Initialize() error {
	err := db.createUsersTable()
	if err != nil {
		return err
	}

	err = db.createServerTable()
	if err != nil {
		return err
	}

	err = db.createColoniesTable()
	if err != nil {
		return err
	}

	err = db.createExecutorsTable()
	if err != nil {
		return err
	}

	err = db.createNodesTable()
	if err != nil {
		return err
	}

	err = db.createFunctionsTable()
	if err != nil {
		return err
	}

	err = db.createProcessesTable()
	if err != nil {
		return err
	}

	err = db.createLogTable()
	if err != nil {
		return err
	}

	err = db.createFileTable()
	if err != nil {
		return err
	}

	err = db.createSnapshotTable()
	if err != nil {
		return err
	}

	err = db.createAttributesTable()
	if err != nil {
		return err
	}

	err = db.createProcessGraphsTable()
	if err != nil {
		return err
	}

	err = db.createGeneratorsTable()
	if err != nil {
		return err
	}

	err = db.createGeneratorArgsTable()
	if err != nil {
		return err
	}

	err = db.createCronsTable()
	if err != nil {
		return err
	}

	err = db.createBlueprintDefinitionsTable()
	if err != nil {
		return err
	}

	err = db.createBlueprintsTable()
	if err != nil {
		return err
	}

	err = db.createBlueprintHistoryTable()
	if err != nil {
		return err
	}

	err = db.createLocationsTable()
	if err != nil {
		return err
	}

	err = db.createProcessesIndex1()
	if err != nil {
		return err
	}

	err = db.createProcessesIndex2()
	if err != nil {
		return err
	}

	err = db.createProcessesIndex3()
	if err != nil {
		return err
	}

	err = db.createProcessesIndex4()
	if err != nil {
		return err
	}

	err = db.createProcessesIndex6()
	if err != nil {
		return err
	}

	err = db.createProcessesIndex7()
	if err != nil {
		return err
	}

	err = db.createProcessesIndex8()
	if err != nil {
		return err
	}

	err = db.createProcessesIndex9()
	if err != nil {
		return err
	}

	err = db.createProcessesIndex10()
	if err != nil {
		return err
	}

	err = db.createProcessesIndex11()
	if err != nil {
		return err
	}

	err = db.createProcessesAssignGIN()
	if err != nil {
		return err
	}

	err = db.createProcessesAssignBTree()
	if err != nil {
		return err
	}

	err = db.createAttributesIndex1()
	if err != nil {
		return err
	}

	err = db.createAttributesIndex2()
	if err != nil {
		return err
	}

	err = db.createRetentionIndex1()
	if err != nil {
		return err
	}

	err = db.createRetentionIndex2()
	if err != nil {
		return err
	}

	err = db.createRetentionIndex3()
	if err != nil {
		return err
	}

	err = db.createRetentionIndex4()
	if err != nil {
		return err
	}

	err = db.createProcessesStateIndex()
	if err != nil {
		return err
	}

	err = db.createProcessGraphsStateIndex()
	if err != nil {
		return err
	}

	err = db.createFileIndex1()
	if err != nil {
		return err
	}

	err = db.createFileIndex2()
	if err != nil {
		return err
	}

	err = db.createFileIndex3()
	if err != nil {
		return err
	}

	err = db.createExecutorsIndex1()
	if err != nil {
		return err
	}

	err = db.createUsersIndex1()
	if err != nil {
		return err
	}

	err = db.createProcessesAssignedExecutorIndex()
	if err != nil {
		return err
	}

	err = db.createProcessGraphsIndex1()
	if err != nil {
		return err
	}

	// TimescaleDB hypertables are a PostgreSQL-only feature; use the plain
	// logs index on SQLite
	sqlStatement := `CREATE INDEX IF NOT EXISTS ` + db.dbPrefix + `LOGS_INDEX1 ON ` + db.dbPrefix + `LOGS (PROCESS_ID)`
	_, err = db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}
