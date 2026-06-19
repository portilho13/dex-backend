package conn

import (
	"context"
	"fmt"
	"sync"
)

type PoolSubscription struct {
	clients map[*Client]struct{}
	mu      sync.Mutex
	cancel  context.CancelFunc
}

func newPoolSubscription(cancel context.CancelFunc) *PoolSubscription {
	return &PoolSubscription{
		clients: make(map[*Client]struct{}),
		cancel:  cancel,
	}
}

func (ps *PoolSubscription) AddClient(client *Client) error {
	if client == nil {
		return fmt.Errorf("client is nil")
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, clientExists := ps.clients[client]; clientExists {
		return fmt.Errorf("client already exists")
	}

	ps.clients[client] = struct{}{}

	return nil
}

func (ps *PoolSubscription) RemoveClient(client *Client) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.clients, client)
}

func (ps *PoolSubscription) ClientCount() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ps.clients)
}
