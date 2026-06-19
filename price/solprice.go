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

type coingeckoResponse struct {
	Solana struct {
		USD float64 `json:"usd"`
	} `json:"solana"`
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
	resp, err := sp.httpClient.Get("https://api.coingecko.com/api/v3/simple/price?ids=solana&vs_currencies=usd")
	if err != nil {
		log.Printf("solprice: fetch error: %v", err)
		return
	}
	defer resp.Body.Close()

	var result coingeckoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("solprice: decode error: %v", err)
		return
	}

	sp.mu.Lock()
	sp.price = result.Solana.USD
	sp.mu.Unlock()

}

func (sp *SolPrice) loop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		sp.fetch()
	}
}
