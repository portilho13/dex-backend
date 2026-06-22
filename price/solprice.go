package price

import (
	"encoding/json"
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

type jupiterResponse struct {
	Data map[string]struct {
		Price float64 `json:"price"`
	} `json:"data"`
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
	resp, err := sp.httpClient.Get("https://api.jup.ag/price/v2?ids=So11111111111111111111111111111111111111112")
	if err != nil {
		log.Printf("solprice: jupiter error: %v", err)
		sp.fetchFallback()
		return
	}
	defer resp.Body.Close()

	var result jupiterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("solprice: jupiter decode error: %v", err)
		sp.fetchFallback()
		return
	}

	if sol, ok := result.Data["So11111111111111111111111111111111111111112"]; ok && sol.Price > 0 {
		sp.mu.Lock()
		sp.price = sol.Price
		sp.mu.Unlock()
		return
	}

	sp.fetchFallback()
}

type coingeckoResponse struct {
	Solana struct {
		USD float64 `json:"usd"`
	} `json:"solana"`
}

func (sp *SolPrice) fetchFallback() {
	resp, err := sp.httpClient.Get("https://api.coingecko.com/api/v3/simple/price?ids=solana&vs_currencies=usd")
	if err != nil {
		log.Printf("solprice: coingecko error: %v", err)
		return
	}
	defer resp.Body.Close()

	var result coingeckoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("solprice: coingecko decode error: %v", err)
		return
	}

	sp.mu.Lock()
	sp.price = result.Solana.USD
	sp.mu.Unlock()
}

func (sp *SolPrice) loop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		sp.fetch()
	}
}
