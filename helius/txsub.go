package helius

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type TxCallback func(poolAddress string, trade Trade)

type Trade struct {
	Signature string
	Kind      string
	Maker     string
	AmountIn  float64
	AmountOut float64
	Timestamp time.Time
}

type TxSubscriber struct {
	apiKey  string
	conn    *websocket.Conn
	mu      sync.Mutex
	subs    map[string]int
	subToPool map[int]string
	nextID  int
	onTrade TxCallback
}

func NewTxSubscriber(apiKey string, onTrade TxCallback) *TxSubscriber {
	ts := &TxSubscriber{
		apiKey:    apiKey,
		subs:      make(map[string]int),
		subToPool: make(map[int]string),
		nextID:    1,
		onTrade:   onTrade,
	}
	go ts.connectLoop()
	return ts
}

func (ts *TxSubscriber) connectLoop() {
	for {
		err := ts.connect()
		if err != nil {
			log.Printf("helius ws: connect error: %v", err)
		}
		time.Sleep(5 * time.Second)
	}
}

func (ts *TxSubscriber) connect() error {
	url := fmt.Sprintf("wss://mainnet.helius-rpc.com/?api-key=%s", ts.apiKey)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}

	ts.mu.Lock()
	ts.conn = conn
	pools := make([]string, 0, len(ts.subs))
	for pool := range ts.subs {
		pools = append(pools, pool)
	}
	ts.subs = make(map[string]int)
	ts.subToPool = make(map[int]string)
	ts.mu.Unlock()

	for _, pool := range pools {
		ts.sendSubscribe(pool)
	}

	log.Printf("helius ws: connected, subscribing to %d pools", len(pools))

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("helius ws: read error: %v", err)
			conn.Close()
			return err
		}

		ts.handleMessage(data)
	}
}

func (ts *TxSubscriber) Subscribe(poolAddress string) {
	ts.mu.Lock()
	if _, exists := ts.subs[poolAddress]; exists {
		ts.mu.Unlock()
		return
	}
	ts.subs[poolAddress] = 0
	ts.mu.Unlock()

	ts.sendSubscribe(poolAddress)
}

func (ts *TxSubscriber) Unsubscribe(poolAddress string) {
	ts.mu.Lock()
	subID, exists := ts.subs[poolAddress]
	if !exists {
		ts.mu.Unlock()
		return
	}
	delete(ts.subs, poolAddress)
	delete(ts.subToPool, subID)
	conn := ts.conn
	ts.mu.Unlock()

	if conn != nil && subID > 0 {
		unsubMsg := map[string]any{
			"jsonrpc": "2.0",
			"id":      ts.nextID,
			"method":  "logsUnsubscribe",
			"params":  []any{subID},
		}
		data, _ := json.Marshal(unsubMsg)
		conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (ts *TxSubscriber) sendSubscribe(poolAddress string) {
	ts.mu.Lock()
	conn := ts.conn
	id := ts.nextID
	ts.nextID++
	ts.mu.Unlock()

	if conn == nil {
		return
	}

	msg := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "logsSubscribe",
		"params": []any{
			map[string]any{
				"mentions": []string{poolAddress},
			},
			map[string]any{
				"commitment": "confirmed",
			},
		},
	}

	data, _ := json.Marshal(msg)
	ts.mu.Lock()
	ts.subToPool[id] = poolAddress
	ts.mu.Unlock()

	conn.WriteMessage(websocket.TextMessage, data)
}

type rpcResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type subConfirm struct {
	Result int `json:"result"`
}

type logNotification struct {
	Result struct {
		Value struct {
			Signature string   `json:"signature"`
			Err       any      `json:"err"`
			Logs      []string `json:"logs"`
		} `json:"value"`
	} `json:"result"`
	Subscription int `json:"subscription"`
}

func (ts *TxSubscriber) handleMessage(data []byte) {
	var resp rpcResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return
	}

	if resp.Method == "logsNotification" {
		var notif logNotification
		if err := json.Unmarshal(resp.Params, &notif); err != nil {
			return
		}

		if notif.Result.Value.Err != nil {
			return
		}

		sig := notif.Result.Value.Signature
		if sig == "" {
			return
		}

		ts.mu.Lock()
		var poolAddress string
		for pool, subID := range ts.subs {
			if subID == notif.Subscription {
				poolAddress = pool
				break
			}
		}
		ts.mu.Unlock()

		if poolAddress == "" {
			return
		}

		kind := "buy"
		for _, logLine := range notif.Result.Value.Logs {
			if contains(logLine, "Sell") || contains(logLine, "sell") {
				kind = "sell"
				break
			}
		}

		trade := Trade{
			Signature: sig,
			Kind:      kind,
			Timestamp: time.Now(),
		}

		ts.onTrade(poolAddress, trade)
		return
	}

	if resp.ID > 0 && resp.Result != nil {
		var subID int
		if err := json.Unmarshal(resp.Result, &subID); err == nil && subID > 0 {
			ts.mu.Lock()
			if pool, ok := ts.subToPool[resp.ID]; ok {
				ts.subs[pool] = subID
				delete(ts.subToPool, resp.ID)
			}
			ts.mu.Unlock()
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
