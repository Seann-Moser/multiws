package wsmodels

import (
	"encoding/json"
	"time"
)

type Session struct {
	Users        []*User                `json:"users"`
	ID           string                 `json:"id"`
	History      []*Event               `json:"history"`
	MaxHistory   int                    `json:"maxHistory"`
	IdleDuration time.Duration          `json:"idleDuration"`
	Meta         map[string]interface{} `json:"meta"`
	Self         User                   `json:"-"`
}

// MarshalBinary implements the encoding.BinaryMarshaler interface
func (s Session) MarshalBinary() ([]byte, error) {
	// Convert the session to a JSON representation
	type sessionAlias Session
	alias := sessionAlias(s)

	// Convert IdleDuration to seconds for serialization
	alias.IdleDuration = s.IdleDuration

	return json.Marshal(alias)
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface
func (s *Session) UnmarshalBinary(data []byte) error {
	// Unmarshal the JSON data
	type sessionAlias Session
	alias := sessionAlias{}
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	// Convert the seconds back to a time.Duration
	alias.IdleDuration = time.Duration(alias.IdleDuration) * time.Second
	*s = Session(alias)
	return nil
}
