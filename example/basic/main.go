package main

import (
	"context"
	"fmt"
	"github.com/Seann-Moser/multiws/wsmodels"
	"github.com/Seann-Moser/multiws/wsqueue"
	"github.com/Seann-Moser/multiws/wssession"
	redis_store "github.com/eko/gocache/store/redis/v4"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"log"
	"log/slog"
	"net/http"
)

func main() {
	r := mux.NewRouter()
	h := New()
	// WebSocket route
	r.HandleFunc("/ws", h.handleWebSocket)

	// Static file serving
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	// Start the server
	fmt.Println("Server started on http://localhost:8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

type handle struct {
	redisStore *redis_store.RedisStore
}

func New() *handle {
	redisStore := redis_store.NewRedis(redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	}))
	return &handle{
		redisStore: redisStore,
	}
}
func (h *handle) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	slog.Info("consumer group", "session", r.URL.Query().Get("session"), "group", r.URL.Query().Get("name"))
	q, err := wsqueue.NewRedisQueue[wsmodels.Event]("127.0.0.1:6379",
		"", 0, 10, r.URL.Query().Get("session")+r.URL.Query().Get("name"), r.URL.Query().Get("name"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer q.Close()

	s := wssession.NewBaseSession(redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	}), q)
	u := wsmodels.User{
		Name: r.URL.Query().Get("name"),
	}
	slog.Info("user connected", "user", u)
	key := r.URL.Query().Get("session") + "_session_info"
	results := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	}).Get(r.Context(), key)
	if results.Err() != nil {
		slog.Error(results.Err().Error())
		return
	}
	slog.Info("results", "results", string(results.String()))
	err = s.Init(context.Background(), r.URL.Query().Get("session"), &u)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	hs := s.WsHandler(func(w http.ResponseWriter, r *http.Request, receiveEvent wsmodels.Event) {
		fmt.Println("received event", receiveEvent, "from", r.URL.Query().Get("name"))
	})
	hs(w, r)

	return
}
