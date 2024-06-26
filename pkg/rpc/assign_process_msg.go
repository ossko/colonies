package rpc

import (
	"encoding/json"
)

const AssignProcessPayloadType = "assignprocessmsg"

type AssignProcessMsg struct {
	ColonyName      string `json:"colonyname"`
	Timeout         int    `json:"timeout"`
	AvailableCPU    string `json:"availablecpu"`
	AvailableMemory string `json:"availablemem"`
	MsgType         string `json:"msgtype"`
}

func CreateAssignProcessMsg(colonyName string, availableCPU string, availableMemory string) *AssignProcessMsg {
	msg := &AssignProcessMsg{}
	msg.ColonyName = colonyName
	msg.AvailableCPU = availableCPU
	msg.AvailableMemory = availableMemory
	msg.MsgType = AssignProcessPayloadType
	msg.Timeout = -1 // Not implemented yet

	return msg
}

func (msg *AssignProcessMsg) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (msg *AssignProcessMsg) ToJSONIndent() (string, error) {
	jsonBytes, err := json.MarshalIndent(msg, "", "    ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (msg *AssignProcessMsg) Equals(msg2 *AssignProcessMsg) bool {
	if msg2 == nil {
		return false
	}

	if msg.MsgType == msg2.MsgType &&
		msg.ColonyName == msg2.ColonyName &&
		msg.AvailableCPU == msg2.AvailableCPU &&
		msg.AvailableMemory == msg2.AvailableMemory {
		return true
	}

	return false
}

func CreateAssignProcessMsgFromJSON(jsonString string) (*AssignProcessMsg, error) {
	var msg *AssignProcessMsg

	err := json.Unmarshal([]byte(jsonString), &msg)
	if err != nil {
		return msg, err
	}

	return msg, nil
}
