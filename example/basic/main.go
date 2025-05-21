package main

import (
	"context"
	"fmt"
	"github.com/Seann-Moser/multiws/wsmodels"
	"github.com/Seann-Moser/multiws/wsqueue"
	"github.com/Seann-Moser/multiws/wssession"
	"github.com/eko/gocache/lib/v4/cache"
	redis_store "github.com/eko/gocache/store/redis/v4"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"log"
	"log/slog"
	"net/http"
)

func main() {
	r := mux.NewRouter()

	// WebSocket route
	r.HandleFunc("/ws", handleWebSocket)

	// Static file serving
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	// Start the server
	fmt.Println("Server started on http://localhost:8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	q, err := wsqueue.NewRedisQueue[wsmodels.Event]("127.0.0.1:6379",
		"", 0, 10, r.URL.Query().Get("session"), r.URL.Query().Get("name"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	redisStore := redis_store.NewRedis(redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	}))

	c := cache.New[wsmodels.Session](redisStore)
	s := wssession.NewBaseSession(c, q)
	u := wsmodels.User{
		Name: r.URL.Query().Get("name"),
	}
	slog.Info("user connected", "user", u)
	err = s.Init(context.Background(), r.URL.Query().Get("session"), &u)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h := s.WsHandler(func(w http.ResponseWriter, r *http.Request, receiveEvent wsmodels.Event) {
		fmt.Println("received event", receiveEvent, "from", r.URL.Query().Get("name"))
	})
	h(w, r)

	return
}
