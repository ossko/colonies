package rpc

import (
	"encoding/json"
)

const GetProcessesPayloadType = "getprocessesmsg"

type GetProcessesMsg struct {
	ColonyName   string `json:"colonyname"`
	Count        int    `json:"count"`
	State        int    `json:"state"`
	ExecutorType string `json:"executortype"`
	Label        string `json:"label"`
	Initiator    string `json:"initiator"`
	MsgType      string `json:"msgtype"`
}

func CreateGetProcessesMsg(colonyName string, count int, state int, executorType string, label string, initiator string) *GetProcessesMsg {
	msg := &GetProcessesMsg{}
	msg.ColonyName = colonyName
	msg.Count = count
	msg.State = state
	msg.ExecutorType = executorType
	msg.Label = label
	msg.Initiator = initiator
	msg.MsgType = GetProcessesPayloadType

	return msg
}

func (msg *GetProcessesMsg) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (msg *GetProcessesMsg) ToJSONIndent() (string, error) {
	jsonBytes, err := json.MarshalIndent(msg, "", "    ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (msg *GetProcessesMsg) Equals(msg2 *GetProcessesMsg) bool {
	if msg2 == nil {
		return false
	}

	if msg.MsgType == msg2.MsgType &&
		msg.ColonyName == msg2.ColonyName &&
		msg.Count == msg2.Count &&
		msg.State == msg2.State &&
		msg.ExecutorType == msg2.ExecutorType &&
		msg.Label == msg2.Label &&
		msg.Initiator == msg2.Initiator {
		return true
	}

	return false
}

func CreateGetProcessesMsgFromJSON(jsonString string) (*GetProcessesMsg, error) {
	var msg *GetProcessesMsg

	err := json.Unmarshal([]byte(jsonString), &msg)
	if err != nil {
		return msg, err
	}

	return msg, nil
}
