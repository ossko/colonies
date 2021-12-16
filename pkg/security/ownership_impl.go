package security

import (
	"colonies/pkg/database"
	"errors"
)

type OwnershipImpl struct {
	db database.Database
}

func CreateOwnership(db database.Database) *OwnershipImpl {
	ownership := &OwnershipImpl{}
	ownership.db = db
	return ownership
}

func (ownership *OwnershipImpl) CheckIfColonyExists(colonyID string) error {
	colony, err := ownership.db.GetColonyByID(colonyID)
	if err != nil {
		return err
	}

	if colony == nil {
		return errors.New("Colony <" + colonyID + "> does not exists")
	}

	return nil
}

func (ownership *OwnershipImpl) CheckIfColonyHasWorker(colonyID string, workerID string) error {
	return nil
}
