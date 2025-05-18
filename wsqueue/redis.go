package wsqueue

import (
	"context"
	"github.com/Seann-Moser/multiws/wsmodels"
	"github.com/redis/go-redis/v9"
	"log/slog"
)

type RedisQueue[T any] struct {
	client      *redis.Client
	channelName string
	channelSize int
}

func NewRedisQueue[T any](addr, password string, db int, channelName string, channelSize int) Queue[T] {
	if channelSize < 0 {
		channelSize = 0
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisQueue[T]{
		client:      client,
		channelName: channelName,
		channelSize: channelSize,
	}
}

func (q *RedisQueue[T]) Subscribe(ctx context.Context, topic string) <-chan wsmodels.Event {
	ch := make(chan wsmodels.Event, q.channelSize)
	pubsub := q.client.Subscribe(ctx, topic)
	go func() {
		for msg := range pubsub.Channel() {
			var event wsmodels.Event
			if err := event.UnmarshalBinary([]byte(msg.Payload)); err != nil {
				slog.Error("failed to unmarshal event", "error", err)
				continue
			}
			ch <- event
		}
		close(ch)
	}()
	return ch
}

func (q *RedisQueue[T]) Produce(ctx context.Context, topic string) chan<- wsmodels.Event {
	ch := make(chan wsmodels.Event, q.channelSize)
	go func() {
		for event := range ch {
			data, err := event.MarshalBinary()
			if err != nil {
				slog.Error("failed to marshal event", "error", err)
				continue
			}
			if err := q.client.Publish(ctx, topic, data).Err(); err != nil {
				slog.Error("failed to publish event", "error", err)
			}
		}
	}()
	return ch
}

func (q *RedisQueue[T]) ConsumeEvent(ctx context.Context, topic string) wsmodels.Event {
	res, err := q.client.BLPop(ctx, 0, topic).Result()
	if err != nil {
		slog.Error("failed to consume event", "error", err)
		return wsmodels.Event{}
	}
	var event wsmodels.Event
	if err := event.UnmarshalBinary([]byte(res[1])); err != nil {
		slog.Error("failed to unmarshal event", "error", err)
	}
	return event
}

func (q *RedisQueue[T]) ProduceEvent(ctx context.Context, topic string, e wsmodels.Event) {
	data, err := e.MarshalBinary()
	if err != nil {
		slog.Error("failed to marshal event", "error", err)
		return
	}
	if err := q.client.RPush(ctx, topic, data).Err(); err != nil {
		slog.Error("failed to produce event", "error", err)
	}
}

func (q *RedisQueue[T]) Close() {
	if err := q.client.Close(); err != nil {
		slog.Error("failed to close redis client", "error", err)
	}
}
