package logger

import "encoding/json"

func (a Action) String() string {
	return string(a)
}

type Action string

const (
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
)

type RequiredFields struct {
	UserID     int64  `json:"userId,omitempty"` //nolint:tagliatelle // the audit wire format
	UserIP     string `json:"userIp,omitempty"` //nolint:tagliatelle // the audit wire format
	Action     Action `json:"action,omitempty"`
	Date       int64  `json:"date,omitempty"`
	DomainId   int64  //nolint:revive // untagged: the field name is the key in the audit message
	ObjectName string
}

type Record struct {
	ID       int64  `json:"id,omitempty"`
	NewState []byte `json:"newState,omitempty"` //nolint:tagliatelle // the audit wire format
}

type Message struct {
	RequiredFields `json:"requiredFields"` //nolint:tagliatelle // the audit wire format

	Records []Record `json:"records,omitempty"`
}

func (m *Message) ToJSON() []byte {
	data, _ := json.Marshal(m)

	return data
}
