package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/colonyos/colonies/pkg/core"
	"github.com/colonyos/colonies/pkg/database/dberr"
)

func (db *SQLiteDatabase) AddAttributes(attributes []core.Attribute) error {
	if len(attributes) == 0 {
		return nil
	}

	// Use batch insert for better performance
	if len(attributes) == 1 {
		return db.AddAttribute(attributes[0])
	}

	// Build batch INSERT statement
	sqlStatement := `INSERT INTO ` + db.dbPrefix + `ATTRIBUTES (ATTRIBUTE_ID, KEY, VALUE, ATTRIBUTE_TYPE, TARGET_ID, TARGET_COLONY_NAME, PROCESSGRAPH_ID, ADDED, STATE) VALUES `

	vals := []interface{}{}
	now := time.Now()
	for i, attr := range attributes {
		if i > 0 {
			sqlStatement += ", "
		}
		base := i * 9
		sqlStatement += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)", base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+9)
		vals = append(vals, attr.ID, attr.Key, attr.Value, attr.AttributeType, attr.TargetID, attr.TargetColonyName, attr.TargetProcessGraphID, sqliteTS(now), attr.State)
	}

	_, err := db.postgresql.Exec(sqlStatement, vals...)
	return err
}

func (db *SQLiteDatabase) AddAttribute(attribute core.Attribute) error {
	sqlStatement := `INSERT INTO  ` + db.dbPrefix + `ATTRIBUTES (ATTRIBUTE_ID, KEY, VALUE, ATTRIBUTE_TYPE, TARGET_ID, TARGET_COLONY_NAME, PROCESSGRAPH_ID, ADDED, STATE) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := db.postgresql.Exec(sqlStatement, attribute.ID, attribute.Key, attribute.Value, attribute.AttributeType, attribute.TargetID, attribute.TargetColonyName, attribute.TargetProcessGraphID, sqliteTS(time.Now()), attribute.State)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) parseAttributes(rows *sql.Rows) ([]core.Attribute, error) {
	var attributes []core.Attribute

	for rows.Next() {
		var attributeID string
		var key string
		var value string
		var attributeType int
		var targetID string
		var targetColonyName string
		var targetProcessGraphID string
		var added time.Time
		var state int
		if err := rows.Scan(&attributeID, &key, &value, &attributeType, &targetID, &targetColonyName, &targetProcessGraphID, &added, &state); err != nil {
			return nil, err
		}

		attribute := core.CreateAttribute(targetID, targetColonyName, targetProcessGraphID, attributeType, key, value)
		attribute.State = state
		attributes = append(attributes, attribute)
	}

	return attributes, nil
}

func (db *SQLiteDatabase) GetAttributeByID(attributeID string) (core.Attribute, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `ATTRIBUTES WHERE ATTRIBUTE_ID=$1`
	rows, err := db.postgresql.Query(sqlStatement, attributeID)
	if err != nil {
		return core.Attribute{}, err
	}

	defer rows.Close()

	attributes, err := db.parseAttributes(rows)
	if err != nil {
		return core.Attribute{}, err
	}

	if len(attributes) > 1 {
		return core.Attribute{}, dberr.Conflict("Expected attributes to be unique")
	} else if len(attributes) == 0 {
		return core.Attribute{}, dberr.NotFound("Attribute does not exists")
	}

	return attributes[0], nil
}

func (db *SQLiteDatabase) GetAttributesByColonyName(colonyName string) ([]core.Attribute, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `ATTRIBUTES WHERE TARGET_COLONY_NAME=$1`
	rows, err := db.postgresql.Query(sqlStatement, colonyName)
	if err != nil {
		return []core.Attribute{}, err
	}

	defer rows.Close()

	attributes, err := db.parseAttributes(rows)
	if err != nil {
		return []core.Attribute{}, err
	}

	return attributes, nil
}

func (db *SQLiteDatabase) GetAttribute(targetID string, key string, attributeType int) (core.Attribute, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `ATTRIBUTES WHERE TARGET_ID=$1 AND KEY=$2 AND ATTRIBUTE_TYPE=$3`
	rows, err := db.postgresql.Query(sqlStatement, targetID, key, attributeType)
	if err != nil {
		return core.Attribute{}, err
	}

	defer rows.Close()

	attributes, err := db.parseAttributes(rows)
	if err != nil {
		return core.Attribute{}, err
	}
	if len(attributes) > 1 {
		return core.Attribute{}, dberr.Conflict("Expected attributes to be unique")
	} else if len(attributes) == 0 {
		return core.Attribute{}, dberr.NotFound("Attribute does not exists")
	}

	return attributes[0], nil
}

func (db *SQLiteDatabase) GetAttributes(targetID string) ([]core.Attribute, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `ATTRIBUTES WHERE TARGET_ID=$1`
	rows, err := db.postgresql.Query(sqlStatement, targetID)
	if err != nil {
		return []core.Attribute{}, err
	}

	defer rows.Close()

	return db.parseAttributes(rows)
}

func (db *SQLiteDatabase) GetAttributesByType(targetID string, attributeType int) ([]core.Attribute, error) {
	sqlStatement := `SELECT * FROM ` + db.dbPrefix + `ATTRIBUTES WHERE TARGET_ID=$1 AND ATTRIBUTE_TYPE=$2`
	rows, err := db.postgresql.Query(sqlStatement, targetID, attributeType)
	if err != nil {
		return []core.Attribute{}, err
	}

	defer rows.Close()

	return db.parseAttributes(rows)
}

func (db *SQLiteDatabase) UpdateAttribute(attribute core.Attribute) error {
	_, err := db.GetAttributeByID(attribute.ID)
	if err != nil {
		return err
	}

	sqlStatement := `UPDATE ` + db.dbPrefix + `ATTRIBUTES SET ATTRIBUTE_ID=$1, VALUE=$2`
	_, err = db.postgresql.Exec(sqlStatement, attribute.ID, attribute.Value)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) SetAttributeState(processID string, state int) error {
	sqlStatement := `UPDATE ` + db.dbPrefix + `ATTRIBUTES SET STATE=$1 WHERE TARGET_ID=$2`
	_, err := db.postgresql.Exec(sqlStatement, state, processID)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) RemoveAttributeByID(attributeID string) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `ATTRIBUTES WHERE ATTRIBUTE_ID=$1`
	_, err := db.postgresql.Exec(sqlStatement, attributeID)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) RemoveAllAttributesByColonyName(colonyName string) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `ATTRIBUTES WHERE TARGET_COLONY_NAME=$1`
	_, err := db.postgresql.Exec(sqlStatement, colonyName)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) RemoveAllAttributesByColonyNameWithState(colonyName string, state int) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `ATTRIBUTES WHERE TARGET_COLONY_NAME=$1 AND STATE=$2 AND PROCESSGRAPH_ID=$3`
	_, err := db.postgresql.Exec(sqlStatement, colonyName, state, "")
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) RemoveAllAttributesByProcessGraphID(processGraphID string) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `ATTRIBUTES WHERE PROCESSGRAPH_ID=$1`
	_, err := db.postgresql.Exec(sqlStatement, processGraphID)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) RemoveAllAttributesInProcessGraphsByColonyName(colonyName string) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `ATTRIBUTES WHERE PROCESSGRAPH_ID!=$1 AND TARGET_COLONY_NAME=$2`
	_, err := db.postgresql.Exec(sqlStatement, "", colonyName)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) RemoveAllAttributesInProcessGraphsByColonyNameWithState(colonyName string, state int) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `ATTRIBUTES WHERE TARGET_COLONY_NAME=$1 AND STATE=$2 AND PROCESSGRAPH_ID!=$3`
	_, err := db.postgresql.Exec(sqlStatement, colonyName, state, "")
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) RemoveAttributesByTargetID(targetID string, attributeType int) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `ATTRIBUTES WHERE TARGET_ID=$1 AND ATTRIBUTE_TYPE=$2`
	_, err := db.postgresql.Exec(sqlStatement, targetID, attributeType)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) RemoveAllAttributesByTargetID(targetID string) error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `ATTRIBUTES WHERE TARGET_ID=$1`
	_, err := db.postgresql.Exec(sqlStatement, targetID)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDatabase) RemoveAllAttributes() error {
	sqlStatement := `DELETE FROM ` + db.dbPrefix + `ATTRIBUTES`
	_, err := db.postgresql.Exec(sqlStatement)
	if err != nil {
		return err
	}

	return nil
}
