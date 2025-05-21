package wsmodels

import (
	"encoding/json"
	"fmt"
)

const (
	EventTypeUserJoined      string = "UserJoined"
	EventTypeUserLeft        string = "UserLeft"
	EventTypeUserDataChanged string = "UserDataChanged"
	EventTypeGeneral         string = "General"
)

type Event struct {
	SenderID   string
	ReceiverID string
	Type       string
	Data       string
	Message    string
	Remote     bool
}

func (e *Event) Set(data interface{}) (err error) {
	d, err := json.Marshal(data)
	if err != nil {
		return err
	}
	e.Data = string(d)
	return
}

func GetDataEvent[T any](e Event) (out *T, err error) {
	var t T
	if e.Data == "" {
		return nil, fmt.Errorf("event data is empty")
	}
	err = json.Unmarshal([]byte(e.Data), &t)
	if err != nil {
		return
	}
	out = &t
	return
}
