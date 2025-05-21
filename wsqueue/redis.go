package wsqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Seann-Moser/multiws/wsmodels"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"strings"
)

type RedisStreamQueue[T any] struct {
	client *redis.Client
	// how many events to buffer in the Go channel
	channelSize int
	group       string
	consumer    string
}

func (q *RedisStreamQueue[T]) ConsumeEvent(ctx context.Context, topic string) wsmodels.Event {
	//TODO implement me
	panic("implement me")
}

func (q *RedisStreamQueue[T]) ProduceEvent(ctx context.Context, topic string, e wsmodels.Event) {
	//TODO implement me
	panic("implement me")
}

// NewRedisStreamQueue returns a queue that speaks Redis Streams.
// addr/password/db are passed straight to redis.Options
func NewRedisQueue[T any](
	addr, password string,
	db, channelSize int, groupBox, consumer string,
) (Queue[T], error) {
	if channelSize < 0 {
		channelSize = 0
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		slog.Error("NewRedisStreamQueue: redis", "err", err)
		return nil, fmt.Errorf("NewRedisStreamQueue: redis:%w", err)
	}
	if groupBox == "" {
		return nil, fmt.Errorf("NewRedisStreamQueue: groupBox not specified")
	}
	if consumer == "" {
		consumer = uuid.New().String()
		slog.Info("consumer not specified, using random consumer", "consumer", consumer)
	}
	return &RedisStreamQueue[T]{
		client:      client,
		channelSize: channelSize,
		group:       groupBox,
		consumer:    consumer,
	}, nil
}

// Produce pushes events into the given stream (topic).  Returns
// a Go channel you can send into.
func (q *RedisStreamQueue[T]) Produce(
	ctx context.Context,
	topic string,
) chan<- wsmodels.Event {
	ch := make(chan wsmodels.Event, q.channelSize)
	go func() {
		defer close(ch)
		defer fmt.Println("finished produce")
		for evt := range ch {
			data, err := json.Marshal(evt)
			if err != nil {
				slog.Error("Produce: marshal", "err", err)
				continue
			}
			// XAdd will append to the stream named by topic
			if _, err := q.client.XAdd(ctx, &redis.XAddArgs{
				Stream: topic,
				Values: map[string]interface{}{"data": data},
			}).Result(); err != nil {
				slog.Error("Produce: XAdd", "err", err)
			}
		}
	}()
	return ch
}

// Subscribe creates (if needed) and then joins a consumer-group on the given
// stream (topic).  Each group sees every message; within a group multiple
// consumers can share load, but each group gets all messages.
//
//   - group:   a logical fan-out channel (e.g. "notifications")
//   - consumer: a unique name for *this* consumer (e.g. pod-id, host name)
//
// It returns a Go channel of events; you must ack each message yourself
// by returning nil (ACK) or requeueing by returning an error.
func (q *RedisStreamQueue[T]) Subscribe(
	ctx context.Context,
	topic string,
) <-chan wsmodels.Event {
	// ensure the group exists (start reading new messages)
	if err := q.client.XGroupCreateMkStream(ctx, topic, q.group, "$").Err(); err != nil {
		// ignore BUSYGROUP if already created
		if !strings.Contains(err.Error(), "BUSYGROUP") {
			slog.Error("Subscribe: XGroupCreateMkStream", "err", err)
			return nil
		}
	}
	defer q.client.XGroupDelConsumer(context.Background(), topic, q.group, q.consumer)

	out := make(chan wsmodels.Event, q.channelSize)
	go func() {
		defer fmt.Println("finished subscribe")
		defer close(out)
		for {
			// read one message at a time, block indefinitely
			streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    q.group,
				Consumer: q.consumer,
				Streams:  []string{topic, ">"},
				Count:    1,
				Block:    0,
			}).Result()
			if err != nil {
				slog.Error("Subscribe: XReadGroup", "err", err)
				return
			}
			for _, msg := range streams[0].Messages {
				raw, ok := msg.Values["data"].(string)
				if !ok {
					slog.Error("Subscribe: bad payload type", "values", msg.Values)
					// ack so we don’t loop forever
					q.client.XAck(ctx, topic, q.group, msg.ID)
					continue
				}

				var evt wsmodels.Event
				if err := json.Unmarshal([]byte(raw), &evt); err != nil {
					slog.Error("Subscribe: unmarshal", "err", err)
					// ack so broken messages don’t block
					q.client.XAck(ctx, topic, q.group, msg.ID)
					continue
				}

				out <- evt
				// ACK immediately after sending into the channel:
				if err := q.client.XAck(ctx, topic, q.group, msg.ID).Err(); err != nil {
					slog.Error("Subscribe: XAck", "err", err)
				}
			}
		}
	}()
	return out
}

// Close the Redis client
func (q *RedisStreamQueue[T]) Close() {
	if err := q.client.Close(); err != nil {
		slog.Error("Close: redis", "err", err)
	}
}
