package conn

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
	send    chan PriceTick
	conn    *websocket.Conn
	manager *PoolManager
	pools   map[string]struct{}
}

func NewClient(conn *websocket.Conn, manager *PoolManager) *Client {
	return &Client{
		send:    make(chan PriceTick, 64),
		conn:    conn,
		manager: manager,
		pools:   make(map[string]struct{}),
	}
}

func (c *Client) ReadLoop() {
	defer c.cleanup()

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var msg IncomingMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg.Action {
		case "subscribe":
			if msg.Pool == "" {
				continue
			}
			if _, already := c.pools[msg.Pool]; already {
				continue
			}
			c.pools[msg.Pool] = struct{}{}
			c.manager.Subscribe(msg.Pool, c)

		case "unsubscribe":
			if _, subscribed := c.pools[msg.Pool]; !subscribed {
				continue
			}
			delete(c.pools, msg.Pool)
			c.manager.Unsubscribe(msg.Pool, c)
		}
	}
}

func (c *Client) WriteLoop() {
	for tick := range c.send {
		msg := OutgoingMessage{
			Type:      "price",
			Pool:      tick.Pool,
			Price:     tick.Price,
			Timestamp: tick.Timestamp.UnixMilli(),
		}

		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("write: marshal error: %v", err)
			continue
		}

		if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return
		}
	}
}

func (c *Client) cleanup() {
	for pool := range c.pools {
		c.manager.Unsubscribe(pool, c)
	}
	close(c.send)
	c.conn.Close()
}
