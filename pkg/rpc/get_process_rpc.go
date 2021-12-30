package rpc

import (
	"encoding/json"
)

const GetProcessMsgType = "GetProcess"

type GetProcessMsg struct {
	RPC       RPC    `json:"rpc"`
	ProcessID string `json:"processid"`
}

func CreateGetProcessMsg(processID string) *GetProcessMsg {
	msg := &GetProcessMsg{}
	msg.RPC.Method = GetProcessMsgType
	msg.ProcessID = processID

	return msg
}

func (msg *GetProcessMsg) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func CreateGetProcessMsgFromJSON(jsonString string) (*GetProcessMsg, error) {
	var msg *GetProcessMsg

	err := json.Unmarshal([]byte(jsonString), &msg)
	if err != nil {
		return msg, err
	}

	return msg, nil
}
