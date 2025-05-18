package wsmodels

import "encoding/json"

type EventType string

const (
	EventTypeUserJoined      EventType = "UserJoined"
	EventTypeUserLeft        EventType = "UserLeft"
	EventTypeUserDataChanged EventType = "UserDataChanged"
	EventTypeGeneral         EventType = "General"
)

type Event struct {
	SenderID   string
	ReceiverID string
	Type       EventType
	Data       interface{}
	Message    string
	Remote     bool
}

func (e *Event) MarshalBinary() ([]byte, error) {
	return json.Marshal(e)
}

func (e *Event) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, e)
}
