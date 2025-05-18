package wssession

import (
	"context"
	"github.com/Seann-Moser/multiws/wsmodels"
	"github.com/Seann-Moser/multiws/wsqueue"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/gorilla/websocket"
	"log/slog"
	"net/http"
	"sync"
)

var _ Session = &BaseSession{}

type BaseSession struct {
	sessionCache   cache.Cache[wsmodels.Session]
	queue          wsqueue.Queue[wsmodels.Event]
	currentSession wsmodels.Session
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func NewBaseSession() Session {
	return &BaseSession{}
}

func (s *BaseSession) ID() string {
	//TODO implement me
	panic("implement me")
}

func (s *BaseSession) Status() string {
	//TODO implement me
	panic("implement me")
}

func (s *BaseSession) Init(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (s *BaseSession) User() string {
	//TODO implement me
	panic("implement me")
}

func (s *BaseSession) ListUsers() {
	//TODO implement me
	panic("implement me")
}

func (s *BaseSession) Disconnect() {
	//TODO implement me
	panic("implement me")
}

func (s *BaseSession) GetHistory() {
	//TODO implement me
	panic("implement me")
}

func (s *BaseSession) SendEvent() {
	//TODO implement me
	panic("implement me")
}

func (s *BaseSession) GetEvent() {
	//TODO implement me
	panic("implement me")
}

func (s *BaseSession) WsHandler(h WsHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("failed to upgrade websocket connection:", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithCancel(r.Context())
		defer conn.Close()
		defer cancel()
		defer s.Disconnect()
		wg := sync.WaitGroup{}
		wg.Add(1)
		wg.Add(1)
		inboundChan := make(chan wsmodels.Event)
		outboundChan := make(chan wsmodels.Event)

		/*
			All of the bellow logic is for local ws connection data todo setup sesion listeners

		*/
		go func() {
			// listen to queue
			defer wg.Done()
			e := s.queue.Subscribe(ctx, "ses_"+s.ID())
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-e:
					if !ok {
						slog.Error("inbound channel closed")
						return
					}
					//todo pre processing
					msg.Remote = true
					if msg.ReceiverID != "" && msg.ReceiverID != s.currentSession.Self.Id {
						continue
					}
					s.processEvent(msg)
					select {
					case <-ctx.Done():
						return
					case outboundChan <- msg:
					default:
						slog.Error("outbound channel full")
					}
					select {
					case <-ctx.Done():
						return
					case inboundChan <- msg:
					default:
						slog.Error("inbound channel full")
					}
				}
			}
		}()

		// outbound message logic
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-outboundChan:
					if !ok {
						slog.Info("outbound channel closed")
						return
					}
					if event.Remote {
						continue
					}
					// todo write to queue

					err := conn.WriteJSON(event)
					if err != nil {
						slog.Error("failed to write event:", "err", err)
						return
					}
				}
			}
		}()
		wg.Add(1)
		// prcoess ws inbound messagew
		produceChan := s.queue.Produce(ctx, "ses_"+s.ID())
		go func() {
			defer wg.Done()
			for {
				var event wsmodels.Event
				err := conn.ReadJSON(&event)
				if err != nil {
					slog.Error("failed to read JSON message:", "err", err)
					return
				}
				s.processEvent(event)
				select {
				case <-ctx.Done():
					return
				case produceChan <- event:
				default:
					slog.Info("produceChan full")
				}
				select {
				case <-ctx.Done():
					return
				case inboundChan <- event:
				default:
					slog.Info("channel full")
				}
			}
		}()

		wg.Wait()
		slog.Info("closed websocket connection")
	}
}

func (s *BaseSession) processEvent(e wsmodels.Event) {
	//todo do any pre processing like updating history/user data session info etc
	// todo write to queue
	switch e.Type {
	case wsmodels.EventTypeUserJoined:
	case wsmodels.EventTypeUserLeft:
	case wsmodels.EventTypeUserDataChanged:
	case wsmodels.EventTypeGeneral:
	}
}
