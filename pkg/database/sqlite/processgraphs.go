package sqlite

import (
	"database/sql"
	"time"

	"github.com/colonyos/colonies/pkg/core"
)

func (db *SQLiteDatabase) AddProcessGraph(processGraph *core.ProcessGraph) error {
	sqlStatement := `INSERT INTO  ` + db.dbPrefix + `PROCESSGRAPHS (PROCESSGRAPH_ID, TARGET_COLONY_NAME, ROOTS, STATE, SUBMISSION_TIME, START_TIME, END_TIME, INITIATOR_ID, INITIATOR_NAME) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := db.postgresql.Exec(sqlStatement, processGraph.ID, processGraph.ColonyName, stringArray(processGraph.Roots), processGraph.State, sqliteTS(time.Now()), sqliteTS(time.Time{}), sqliteTS(time.Time{}), processGraph.InitiatorID, processGraph.InitiatorName)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) parseProcessGraphs(rows *sql.Rows) ([]*core.ProcessGraph, error) {
	var graphs []*core.ProcessGraph

	for rows.Next() {
		var processGraphID string
		var colonyName string
		var roots []string
		var state int
		var submissionTime time.Time
		var startTime time.Time
		var endTime time.Time
		var initiatorID string
		var initiatorName string

		if err := rows.Scan(&processGraphID, &colonyName, (*stringArray)(&roots), &state, &submissionTime, &startTime, &endTime, &initiatorID, &initiatorName); err != nil {
			return nil, err
		}

		graph, err := core.CreateProcessGraph(colonyName)
		if err != nil {
			return graphs, err
		}

		graph.ID = processGraphID
		graph.ColonyName = colonyName
		graph.State = state
		graph.SubmissionTime = submissionTime
		graph.StartTime = startTime
		graph.EndTime = endTime
		graph.InitiatorID = initiatorID
		graph.InitiatorName = initiatorName

		for _, root := range roots {
			graph.AddRoot(root)
		}

		graphs = append(graphs, graph)
	}

	return graphs, nil
}

func (db *SQLiteDatabase) GetProcessGraphByID(processGraphID string) (*core.ProcessGraph, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `PROCESSGRAPHS WHERE PROCESSGRAPH_ID=$1`
	rows, err := db.postgresql.Query(sqlStatement, processGraphID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	processGraphs, err := db.parseProcessGraphs(rows)
	if err != nil {
		return nil, err
	}

	if len(processGraphs) == 0 {
		return nil, nil
	}

	return processGraphs[0], nil
}

func (db *SQLiteDatabase) SetProcessGraphState(processGraphID string, state int) error {
	graph, err := db.GetProcessGraphByID(processGraphID)
	if err != nil {
		return err
	}

	if graph.State == core.WAITING && state == core.RUNNING {
		sqlStatement := `UPDATE ` + db.dbPrefix + `PROCESSGRAPHS SET START_TIME=$1, STATE=$2 WHERE PROCESSGRAPH_ID=$3`
		_, err := db.postgresql.Exec(sqlStatement, sqliteTS(time.Now()), state, processGraphID)
		if err != nil {
			return err
		}
	} else if state == core.SUCCESS || state == core.FAILED {
		sqlStatement := `UPDATE ` + db.dbPrefix + `PROCESSGRAPHS SET END_TIME=$1, STATE=$2 WHERE PROCESSGRAPH_ID=$3`
		_, err := db.postgresql.Exec(sqlStatement, sqliteTS(time.Now()), state, processGraphID)
		if err != nil {
			return err
		}
	} else {
		sqlStatement := `UPDATE ` + db.dbPrefix + `PROCESSGRAPHS SET STATE=$1 WHERE PROCESSGRAPH_ID=$2`
		_, err := db.postgresql.Exec(sqlStatement, state, processGraphID)
		if err != nil {
			return err
		}

	}

	return nil
}

func (db *SQLiteDatabase) findProcessGraphsByState(colonyName string, state int, count int) ([]*core.ProcessGraph, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `PROCESSGRAPHS WHERE TARGET_COLONY_NAME=$1 AND STATE=$2 ORDER BY SUBMISSION_TIME DESC LIMIT $3`
	rows, err := db.postgresql.Query(sqlStatement, colonyName, state, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches, err := db.parseProcessGraphs(rows)
	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (db *SQLiteDatabase) FindWaitingProcessGraphs(colonyName string, count int) ([]*core.ProcessGraph, error) {
	return db.findProcessGraphsByState(colonyName, core.WAITING, count)
}

func (db *SQLiteDatabase) FindRunningProcessGraphs(colonyName string, count int) ([]*core.ProcessGraph, error) {
	return db.findProcessGraphsByState(colonyName, core.RUNNING, count)
}

func (db *SQLiteDatabase) FindSuccessfulProcessGraphs(colonyName string, count int) ([]*core.ProcessGraph, error) {
	return db.findProcessGraphsByState(colonyName, core.SUCCESS, count)
}

func (db *SQLiteDatabase) FindFailedProcessGraphs(colonyName string, count int) ([]*core.ProcessGraph, error) {
	return db.findProcessGraphsByState(colonyName, core.FAILED, count)
}

func (db *SQLiteDatabase) countProcessGraphsByColonyName(state int, colonyName string) (int, error) {
	sqlStatement := `SELECT COUNT(*) FROM ` + db.dbPrefix + `PROCESSGRAPHS WHERE STATE=$1 AND TARGET_COLONY_NAME=$2`
	rows, err := db.postgresql.Query(sqlStatement, state, colonyName)
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
func (db *SQLiteDatabase) CountWaitingProcessGraphsByColonyName(colonyName string) (int, error) {
	return db.countProcessGraphsByColonyName(core.WAITING, colonyName)
}

func (db *SQLiteDatabase) CountRunningProcessGraphsByColonyName(colonyName string) (int, error) {
	return db.countProcessGraphsByColonyName(core.RUNNING, colonyName)
}

func (db *SQLiteDatabase) CountSuccessfulProcessGraphsByColonyName(colonyName string) (int, error) {
	return db.countProcessGraphsByColonyName(core.SUCCESS, colonyName)
}

func (db *SQLiteDatabase) CountFailedProcessGraphsByColonyName(colonyName string) (int, error) {
	return db.countProcessGraphsByColonyName(core.FAILED, colonyName)
}

func (db *SQLiteDatabase) countProcessGraphs(state int) (int, error) {
	sqlStatement := `SELECT COUNT(*) FROM ` + db.dbPrefix + `PROCESSGRAPHS WHERE STATE=$1`
	rows, err := db.postgresql.Query(sqlStatement, state)
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

func (db *SQLiteDatabase) RemoveAllProcessGraphsByColonyName(colonyName string) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `PROCESSGRAPHS WHERE TARGET_COLONY_NAME=$1`
	_, err := db.postgresql.Exec(sqlStatement, colonyName)
	if err != nil {
		return err
	}

	return db.RemoveAllProcessesInProcessGraphsByColonyName(colonyName)
}

// XXX: This function may remove all belonging processes if the graph is running.
func (db *SQLiteDatabase) RemoveAllWaitingProcessGraphsByColonyName(colonyName string) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `PROCESSGRAPHS WHERE TARGET_COLONY_NAME=$1 AND STATE=$2`
	_, err := db.postgresql.Exec(sqlStatement, colonyName, core.WAITING)
	if err != nil {
		return err
	}

	err = db.RemoveAllProcessesInProcessGraphsByColonyNameWithState(colonyName, core.WAITING)
	if err != nil {
		return err
	}

	return nil
}

// XXX: This function can cause inconsisteny, for example if the processgraph is running, and all running processes
// is removed it will no longer be possible to resolve the processgraph
func (db *SQLiteDatabase) RemoveAllRunningProcessGraphsByColonyName(colonyName string) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `PROCESSGRAPHS WHERE TARGET_COLONY_NAME=$1 AND STATE=$2`
	_, err := db.postgresql.Exec(sqlStatement, colonyName, core.RUNNING)
	if err != nil {
		return err
	}

	err = db.RemoveAllProcessesInProcessGraphsByColonyNameWithState(colonyName, core.RUNNING)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) RemoveAllSuccessfulProcessGraphsByColonyName(colonyName string) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `PROCESSGRAPHS WHERE TARGET_COLONY_NAME=$1 AND STATE=$2`
	_, err := db.postgresql.Exec(sqlStatement, colonyName, core.SUCCESS)
	if err != nil {
		return err
	}

	err = db.RemoveAllProcessesInProcessGraphsByColonyNameWithState(colonyName, core.SUCCESS)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) RemoveAllFailedProcessGraphsByColonyName(colonyName string) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `PROCESSGRAPHS WHERE TARGET_COLONY_NAME=$1 AND STATE=$2`
	_, err := db.postgresql.Exec(sqlStatement, colonyName, core.FAILED)
	if err != nil {
		return err
	}

	err = db.RemoveAllProcessesInProcessGraphsByColonyNameWithState(colonyName, core.FAILED)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) RemoveProcessGraphByID(processGraphID string) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `PROCESSGRAPHS WHERE PROCESSGRAPH_ID=$1`
	_, err := db.postgresql.Exec(sqlStatement, processGraphID)
	if err != nil {
		return err
	}

	return db.RemoveAllProcessesByProcessGraphID(processGraphID)
}

func (db *SQLiteDatabase) CountWaitingProcessGraphs() (int, error) {
	return db.countProcessGraphs(core.WAITING)
}

func (db *SQLiteDatabase) CountRunningProcessGraphs() (int, error) {
	return db.countProcessGraphs(core.RUNNING)
}

func (db *SQLiteDatabase) CountSuccessfulProcessGraphs() (int, error) {
	return db.countProcessGraphs(core.SUCCESS)
}

func (db *SQLiteDatabase) CountFailedProcessGraphs() (int, error) {
	return db.countProcessGraphs(core.FAILED)
}
