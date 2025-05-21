package wssession

import (
	"context"
	"github.com/Seann-Moser/multiws/wsmodels"
	"net/http"
)

type WsHandler func(w http.ResponseWriter, r *http.Request, receiveEvent wsmodels.Event)

type Session interface {
	ID() string
	Status() string
	Init(ctx context.Context, sessionID string, user *wsmodels.User) error

	User() *wsmodels.User
	ListUsers() []*wsmodels.User
	Disconnect()

	GetHistory() []*wsmodels.Event
	SendEvent(ctx context.Context, e wsmodels.Event)
	GetEvent() chan<- wsmodels.Event

	WsHandler(h WsHandler) http.HandlerFunc
}
