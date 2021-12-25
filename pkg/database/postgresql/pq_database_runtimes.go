package postgresql

import (
	"colonies/pkg/core"
	"database/sql"
	"errors"

	_ "github.com/lib/pq"
)

func (db *PQDatabase) AddRuntime(runtime *core.Runtime) error {
	sqlStatement := `INSERT INTO  ` + db.dbPrefix + `RUNTIMES (RUNTIME_ID, NAME, COLONY_ID, CPU, CORES, MEM, GPU, GPUS, STATUS) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := db.postgresql.Exec(sqlStatement, runtime.ID, runtime.Name, runtime.ColonyID, runtime.CPU, runtime.Cores, runtime.Mem, runtime.GPU, runtime.GPUs, 0)
	if err != nil {
		return err
	}

	return nil
}

func (db *PQDatabase) parseRuntimes(rows *sql.Rows) ([]*core.Runtime, error) {
	var runtimes []*core.Runtime

	for rows.Next() {
		var id string
		var name string
		var colonyID string
		var cpu string
		var cores int
		var mem int
		var gpu string
		var gpus int
		var status int
		if err := rows.Scan(&id, &name, &colonyID, &cpu, &cores, &mem, &gpu, &gpus, &status); err != nil {
			return nil, err
		}

		runtime := core.CreateRuntimeFromDB(id, name, colonyID, cpu, cores, mem, gpu, gpus, status)
		runtimes = append(runtimes, runtime)
	}

	return runtimes, nil
}

func (db *PQDatabase) GetRuntimes() ([]*core.Runtime, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `RUNTIMES`
	rows, err := db.postgresql.Query(sqlStatement)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	return db.parseRuntimes(rows)
}

func (db *PQDatabase) GetRuntimeByID(runtimeID string) (*core.Runtime, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `RUNTIMES WHERE RUNTIME_ID=$1`
	rows, err := db.postgresql.Query(sqlStatement, runtimeID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	runtimes, err := db.parseRuntimes(rows)
	if err != nil {
		return nil, err
	}

	if len(runtimes) > 1 {
		return nil, errors.New("Expected one runtime, runtime id should be unique")
	}

	if len(runtimes) == 0 {
		return nil, nil
	}

	return runtimes[0], nil
}

func (db *PQDatabase) GetRuntimesByColonyID(colonyID string) ([]*core.Runtime, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `RUNTIMES WHERE COLONY_ID=$1`
	rows, err := db.postgresql.Query(sqlStatement, colonyID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	runtimes, err := db.parseRuntimes(rows)
	if err != nil {
		return nil, err
	}

	return runtimes, nil
}

func (db *PQDatabase) ApproveRuntime(runtime *core.Runtime) error {
	sqlStatement := `UPDATE ` + db.dbPrefix + `RUNTIMES SET STATUS=1 WHERE RUNTIME_ID=$1`
	_, err := db.postgresql.Exec(sqlStatement, runtime.ID)
	if err != nil {
		return err
	}

	runtime.Approve()

	return nil
}

func (db *PQDatabase) RejectRuntime(runtime *core.Runtime) error {
	sqlStatement := `UPDATE ` + db.dbPrefix + `RUNTIMES SET STATUS=2 WHERE RUNTIME_ID=$1`
	_, err := db.postgresql.Exec(sqlStatement, runtime.ID)
	if err != nil {
		return err
	}

	runtime.Reject()

	return nil
}

func (db *PQDatabase) DeleteRuntimeByID(runtimeID string) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `RUNTIMES WHERE RUNTIME_ID=$1`
	_, err := db.postgresql.Exec(sqlStatement, runtimeID)
	if err != nil {
		return err
	}

	return nil
}

func (db *PQDatabase) DeleteRuntimesByColonyID(colonyID string) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `RUNTIMES WHERE COLONY_ID=$1`
	_, err := db.postgresql.Exec(sqlStatement, colonyID)
	if err != nil {
		return err
	}

	return nil
}