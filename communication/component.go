package communication

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type Connection interface {
	SendMessage(message []byte) error
}

type Service interface {
	HandleMessage(ctx context.Context, connectionID string, message Message) error
	HandleDisconnect(ctx context.Context, connectionID string)
	NewConnection(connection Connection) (string, error)
	SendMessageToConnection(connectionID string, message []byte) error

	AddListener(listener Listener)
}

type Listener interface {
	OnMessage(ctx context.Context, connectionID string, message Message)
	OnDisconnect(ctx context.Context, connectionID string)
}

type service struct {
	connectionStore cmap.ConcurrentMap[string, Connection]
	listeners       cmap.ConcurrentMap[string, Listener]
}

var _ Service = (*service)(nil)

func NewCommunicationService() *service {
	return &service{
		connectionStore: cmap.New[Connection](),
		listeners:       cmap.New[Listener](),
	}
}

func (s *service) NewConnection(connection Connection) (string, error) {
	connectionID := uuid.NewString()

	s.connectionStore.Set(connectionID, connection)

	return connectionID, nil
}

func (s *service) HandleMessage(ctx context.Context, connectionID string, message Message) error {
	_, ok := s.connectionStore.Get(connectionID)
	if !ok {
		return fmt.Errorf("connection not found")
	}

	s.BroadcastMessageToAllListeners(ctx, connectionID, message)

	return nil
}

func (s *service) HandleDisconnect(ctx context.Context, connectionID string) {
	s.connectionStore.Pop(connectionID)
	s.BroadcastDisconnectToAllListeners(ctx, connectionID)
}

func (s *service) SendMessageToConnection(connectionID string, message []byte) error {
	connection, ok := s.connectionStore.Get(connectionID)
	if !ok {
		return fmt.Errorf("connection not found")
	}

	return connection.SendMessage(message)
}

func (s *service) AddListener(listener Listener) {
	s.listeners.Set(uuid.NewString(), listener)
}

func (s *service) BroadcastMessageToAllListeners(ctx context.Context, connectionID string, message Message) {
	s.listeners.IterCb(func(_ string, listener Listener) {
		listener.OnMessage(ctx, connectionID, message)
	})
}

func (s *service) BroadcastDisconnectToAllListeners(ctx context.Context, connectionID string) {
	s.listeners.IterCb(func(_ string, listener Listener) {
		listener.OnDisconnect(ctx, connectionID)
	})
}
