package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/portilho13/dex-backend/api"
	"github.com/portilho13/dex-backend/cache"
	"github.com/portilho13/dex-backend/conn"
	"github.com/portilho13/dex-backend/geckoterminal"
	"github.com/portilho13/dex-backend/helius"
	"github.com/portilho13/dex-backend/price"
)

func main() {
	heliusKey := os.Getenv("HELIUS_API_KEY")
	if heliusKey == "" {
		log.Fatal("HELIUS_API_KEY is not set")
	}

	rpcClient := rpc.New("https://mainnet.helius-rpc.com/?api-key=" + heliusKey)
	geckoClient := geckoterminal.NewClient()
	solPrice := price.NewSolPrice()
	apiCache := cache.New()

	var manager *conn.PoolManager

	txSub := helius.NewTxSubscriber(heliusKey, func(poolAddress string, trade helius.Trade) {
		for _, pool := range manager.ActivePools() {
			msg := conn.OutgoingMessage{
				Type:      "trade",
				Pool:      pool,
				Kind:      trade.Kind,
				VolumeBase: trade.AmountOut,
				VolumeSOL: trade.AmountIn,
				TxHash:    trade.Signature,
				Maker:     trade.Maker,
				Timestamp: trade.Timestamp.UnixMilli(),
			}
			manager.Broadcast(pool, msg)
		}
	})

	manager = conn.NewPoolManager(rpcClient, solPrice, txSub)

	http.Handle("/ws", conn.NewWebSocketHandler(manager))
	http.Handle("/ohlcv", api.NewOHLCVHandler(geckoClient, apiCache))
	http.Handle("/pool-info", api.NewPoolInfoHandler(geckoClient, apiCache))
	http.Handle("/trades", api.NewTradesHandler(geckoClient, apiCache))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
