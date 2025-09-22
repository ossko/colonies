package server

import (
	"errors"

	"github.com/colonyos/colonies/pkg/database"
)

type InitiatorType int

const (
	Executor InitiatorType = iota
	User
)

type Initiator struct {
	Name string
	Type InitiatorType
}

func resolveInitiator(
	colonyName string,
	recoveredID string,
	db database.Database) (string, error) {

	executor, err := db.GetExecutorByID(recoveredID)
	if err != nil {
		return "", err
	}

	if executor != nil {
		return executor.Name, nil
	} else {
		user, err := db.GetUserByID(colonyName, recoveredID)
		if err != nil {
			return "", err
		}
		if user != nil {
			return user.Name, nil
		} else {
			return "", errors.New("Could not derive InitiatorName")
		}
	}
}

func deriveInitiator(
	colonyName string,
	recoveredID string,
	db database.Database) (*Initiator, error) {

	executor, err := db.GetExecutorByID(recoveredID)
	if err != nil {
		return nil, err
	}

	if executor != nil {
		return &Initiator{Name: executor.Name, Type: Executor}, nil
	} else {
		user, err := db.GetUserByID(colonyName, recoveredID)
		if err != nil {
			return nil, err
		}
		if user != nil {
			return &Initiator{Name: user.Name, Type: User}, nil
		} else {
			return nil, errors.New("could not derive initiator")
		}
	}
}
