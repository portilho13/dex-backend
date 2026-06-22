package price

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type SolPrice struct {
	price      float64
	mu         sync.RWMutex
	httpClient *http.Client
}

type binanceResponse struct {
	Price string `json:"price"`
}

func NewSolPrice() *SolPrice {
	sp := &SolPrice{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
	sp.fetch()
	go sp.loop()
	return sp
}

func (sp *SolPrice) USD() float64 {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.price
}

func (sp *SolPrice) fetch() {
	resp, err := sp.httpClient.Get("https://api.binance.com/api/v3/ticker/price?symbol=SOLUSDT")
	if err != nil {
		log.Printf("solprice: fetch error: %v", err)
		return
	}
	defer resp.Body.Close()

	var result binanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("solprice: decode error: %v", err)
		return
	}

	var p float64
	fmt.Sscanf(result.Price, "%f", &p)
	if p > 0 {
		sp.mu.Lock()
		sp.price = p
		sp.mu.Unlock()
	}
}

func (sp *SolPrice) loop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		sp.fetch()
	}
}
