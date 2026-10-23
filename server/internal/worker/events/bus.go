package events

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
)

const CandleSubject = "market.candle.updated"

type CandleHandler func(context.Context, models.Candle) error
type SnapshotHandler func(context.Context, CandleSnapshot) error

type CandleBus interface {
	PublishCandle(context.Context, models.Candle) error
	SubscribeCandles(context.Context, string, CandleHandler) (io.Closer, error)
	Close() error
}

type EventBus interface {
	CandleBus
	PublishSnapshot(context.Context, CandleSnapshot) error
	SubscribeSnapshots(context.Context, string, int, SnapshotHandler) (io.Closer, error)
	PublishSignal(context.Context, domain.Signal) error
}

type ChannelBus struct {
	mu                  sync.RWMutex
	subscribers         map[string]chan models.Candle
	snapshotSubscribers map[string]chan CandleSnapshot
	signals             chan domain.Signal
}

func NewChannelBus() *ChannelBus {
	return &ChannelBus{subscribers: make(map[string]chan models.Candle), snapshotSubscribers: make(map[string]chan CandleSnapshot), signals: make(chan domain.Signal, 256)}
}

func (b *ChannelBus) PublishSnapshot(ctx context.Context, snapshot CandleSnapshot) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, subscriber := range b.snapshotSubscribers {
		select {
		case subscriber <- snapshot:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (b *ChannelBus) SubscribeSnapshots(ctx context.Context, name string, _ int, handler SnapshotHandler) (io.Closer, error) {
	if name == "" || handler == nil {
		return nil, fmt.Errorf("snapshot subscription name and handler are required")
	}
	b.mu.Lock()
	if _, exists := b.snapshotSubscribers[name]; exists {
		b.mu.Unlock()
		return nil, fmt.Errorf("snapshot subscription %q already exists", name)
	}
	messages := make(chan CandleSnapshot, 256)
	b.snapshotSubscribers[name] = messages
	b.mu.Unlock()
	stop := make(chan struct{})
	go func() {
		defer close(stop)
		for {
			select {
			case <-ctx.Done():
				return
			case snapshot, ok := <-messages:
				if !ok {
					return
				}
				if err := handler(ctx, snapshot); err != nil {
					continue
				}
			}
		}
	}()
	return closerFunc(func() {
		b.mu.Lock()
		if current, ok := b.snapshotSubscribers[name]; ok && current == messages {
			delete(b.snapshotSubscribers, name)
			close(messages)
		}
		b.mu.Unlock()
		<-stop
	}), nil
}

func (b *ChannelBus) PublishSignal(ctx context.Context, signal domain.Signal) error {
	select {
	case b.signals <- signal:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (b *ChannelBus) Publish(ctx context.Context, signal domain.Signal) error {
	return b.PublishSignal(ctx, signal)
}
func (b *ChannelBus) Signals() <-chan domain.Signal { return b.signals }

func (b *ChannelBus) PublishCandle(ctx context.Context, candle models.Candle) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, subscriber := range b.subscribers {
		select {
		case subscriber <- candle:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (b *ChannelBus) SubscribeCandles(ctx context.Context, name string, handler CandleHandler) (io.Closer, error) {
	if name == "" || handler == nil {
		return nil, fmt.Errorf("candle subscription name and handler are required")
	}
	b.mu.Lock()
	if _, exists := b.subscribers[name]; exists {
		b.mu.Unlock()
		return nil, fmt.Errorf("candle subscription %q already exists", name)
	}
	messages := make(chan models.Candle, 256)
	b.subscribers[name] = messages
	b.mu.Unlock()

	stop := make(chan struct{})
	go func() {
		defer close(stop)
		for {
			select {
			case <-ctx.Done():
				return
			case candle, ok := <-messages:
				if !ok {
					return
				}
				if err := handler(ctx, candle); err != nil {
					continue
				}
			}
		}
	}()
	return closerFunc(func() {
		b.mu.Lock()
		if current, exists := b.subscribers[name]; exists && current == messages {
			delete(b.subscribers, name)
			close(messages)
		}
		b.mu.Unlock()
	}), nil
}

func (b *ChannelBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for name, subscriber := range b.subscribers {
		delete(b.subscribers, name)
		close(subscriber)
	}
	for name, subscriber := range b.snapshotSubscribers {
		delete(b.snapshotSubscribers, name)
		close(subscriber)
	}
	return nil
}

type closerFunc func()

func (f closerFunc) Close() error {
	f()
	return nil
}

func EncodeCandle(candle models.Candle) ([]byte, error) { return json.Marshal(candle) }

func DecodeCandle(data []byte) (models.Candle, error) {
	var candle models.Candle
	err := json.Unmarshal(data, &candle)
	return candle, err
}
