package connectionPool

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/h-varmazyar/gatec/configs"
	"github.com/sirupsen/logrus"
)

type WebSocketPool struct {
	cfg                     *configs.SocketConfig
	hasCapacityConnections  []*socketClient
	fullConnections         map[string]*socketClient
	reservedConnectionCount int
	lock                    *sync.Mutex
}

func NewWebSocketPool(cfg *configs.SocketConfig) (*WebSocketPool, error) {
	pool := &WebSocketPool{
		hasCapacityConnections:  make([]*socketClient, 0),
		fullConnections:         make(map[string]*socketClient),
		cfg:                     cfg,
		reservedConnectionCount: 0,
		lock:                    new(sync.Mutex),
	}

	return pool, nil
}

func (p *WebSocketPool) RegisterChannel(key, channelName string, responseChannel chan AsyncResponse, withCompression bool) error {
	logrus.Infof("registering new channel: %v", channelName)
	if !p.hasCapacity() {
		return errors.New("max connection exceeded!")
	}

	var connection *socketClient
	var err error
	if len(p.hasCapacityConnections) == 0 {
		logrus.Infof("making new connection")
		connection, err = NewClient(p.cfg)
		if err != nil {
			logrus.WithError(err).Errorf("failed to create new client")
			return err
		}
	} else {
		p.lock.Lock()
		connection = p.hasCapacityConnections[0]
		if len(p.hasCapacityConnections) > 1 {
			p.hasCapacityConnections = p.hasCapacityConnections[1:]
		} else {
			p.hasCapacityConnections = []*socketClient{}
		}
		p.reservedConnectionCount++
		p.lock.Unlock()
	}

	if err = connection.RegisterChannel(key, channelName, responseChannel, withCompression); err != nil {
		logrus.WithError(err).Errorf("failed to register new channel")
		return err
	}

	p.lock.Lock()
	defer p.lock.Unlock()
	if connection.hasCapacity() {
		p.hasCapacityConnections = append(p.hasCapacityConnections, connection)
	} else {
		p.fullConnections[uuid.New().String()] = connection
	}
	p.reservedConnectionCount--
	if p.reservedConnectionCount < 0 {
		p.reservedConnectionCount = 0
	}

	return nil
}

func (p *WebSocketPool) Shutdown() {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, connection := range p.hasCapacityConnections {
		connection.Close()
	}
	for _, connection := range p.fullConnections {
		connection.Close()
	}

	p.hasCapacityConnections = []*socketClient{}
	p.fullConnections = make(map[string]*socketClient)
	p.reservedConnectionCount = 0
}

func (p *WebSocketPool) ReconnectChannel(key, channelName string, responseChannel chan AsyncResponse, withCompression bool) error {
	fmt.Println("reconnecting channel")
	p.lock.Lock()
	connections := make([]*socketClient, 0, len(p.hasCapacityConnections)+len(p.fullConnections))
	connections = append(connections, p.hasCapacityConnections...)
	for _, connection := range p.fullConnections {
		connections = append(connections, connection)
	}
	p.lock.Unlock()

	for _, connection := range connections {
		if connection.HasSubscription(key) {
			connection.UnregisterChannel(key)
			break
		}
	}

	return p.RegisterChannel(key, channelName, responseChannel, withCompression)
}

func (p *WebSocketPool) hasCapacity() bool {
	return len(p.fullConnections)+len(p.hasCapacityConnections)+p.reservedConnectionCount < p.cfg.MaxConnection
}
