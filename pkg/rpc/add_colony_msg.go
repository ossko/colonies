package rpc

import (
	"colonies/pkg/core"
	"encoding/json"
)

const AddColonyPayloadType = "addcolonymsg"

type AddColonyMsg struct {
	Colony  *core.Colony `json:"colony"`
	MsgType string       `json:"msgtype"`
}

func CreateAddColonyMsg(colony *core.Colony) *AddColonyMsg {
	msg := &AddColonyMsg{}
	msg.Colony = colony
	msg.MsgType = AddColonyPayloadType

	return msg
}

func (msg *AddColonyMsg) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (msg *AddColonyMsg) ToJSONIndent() (string, error) {
	jsonBytes, err := json.MarshalIndent(msg, "", "    ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func CreateAddColonyMsgFromJSON(jsonString string) (*AddColonyMsg, error) {
	var msg *AddColonyMsg

	err := json.Unmarshal([]byte(jsonString), &msg)
	if err != nil {
		return msg, err
	}

	return msg, nil
}