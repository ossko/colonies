package rpc

import (
	"encoding/json"
)

const GetFunctionsPayloadType = "getfunctionsmsg"

type GetFunctionsMsg struct {
	ExecutorName string `json:"executorname"`
	ColonyName   string `json:"colonyname"`
	MsgType      string `json:"msgtype"`
}

func CreateGetFunctionsMsg(colonyName string, executorName string) *GetFunctionsMsg {
	msg := &GetFunctionsMsg{}
	msg.ColonyName = colonyName
	msg.ExecutorName = executorName
	msg.MsgType = GetFunctionsPayloadType

	return msg
}

func CreateGetFunctionsByColonyNameMsg(colonyName string) *GetFunctionsMsg {
	msg := &GetFunctionsMsg{}
	msg.ColonyName = colonyName
	msg.MsgType = GetFunctionsPayloadType

	return msg
}

func (msg *GetFunctionsMsg) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (msg *GetFunctionsMsg) ToJSONIndent() (string, error) {
	jsonBytes, err := json.MarshalIndent(msg, "", "    ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (msg *GetFunctionsMsg) Equals(msg2 *GetFunctionsMsg) bool {
	if msg2 == nil {
		return false
	}

	if msg.MsgType == msg2.MsgType && (msg.ExecutorName == msg2.ExecutorName || msg.ColonyName == msg.ColonyName) {
		return true
	}

	return false
}

func CreateGetFunctionsMsgFromJSON(jsonString string) (*GetFunctionsMsg, error) {
	var msg *GetFunctionsMsg

	err := json.Unmarshal([]byte(jsonString), &msg)
	if err != nil {
		return msg, err
	}

	return msg, nil
}
