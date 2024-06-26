package rpc

import (
	"encoding/json"
)

const GetSnapshotsPayloadType = "getsnapshotsmsg"

type GetSnapshotsMsg struct {
	ColonyName string `json:"colonyname"`
	MsgType    string `json:"msgtype"`
}

func CreateGetSnapshotsMsg(colonyName string) *GetSnapshotsMsg {
	msg := &GetSnapshotsMsg{}
	msg.MsgType = GetSnapshotsPayloadType
	msg.ColonyName = colonyName

	return msg
}

func (msg *GetSnapshotsMsg) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (msg *GetSnapshotsMsg) Equals(msg2 *GetSnapshotsMsg) bool {
	if msg2 == nil {
		return false
	}

	if msg.MsgType == msg2.MsgType &&
		msg.ColonyName == msg2.ColonyName {
		return true
	}

	return false
}

func CreateGetSnapshotsMsgFromJSON(jsonString string) (*GetSnapshotsMsg, error) {
	var msg *GetSnapshotsMsg

	err := json.Unmarshal([]byte(jsonString), &msg)
	if err != nil {
		return msg, err
	}

	return msg, nil
}
