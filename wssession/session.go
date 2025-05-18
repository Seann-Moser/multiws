package wssession

import (
	"context"
	"github.com/Seann-Moser/multiws/wsmodels"
	"net/http"
)

type WsHandler func(w http.ResponseWriter, r *http.Request, receiveEvent chan wsmodels.Event, sendEvent chan wsmodels.Event)

type Session interface {
	ID() string
	Status() string
	Init(ctx context.Context) error

	User() string
	ListUsers()
	Disconnect()

	GetHistory()
	SendEvent()
	GetEvent()

	WsHandler(h WsHandler) http.HandlerFunc
}
