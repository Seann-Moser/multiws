package wssession

import (
	"context"
	"fmt"
	"github.com/Seann-Moser/multiws/wsmodels"
	"github.com/Seann-Moser/multiws/wsqueue"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/gorilla/websocket"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

var _ Session = &BaseSession{}

type BaseSession struct {
	sessionCache   *cache.Cache[wsmodels.Session]
	queue          wsqueue.Queue[wsmodels.Event]
	currentSession *wsmodels.Session

	produceChan chan<- wsmodels.Event
	consumeChan <-chan wsmodels.Event

	wsOutbound chan wsmodels.Event

	lastEventSent time.Time
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections
	},
}

func NewBaseSession(
	sessionCache *cache.Cache[wsmodels.Session],
	queue wsqueue.Queue[wsmodels.Event]) Session {
	return &BaseSession{
		sessionCache: sessionCache,
		queue:        queue,
		wsOutbound:   make(chan wsmodels.Event, 100),
	}
}

func (s *BaseSession) ID() string {
	return s.currentSession.ID
}

func (s *BaseSession) Status() string {
	return ""
}

func (s *BaseSession) Init(ctx context.Context, sessionID string, user *wsmodels.User) error {
	if s.currentSession != nil {
		return fmt.Errorf("session already initialized")
	}
	s.lastEventSent = time.Now()
	key := sessionID + "_session_info"
	ses, err := s.sessionCache.Get(ctx, key)
	if err == nil {
		s.currentSession = &ses
		slog.Info("session found in cache")
		if user != nil {
			user.Host = len(s.currentSession.Users) == 0
			user.Joined = time.Now().Unix()
			user.Status = wsmodels.StatusConnected
			user.LastSeen = time.Now().Unix()
			if user.Meta == nil {
				user.Meta = map[string]interface{}{}
			}
			s.currentSession.Self = *user
		}

		s.produceChan = s.queue.Produce(ctx, "ses_"+s.ID())
		s.consumeChan = s.queue.Subscribe(ctx, "ses_"+s.ID())

		e := wsmodels.Event{
			Type: wsmodels.EventTypeUserJoined,
		}
		err = e.Set(s.currentSession.Self)
		if err != nil {
			return err
		}
		s.SendEvent(ctx, e)
		return nil
	}
	slog.Info("session not found in cache")

	session := wsmodels.Session{
		Users:        []*wsmodels.User{},
		ID:           sessionID,
		History:      make([]*wsmodels.Event, 0),
		MaxHistory:   20,
		IdleDuration: time.Minute,
		Meta:         map[string]interface{}{},
	}
	if user != nil {
		user.Host = true
		user.Joined = time.Now().Unix()
		user.Status = wsmodels.StatusConnected
		user.LastSeen = time.Now().Unix()
		if user.Meta == nil {
			user.Meta = map[string]interface{}{}
		}
		session.Self = *user
	}

	err = s.sessionCache.Set(ctx, key, session, store.WithExpiration(6*time.Hour))
	if err != nil {
		slog.Error("failed to set session info", "err", err)
		return err
	}
	s.currentSession = &session
	s.produceChan = s.queue.Produce(ctx, "ses_"+s.ID())
	s.consumeChan = s.queue.Subscribe(ctx, "ses_"+s.ID())

	return nil
}

func (s *BaseSession) User() *wsmodels.User {
	if s.currentSession == nil {
		return nil
	}
	return &s.currentSession.Self
}

func (s *BaseSession) ListUsers() []*wsmodels.User {
	if s.currentSession == nil {
		return nil
	}
	return s.currentSession.Users
}

func (s *BaseSession) Disconnect() {
	if s.currentSession == nil {
		return
	}
	e := wsmodels.Event{
		Type: wsmodels.EventTypeUserLeft,
	}
	err := e.Set(s.currentSession.Self)
	if err != nil {
		return
	}
	s.SendEvent(context.Background(), e)

	s.queue.Close()
	s.currentSession = nil
}

func (s *BaseSession) GetHistory() []*wsmodels.Event {
	if s.currentSession == nil {
		return nil
	}
	return s.currentSession.History
}

func (s *BaseSession) SendEvent(ctx context.Context, e wsmodels.Event) {
	if s.currentSession == nil {
		return
	}
	if e.SenderID == "" {
		e.SenderID = s.currentSession.Self.Id
	}
	s.lastEventSent = time.Now()

	select {
	case <-ctx.Done():
		return
	case s.produceChan <- e:
		return
	default:
		slog.Error("channel full")
	}
	if s.currentSession.Self.Status == wsmodels.StatusIdle {
		s.currentSession.Self.Status = wsmodels.StatusConnected
		es := wsmodels.Event{
			Type:    wsmodels.EventTypeUserDataChanged,
			Message: "",
			Remote:  false,
		}
		err := es.Set(s.currentSession.Self)
		if err != nil {
			slog.Error("failed to set user data", "err", err)
			return
		}
		select {
		case <-ctx.Done():
			return
		case s.produceChan <- es:
			return
		default:
			slog.Error("channel full")
		}
	}
}

func (s *BaseSession) GetEvent() chan<- wsmodels.Event {
	return nil
}

func (s *BaseSession) WsHandler(h WsHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("failed to upgrade websocket connection:", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer conn.Close()
		defer s.Disconnect()
		if s.currentSession == nil {
			slog.Error("session not initialized")
			return
		}
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		wg := sync.WaitGroup{}

		/*
			All of the bellow logic is for local ws connection data todo setup sesion listeners

		*/
		slog.Info("starting ws handler")
		wg.Add(1)
		go func() {
			// listen to queue
			defer wg.Done()
			defer fmt.Println("finished consumeChan ")
			slog.Info("subscribed to queue")
			for {
				select {
				case <-ctx.Done():
					slog.Info("closing queue subscription")
					return
				case msg, ok := <-s.consumeChan:
					if !ok {
						slog.Error("inbound channel closed")
						return
					}
					slog.Info("received event from queue", ": ", msg, "u", s.currentSession.Self.Name)

					//todo pre processing
					msg.Remote = true
					if msg.ReceiverID != "" && msg.ReceiverID != s.currentSession.Self.Id {
						continue
					}
					if s.processEvent(msg) {
						slog.Info("event processed", "event", msg.Type, "remote", msg.Remote, "sender", msg.SenderID, "receiver", msg.ReceiverID)
						continue
					}
					select {
					case <-ctx.Done():
						return
					case s.wsOutbound <- msg:
					default:
						slog.Error("outbound channel full")
					}
				}
			}
		}()

		wg.Add(1)
		go func() {
			ticker := time.NewTicker(time.Second * 5)
			defer ticker.Stop()
			defer fmt.Println("finished ticker")
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if s.currentSession == nil {
						return
					}
					if time.Since(s.lastEventSent) > s.currentSession.IdleDuration && s.currentSession.Self.Status == wsmodels.StatusConnected {
						s.currentSession.Self.Status = wsmodels.StatusIdle
						slog.Info("user idle for too long", "user", s.currentSession.Self.Name, "time", time.Since(s.lastEventSent))
						e := wsmodels.Event{
							Type:    wsmodels.EventTypeUserDataChanged,
							Message: "",
							Remote:  false,
						}
						err := e.Set(s.currentSession.Self)
						if err != nil {
							slog.Error("failed to set user data", "err", err)
							return
						}
						s.SendEvent(ctx, e)
					}
				}
			}

		}()

		// outbound message logic
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer fmt.Println("finished outbound")
			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-s.wsOutbound:
					if !ok {
						slog.Info("outbound channel closed")
						return
					}
					err := conn.WriteJSON(event)
					if err != nil {
						slog.Error("failed to write event:", "err", err)
						return
					}
					h(w, r, event)
				}
			}
		}()

		// prcoess ws inbound messagew
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer fmt.Println("finished inbound")
			for {
				var event wsmodels.Event
				err := conn.ReadJSON(&event)
				if err != nil {
					slog.Error("failed to read JSON message:", "err", err, "")
					s.Disconnect()
					continue
				}
				if s.processEvent(event) {
					continue
				}
				s.lastEventSent = time.Now()
				s.SendEvent(ctx, event)

				if s.currentSession.Self.Status == wsmodels.StatusIdle {
					s.currentSession.Self.Status = wsmodels.StatusConnected
					es := wsmodels.Event{
						Type:    wsmodels.EventTypeUserDataChanged,
						Message: "",
						Remote:  false,
					}
					err := es.Set(s.currentSession.Self)
					if err != nil {
						slog.Error("failed to set user data", "err", err)
						continue
					}
					select {
					case <-ctx.Done():
						return
					case s.produceChan <- es:
						continue
					default:
						slog.Error("channel full")
					}
				}
			}
		}()
		wg.Wait()
		slog.Info("closed websocket connection")
	}
}

func (s *BaseSession) processEvent(e wsmodels.Event) bool {
	//todo do any pre processing like updating history/user data session info etc
	// todo write to queue
	if s.currentSession == nil {
		return true
	}
	// todo if is host sync data to redis client
	switch e.Type {
	case wsmodels.EventTypeUserJoined:
		user, err := wsmodels.GetDataEvent[wsmodels.User](e)
		if err != nil {
			slog.Error("failed to unmarshal user", "err", err, "event", e, "data", string(e.Data))
			return true
		}
		s.currentSession.Users = append(s.currentSession.Users, user)
		if !s.currentSession.Self.Host {
			return false
		}
		key := s.ID() + "_session_info"
		err = s.sessionCache.Set(context.Background(), key, *s.currentSession, store.WithExpiration(6*time.Hour))
		if err != nil {
			slog.Error("failed to set session info", "err", err)
		}
		return false
	case wsmodels.EventTypeUserLeft:
		// update s.current users
		//todo if host left assign to next oldest user
		//s.currentSession.Users
	case wsmodels.EventTypeUserDataChanged:
		// update s.current users
		if !s.currentSession.Self.Host {
			return false
		}

	case wsmodels.EventTypeGeneral:
		s.currentSession.History = append(s.currentSession.History, &e)
		if !s.currentSession.Self.Host {
			return false
		}
		key := s.ID() + "_session_info"
		err := s.sessionCache.Set(context.Background(), key, *s.currentSession, store.WithExpiration(6*time.Hour))
		if err != nil {
			slog.Error("failed to set session info", "err", err)
		}
		return false
	}
	return false
}
