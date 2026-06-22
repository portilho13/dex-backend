package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/portilho13/dex-backend/cache"
	"github.com/portilho13/dex-backend/geckoterminal"
)

type OHLCVHandler struct {
	gecko *geckoterminal.Client
	cache *cache.Cache
}

func NewOHLCVHandler(gecko *geckoterminal.Client, c *cache.Cache) *OHLCVHandler {
	return &OHLCVHandler{gecko: gecko, cache: c}
}

func (h *OHLCVHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "missing address parameter", http.StatusBadRequest)
		return
	}

	aggregate := r.URL.Query().Get("aggregate")
	if aggregate == "" {
		aggregate = "15"
	}

	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "minute"
	}

	key := fmt.Sprintf("ohlcv:%s:%s:%s", address, aggregate, timeframe)

	if cached, ok := h.cache.Get(key); ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}

	candles, err := h.gecko.GetOHLCV(address, aggregate, timeframe)
	if err != nil {
		writeGeckoError(w, err)
		return
	}

	h.cache.Set(key, candles, 15*time.Second)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(candles)
}
