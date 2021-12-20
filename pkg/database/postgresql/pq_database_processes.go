package postgresql

import (
	"colonies/pkg/core"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

func (db *PQDatabase) AddProcess(process *core.Process) error {
	targetComputerIDs := process.TargetComputerIDs()
	if len(process.TargetComputerIDs()) == 0 {
		targetComputerIDs = []string{"*"}
	}

	submissionTime := time.Now()

	sqlStatement := `INSERT INTO  ` + db.dbPrefix + `PROCESSES (PROCESS_ID, TARGET_COLONY_ID, TARGET_COMPUTER_IDS, ASSIGNED_COMPUTER_ID, STATUS, IS_ASSIGNED, COMPUTER_TYPE, SUBMISSION_TIME, START_TIME, END_TIME, DEADLINE, RETRIES, TIMEOUT, MAX_RETRIES, LOG, MEM, CORES, GPUs) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`
	_, err := db.postgresql.Exec(sqlStatement, process.ID(), process.TargetColonyID(), pq.Array(targetComputerIDs), process.AssignedComputerID(), process.Status(), process.Assigned(), process.ComputerType(), submissionTime, time.Time{}, time.Time{}, process.Deadline(), 0, process.Timeout(), process.MaxRetries(), "", process.Mem(), process.Cores(), process.GPUs())
	if err != nil {
		return err
	}

	err = db.AddAttributes(process.InAttributes())
	if err != nil {
		// XXX: Should we also remove the process we just added?
		return err
	}
	err = db.AddAttributes(process.ErrAttributes())
	if err != nil {
		// XXX: Should we also remove the process we just added?
		return err
	}
	err = db.AddAttributes(process.OutAttributes())
	if err != nil {
		// XXX: Should we also remove the process we just added?
		return err
	}

	process.SetSubmissionTime(submissionTime)

	return nil
}

func (db *PQDatabase) parseProcesses(rows *sql.Rows) ([]*core.Process, error) {
	var processes []*core.Process

	for rows.Next() {
		var processID string
		var targetColonyID string
		var targetComputerIDs []string
		var assignedComputerID string
		var status int
		var isAssigned bool
		var computerType string
		var submissionTime time.Time
		var startTime time.Time
		var endTime time.Time
		var deadline time.Time
		var timeout int
		var retries int
		var maxRetries int
		var log string
		var mem int
		var cores int
		var gpus int

		if err := rows.Scan(&processID, &targetColonyID, pq.Array(&targetComputerIDs), &assignedComputerID, &status, &isAssigned, &computerType, &submissionTime, &startTime, &endTime, &deadline, &timeout, &retries, &maxRetries, &log, &mem, &cores, &gpus); err != nil {
			return nil, err
		}

		inAttributes, err := db.GetAttributes(processID, core.IN)
		if err != nil {
			return nil, err
		}

		errAttributes, err := db.GetAttributes(processID, core.ERR)
		if err != nil {
			return nil, err
		}

		outAttributes, err := db.GetAttributes(processID, core.OUT)
		if err != nil {
			return nil, err
		}

		process := core.CreateProcessFromDB(processID, targetColonyID, targetComputerIDs, assignedComputerID, status, isAssigned, computerType, submissionTime, startTime, endTime, deadline, timeout, retries, maxRetries, log, mem, cores, gpus, inAttributes, errAttributes, outAttributes)
		processes = append(processes, process)
	}

	return processes, nil
}

func (db *PQDatabase) GetProcesses() ([]*core.Process, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `PROCESSES`
	rows, err := db.postgresql.Query(sqlStatement)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	return db.parseProcesses(rows)
}

func (db *PQDatabase) GetProcessByID(processID string) (*core.Process, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `PROCESSES WHERE PROCESS_ID=$1`
	rows, err := db.postgresql.Query(sqlStatement, processID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	processes, err := db.parseProcesses(rows)
	if err != nil {
		return nil, err
	}

	if len(processes) > 1 {
		return nil, errors.New("Expected one process, process id should be unique")
	}

	if len(processes) == 0 {
		return nil, nil
	}

	return processes[0], nil
}

func (db *PQDatabase) selectCandidate(candidates []*core.Process) *core.Process {
	if len(candidates) > 0 {
		return candidates[0]
	} else {
		return nil
	}
}

func (db *PQDatabase) FindWaitingProcesses(colonyID string, count int) ([]*core.Process, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `PROCESSES WHERE TARGET_COLONY_ID=$1 AND STATUS=$2 ORDER BY SUBMISSION_TIME LIMIT $3`
	rows, err := db.postgresql.Query(sqlStatement, colonyID, core.WAITING, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches, err := db.parseProcesses(rows)
	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (db *PQDatabase) FindRunningProcesses(colonyID string, count int) ([]*core.Process, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `PROCESSES WHERE TARGET_COLONY_ID=$1 AND STATUS=$2 ORDER BY START_TIME LIMIT $3`
	rows, err := db.postgresql.Query(sqlStatement, colonyID, core.RUNNING, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches, err := db.parseProcesses(rows)
	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (db *PQDatabase) FindSuccessfulProcesses(colonyID string, count int) ([]*core.Process, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `PROCESSES WHERE TARGET_COLONY_ID=$1 AND STATUS=$2 ORDER BY END_TIME LIMIT $3`
	rows, err := db.postgresql.Query(sqlStatement, colonyID, core.SUCCESS, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches, err := db.parseProcesses(rows)
	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (db *PQDatabase) FindFailedProcesses(colonyID string, count int) ([]*core.Process, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `PROCESSES WHERE TARGET_COLONY_ID=$1 AND STATUS=$2 ORDER BY END_TIME LIMIT $3`
	rows, err := db.postgresql.Query(sqlStatement, colonyID, core.FAILED, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches, err := db.parseProcesses(rows)
	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (db *PQDatabase) FindUnassignedProcesses(colonyID string, computerID string, count int) ([]*core.Process, error) {
	// Note: The @> function tests if an array is a subset of another array
	// We need to do that since the TARGET_COMPUTER_IDS can contains many IDs
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `PROCESSES WHERE IS_ASSIGNED=FALSE AND TARGET_COLONY_ID=$1 AND (TARGET_COMPUTER_IDS@>$2 OR TARGET_COMPUTER_IDS@>$3) ORDER BY SUBMISSION_TIME LIMIT $4`
	rows, err := db.postgresql.Query(sqlStatement, colonyID, pq.Array([]string{computerID}), pq.Array([]string{"*"}), count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches, err := db.parseProcesses(rows)
	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (db *PQDatabase) DeleteProcessByID(processID string) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `PROCESSES WHERE PROCESS_ID=$1`
	_, err := db.postgresql.Exec(sqlStatement, processID)
	if err != nil {
		return err
	}

	// TODO test this code
	err = db.DeleteAllAttributesByProcessID(processID)
	if err != nil {
		return err
	}

	return nil
}

func (db *PQDatabase) DeleteAllProcesses() error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `PROCESSES`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	err = db.DeleteAllAttributes()
	if err != nil {
		return err
	}

	return nil
}

func (db *PQDatabase) ResetProcess(process *core.Process) error {
	sqlStatement := `UPDATE ` + db.dbPrefix + `PROCESSES SET IS_ASSIGNED=FALSE, START_TIME=$1, END_TIME=$2, ASSIGNED_COMPUTER_ID=$3, STATUS=$4 WHERE process_ID=$5`
	_, err := db.postgresql.Exec(sqlStatement, time.Time{}, time.Time{}, "", core.WAITING, process.ID())
	if err != nil {
		return err
	}

	process.SetStartTime(time.Time{})
	process.SetEndTime(time.Time{})
	process.SetAssignedComputerID("")
	process.SetStatus(core.WAITING)

	return nil
}

func (db *PQDatabase) ResetAllProcesses(process *core.Process) error {
	sqlStatement := `UPDATE ` + db.dbPrefix + `PROCESSES SET IS_ASSIGNED=FALSE, START_TIME=$1, END_TIME=$2, ASSIGNED_COMPUTER_ID=$3, STATUS=$4`
	_, err := db.postgresql.Exec(sqlStatement, time.Time{}, time.Time{}, "", core.WAITING)
	if err != nil {
		return err
	}

	return nil
}

func (db *PQDatabase) AssignComputer(computerID string, process *core.Process) error {
	startTime := time.Now()

	sqlStatement := `UPDATE ` + db.dbPrefix + `PROCESSES SET IS_ASSIGNED=TRUE, START_TIME=$1, ASSIGNED_COMPUTER_ID=$2, STATUS=$3 WHERE PROCESS_ID=$4`
	_, err := db.postgresql.Exec(sqlStatement, startTime, computerID, core.RUNNING, process.ID())
	if err != nil {
		return err
	}

	process.SetStartTime(startTime)
	process.Assign()
	process.SetAssignedComputerID(computerID)
	process.SetStatus(core.RUNNING)

	return nil
}

func (db *PQDatabase) UnassignComputer(process *core.Process) error {
	endTime := time.Now()

	sqlStatement := `UPDATE ` + db.dbPrefix + `PROCESSES SET IS_ASSIGNED=FALSE, END_TIME=$1, STATUS=$2 WHERE PROCESS_ID=$3`
	_, err := db.postgresql.Exec(sqlStatement, endTime, core.FAILED, process.ID())
	if err != nil {
		return err
	}

	process.SetEndTime(endTime)
	process.Unassign()
	process.SetStatus(core.FAILED)

	return nil
}

func (db *PQDatabase) MarkSuccessful(process *core.Process) error {
	if process.Status() == core.FAILED {
		return errors.New("Tried to set failed process as completed")
	}

	if process.Status() == core.WAITING {
		return errors.New("Tried to set waiting process as completed without being running")
	}

	processFromDB, err := db.GetProcessByID(process.ID())
	if err != nil {
		return err
	}

	if processFromDB.Status() == core.FAILED {
		return errors.New("Tried to set failed process (from db) as successful")
	}

	if processFromDB.Status() == core.WAITING {
		return errors.New("Tried to set waiting process (from db) as successful without being running")
	}

	endTime := time.Now()

	sqlStatement := `UPDATE ` + db.dbPrefix + `PROCESSES SET END_TIME=$1, STATUS=$2 WHERE PROCESS_ID=$3`
	_, err = db.postgresql.Exec(sqlStatement, endTime, core.SUCCESS, process.ID())
	if err != nil {
		return err
	}

	process.SetEndTime(endTime)
	process.SetStatus(core.SUCCESS)

	return nil
}

func (db *PQDatabase) MarkFailed(process *core.Process) error {
	endTime := time.Now()

	if process.Status() == core.SUCCESS {
		return errors.New("Tried to set successful process as failed")
	}

	if process.Status() == core.WAITING {
		return errors.New("Tried to set waiting process as failed without being running")
	}

	processFromDB, err := db.GetProcessByID(process.ID())
	if err != nil {
		return err
	}

	if processFromDB.Status() == core.SUCCESS {
		return errors.New("Tried to set successful (from db) as failed")
	}

	if processFromDB.Status() == core.WAITING {
		return errors.New("Tried to set successful process (from db) as failed without being running")
	}

	sqlStatement := `UPDATE ` + db.dbPrefix + `PROCESSES SET END_TIME=$1, STATUS=$2 WHERE PROCESS_ID=$3`
	_, err = db.postgresql.Exec(sqlStatement, endTime, core.FAILED, process.ID())
	if err != nil {
		return err
	}

	process.SetEndTime(endTime)
	process.SetStatus(core.SUCCESS)

	return nil
}

func (db *PQDatabase) NumberOfProcesses() (int, error) {
	sqlStatement := `SELECT COUNT(*) FROM ` + db.dbPrefix + `PROCESSES`
	rows, err := db.postgresql.Query(sqlStatement)
	if err != nil {
		return -1, err
	}

	defer rows.Close()

	rows.Next()
	var count int
	err = rows.Scan(&count)
	if err != nil {
		return -1, err
	}

	return count, nil
}

func (db *PQDatabase) countProcesses(status int) (int, error) {
	sqlStatement := `SELECT COUNT(*) FROM ` + db.dbPrefix + `PROCESSES WHERE STATUS=$1`
	rows, err := db.postgresql.Query(sqlStatement, status)
	if err != nil {
		return -1, err
	}

	defer rows.Close()

	rows.Next()
	var count int
	err = rows.Scan(&count)
	if err != nil {
		return -1, err
	}

	return count, nil
}

func (db *PQDatabase) NumberOfWaitingProcesses() (int, error) {
	return db.countProcesses(core.WAITING)
}

func (db *PQDatabase) NumberOfRunningProcesses() (int, error) {
	return db.countProcesses(core.RUNNING)
}

func (db *PQDatabase) NumberOfSuccessfulProcesses() (int, error) {
	return db.countProcesses(core.SUCCESS)
}

func (db *PQDatabase) NumberOfFailedProcesses() (int, error) {
	return db.countProcesses(core.FAILED)
}