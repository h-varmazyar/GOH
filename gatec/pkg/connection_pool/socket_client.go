package connectionPool

import (
	"errors"
	"sync"

	"github.com/centrifugal/centrifuge-go"
	"github.com/h-varmazyar/gatec/configs"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type AsyncResponse struct {
	Key  string
	Data string
}

type socketClient struct {
	centrifugeClient *centrifuge.Client
	subscriptions    map[string]*centrifuge.Subscription
	cfg              *configs.SocketConfig
	lock             *sync.Mutex
}

func NewClient(cfg *configs.SocketConfig) (*socketClient, error) {
	client := centrifuge.NewJsonClient(cfg.URL, centrifuge.Config{})
	err := client.Connect()
	if err != nil {
		return nil, err
	}
	return &socketClient{
		centrifugeClient: client,
		subscriptions:    make(map[string]*centrifuge.Subscription),
		cfg:              cfg,
		lock:             new(sync.Mutex),
	}, nil
}

func (c *socketClient) hasCapacity() bool {
	return c.cfg.MaxChannelPerConnection > len(c.subscriptions)
}

func (c *socketClient) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()

	for key, sub := range c.subscriptions {
		if sub == nil {
			continue
		}
		if err := sub.Unsubscribe(); err != nil {
			logrus.WithError(err).Warnf("failed to unsubscribe channel key=%s", key)
		}
	}

	if c.centrifugeClient != nil {
		c.centrifugeClient.Close()
	}
}

func (c *socketClient) HasSubscription(key string) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	_, ok := c.subscriptions[key]
	return ok
}

func (c *socketClient) UnregisterChannel(key string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	sub, ok := c.subscriptions[key]
	if !ok || sub == nil {
		return
	}

	if err := sub.Unsubscribe(); err != nil {
		logrus.WithError(err).Warnf("failed to unsubscribe channel key=%s", key)
	}
	delete(c.subscriptions, key)
}

func (c *socketClient) RegisterChannel(key, channelname string, responseChannel chan AsyncResponse, withCompression bool) error {
	if !c.hasCapacity() {
		return errors.New("max channel capacity exeeded!")
	}

	cfg := centrifuge.SubscriptionConfig{
		Recoverable: true,
		JoinLeave:   true,
	}

	if withCompression {
		cfg.Delta = "fossil"
	}

	logrus.Infof("making subscription")

	sub, err := c.centrifugeClient.NewSubscription(channelname, cfg)
	if err != nil {
		return err
	}

	sub.OnSubscribing(func(e centrifuge.SubscribingEvent) {
		log.Printf("Subscribing on channel %s - %d (%s)", sub.Channel, e.Code, e.Reason)
	})
	sub.OnSubscribed(func(e centrifuge.SubscribedEvent) {
		log.Printf("Subscribed on channel %s, (was recovering: %v, recovered: %v)", sub.Channel, e.WasRecovering, e.Recovered)
	})
	sub.OnUnsubscribed(func(e centrifuge.UnsubscribedEvent) {
		log.Printf("Unsubscribed from channel %s - %d (%s)", sub.Channel, e.Code, e.Reason)
	})

	sub.OnError(func(e centrifuge.SubscriptionErrorEvent) {
		log.Printf("Subscription error %s: %s", sub.Channel, e.Error)
	})

	sub.OnPublication(func(e centrifuge.PublicationEvent) {
		response := AsyncResponse{
			Key:  key,
			Data: string(e.Data),
		}
		responseChannel <- response
		// log.Printf("Someone says via channel %s: %s (offset %d)", sub.Channel, response, e.Offset)
	})
	sub.OnJoin(func(e centrifuge.JoinEvent) {
		log.Printf("Someone joined %s: user id %s, client id %s", sub.Channel, e.User, e.Client)
	})
	sub.OnLeave(func(e centrifuge.LeaveEvent) {
		log.Printf("Someone left %s: user id %s, client id %s", sub.Channel, e.User, e.Client)
	})

	err = sub.Subscribe()
	if err != nil {
		log.Fatalln(err)
	}

	c.lock.Lock()
	// todo: map key must be changed
	c.subscriptions[key] = sub
	c.lock.Unlock()

	return nil
}
