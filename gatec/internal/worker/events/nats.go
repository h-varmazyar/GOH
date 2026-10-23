package events

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type NATSBus struct {
	conn            *nats.Conn
	js              nats.JetStreamContext
	stream          string
	candleSubject   string
	snapshotSubject string
	consumer        string
	mu              sync.Mutex
	closed          bool
}

func NewNATSBus(cfg configs.NATS) (*NATSBus, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("NATS_URL is empty")
	}
	conn, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}
	stream := cfg.Stream
	if stream == "" {
		stream = "GATEC_WORKERS"
	}
	subject := cfg.CandleSubject
	if subject == "" {
		subject = CandleSubject
	}
	snapshotSubject := cfg.SnapshotSubject
	if snapshotSubject == "" {
		snapshotSubject = IndicatorSnapshotSubject
	}
	streamInfo, infoErr := js.StreamInfo(stream)
	if infoErr == nil {
		wanted := append([]string(nil), streamInfo.Config.Subjects...)
		for _, required := range []string{subject, snapshotSubject, "signals.>"} {
			found := false
			for _, existing := range wanted {
				if existing == required {
					found = true
					break
				}
			}
			if !found {
				wanted = append(wanted, required)
			}
		}
		if len(wanted) != len(streamInfo.Config.Subjects) {
			updated := streamInfo.Config
			updated.Subjects = wanted
			if _, err = js.UpdateStream(&updated); err != nil {
				conn.Close()
				return nil, fmt.Errorf("update NATS stream subjects: %w", err)
			}
		}
	} else if _, err = js.AddStream(&nats.StreamConfig{Name: stream, Subjects: []string{subject, snapshotSubject, "signals.>"}, Storage: nats.FileStorage}); err != nil && err != nats.ErrStreamNameAlreadyInUse {
		conn.Close()
		return nil, fmt.Errorf("create NATS stream: %w", err)
	}
	return &NATSBus{conn: conn, js: js, stream: stream, candleSubject: subject, snapshotSubject: snapshotSubject, consumer: cfg.ConsumerPrefix}, nil
}

func (b *NATSBus) PublishCandle(ctx context.Context, candle models.Candle) error {
	data, err := EncodeCandle(candle)
	if err != nil {
		return err
	}
	_, err = b.js.Publish(b.candleSubject, data, nats.Context(ctx))
	return err
}

func (b *NATSBus) SubscribeCandles(ctx context.Context, name string, handler CandleHandler) (io.Closer, error) {
	if name == "" {
		name = b.consumer
	}
	if name == "" {
		name = "gatec-indicator"
	}
	subscription, err := b.js.PullSubscribe(b.candleSubject, name, nats.BindStream(b.stream), nats.ManualAck())
	if err != nil {
		return nil, fmt.Errorf("subscribe to NATS candles: %w", err)
	}
	stop := make(chan struct{})
	go func() {
		defer close(stop)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			messages, fetchErr := subscription.Fetch(16, nats.MaxWait(time.Second))
			if fetchErr != nil {
				if ctx.Err() != nil {
					return
				}
				if !errors.Is(fetchErr, nats.ErrTimeout) {
					logrus.WithError(fetchErr).Warn("fetch NATS candle batch")
				}
				continue
			}
			for _, message := range messages {
				candle, decodeErr := DecodeCandle(message.Data)
				if decodeErr == nil {
					decodeErr = handler(ctx, candle)
				}
				if decodeErr == nil {
					_ = message.Ack()
				} else {
					_ = message.NakWithDelay(time.Second)
				}
			}
		}
	}()
	return closerFunc(func() {
		_ = subscription.Unsubscribe()
		<-stop
	}), nil
}

func (b *NATSBus) PublishSnapshot(ctx context.Context, snapshot CandleSnapshot) error {
	data, err := EncodeSnapshot(snapshot)
	if err != nil {
		return err
	}
	_, err = b.js.Publish(b.snapshotSubject, data, nats.Context(ctx))
	return err
}

func (b *NATSBus) SubscribeSnapshots(ctx context.Context, name string, shards int, handler SnapshotHandler) (io.Closer, error) {
	if name == "" {
		name = b.consumer + "-signal"
	}
	if handler == nil {
		return nil, fmt.Errorf("snapshot handler is required")
	}
	if shards < 1 {
		shards = 1
	}
	subscription, err := b.js.PullSubscribe(b.snapshotSubject, name, nats.BindStream(b.stream), nats.ManualAck())
	if err != nil {
		return nil, fmt.Errorf("subscribe to candle snapshots: %w", err)
	}
	stop := make(chan struct{})
	go func() {
		defer close(stop)
		type queuedSnapshot struct {
			message  *nats.Msg
			snapshot CandleSnapshot
		}
		lanes := make([]chan queuedSnapshot, shards)
		var workers sync.WaitGroup
		for i := range lanes {
			lanes[i] = make(chan queuedSnapshot, 32)
			workers.Add(1)
			go func(queue <-chan queuedSnapshot) {
				defer workers.Done()
				for item := range queue {
					handlerErr := handler(ctx, item.snapshot)
					if handlerErr == nil {
						_ = item.message.Ack()
					} else {
						_ = item.message.NakWithDelay(time.Second)
					}
				}
			}(lanes[i])
		}
		defer func() {
			for _, lane := range lanes {
				close(lane)
			}
			workers.Wait()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			messages, fetchErr := subscription.Fetch(32, nats.MaxWait(time.Second))
			if fetchErr != nil {
				if ctx.Err() != nil {
					return
				}
				if !errors.Is(fetchErr, nats.ErrTimeout) {
					logrus.WithError(fetchErr).Warn("fetch NATS snapshot batch")
				}
				continue
			}
			for _, message := range messages {
				snapshot, decodeErr := DecodeSnapshot(message.Data)
				if decodeErr != nil {
					_ = message.NakWithDelay(time.Second)
					continue
				}
				shard := int(snapshot.Candle.MarketID % uint(shards))
				select {
				case lanes[shard] <- queuedSnapshot{message: message, snapshot: snapshot}:
				case <-ctx.Done():
					_ = message.NakWithDelay(time.Second)
					return
				}
			}
		}
	}()
	return closerFunc(func() { _ = subscription.Unsubscribe(); <-stop }), nil
}

func (b *NATSBus) PublishSignal(ctx context.Context, signal domain.Signal) error {
	data, err := EncodeSignal(signal)
	if err != nil {
		return err
	}
	_, err = b.js.Publish(SignalSubject(signal), data, nats.Context(ctx))
	return err
}

// Publish adapts the event bus to the signal-domain publisher contract.
func (b *NATSBus) Publish(ctx context.Context, signal domain.Signal) error {
	return b.PublishSignal(ctx, signal)
}

func SignalSubject(signal domain.Signal) string {
	prefix := "signals"
	if signal.Mode == domain.ModeShadow {
		prefix = "signals.shadow"
	}
	return strings.Join([]string{prefix, subjectToken(signal.Market), subjectToken(signal.Timeframe), subjectToken(signal.Symbol), strings.ToLower(string(signal.Type))}, ".")
}

func subjectToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if valid {
			out.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}

func (b *NATSBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true
	return b.conn.Drain()
}

var _ domain.SignalPublisher = (*NATSBus)(nil)
