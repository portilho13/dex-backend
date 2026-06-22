package api

import (
	"errors"
	"net/http"

	"github.com/portilho13/dex-backend/geckoterminal"
)

func writeGeckoError(w http.ResponseWriter, err error) {
	var apiErr *geckoterminal.APIError
	if errors.As(err, &apiErr) {
		status := apiErr.Status
		if status == 429 {
			w.Header().Set("Retry-After", "2")
			http.Error(w, "rate limited, try again", http.StatusTooManyRequests)
			return
		}
		if status == 404 {
			http.Error(w, "pool not found on GeckoTerminal", http.StatusNotFound)
			return
		}
		http.Error(w, apiErr.Message, http.StatusBadGateway)
		return
	}
	http.Error(w, err.Error(), http.StatusBadGateway)
}
