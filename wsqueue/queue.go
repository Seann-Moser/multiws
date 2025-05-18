package wsqueue

import (
	"context"
	"github.com/Seann-Moser/multiws/wsmodels"
)

type Queue[T any] interface {
	Subscribe(ctx context.Context, topic string) <-chan wsmodels.Event
	Produce(ctx context.Context, topic string) chan<- wsmodels.Event

	ConsumeEvent(ctx context.Context, topic string) wsmodels.Event
	ProduceEvent(ctx context.Context, topic string, e wsmodels.Event)

	Close()
}
