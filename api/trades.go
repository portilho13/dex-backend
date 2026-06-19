package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/portilho13/dex-backend/cache"
	"github.com/portilho13/dex-backend/geckoterminal"
)

type TradesHandler struct {
	gecko *geckoterminal.Client
	cache *cache.Cache
}

func NewTradesHandler(gecko *geckoterminal.Client, c *cache.Cache) *TradesHandler {
	return &TradesHandler{gecko: gecko, cache: c}
}

func (h *TradesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "missing address parameter", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("trades:%s", address)

	if cached, ok := h.cache.Get(key); ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}

	trades, err := h.gecko.GetTrades(address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	h.cache.Set(key, trades, 10*time.Second)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trades)
}
