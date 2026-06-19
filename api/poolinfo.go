package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/portilho13/dex-backend/cache"
	"github.com/portilho13/dex-backend/geckoterminal"
)

type PoolInfoHandler struct {
	gecko *geckoterminal.Client
	cache *cache.Cache
}

func NewPoolInfoHandler(gecko *geckoterminal.Client, c *cache.Cache) *PoolInfoHandler {
	return &PoolInfoHandler{gecko: gecko, cache: c}
}

func (h *PoolInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "missing address parameter", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("pool-info:%s", address)

	if cached, ok := h.cache.Get(key); ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}

	details, err := h.gecko.GetPoolDetails(address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	h.cache.Set(key, details, 60*time.Second)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(details)
}
