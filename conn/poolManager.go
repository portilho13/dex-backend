package conn

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/portilho13/dex-backend/constants"
	"github.com/portilho13/dex-backend/dex"
	"github.com/portilho13/dex-backend/price"
)

type TxSubscriber interface {
	Subscribe(poolAddress string)
	Unsubscribe(poolAddress string)
}

type PoolManager struct {
	rpc      *rpc.Client
	solPrice *price.SolPrice
	txSub    TxSubscriber
	pools    map[string]*PoolSubscription
	mu       sync.RWMutex
}

func NewPoolManager(rpc *rpc.Client, solPrice *price.SolPrice, txSub TxSubscriber) *PoolManager {
	return &PoolManager{
		rpc:      rpc,
		solPrice: solPrice,
		txSub:    txSub,
		pools:    make(map[string]*PoolSubscription),
	}
}

func (pm *PoolManager) Subscribe(poolAddress string, client *Client) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.pools[poolAddress]; !exists {
		ctx, cancel := context.WithCancel(context.Background())
		ps := newPoolSubscription(cancel)
		pm.pools[poolAddress] = ps
		go pm.poll(ctx, poolAddress)
		if pm.txSub != nil {
			pm.txSub.Subscribe(poolAddress)
		}
	}

	pm.pools[poolAddress].AddClient(client)
}

func (pm *PoolManager) Unsubscribe(poolAddress string, client *Client) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	ps, exists := pm.pools[poolAddress]
	if !exists {
		return
	}

	ps.RemoveClient(client)

	if ps.ClientCount() == 0 {
		ps.cancel()
		delete(pm.pools, poolAddress)
		if pm.txSub != nil {
			pm.txSub.Unsubscribe(poolAddress)
		}
	}
}

func (pm *PoolManager) Broadcast(poolAddress string, msg OutgoingMessage) {
	pm.mu.RLock()
	ps, exists := pm.pools[poolAddress]
	pm.mu.RUnlock()

	if !exists {
		return
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	for client := range ps.clients {
		select {
		case client.send <- msg:
		default:
		}
	}
}

func (pm *PoolManager) ActivePools() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	pools := make([]string, 0, len(pm.pools))
	for addr := range pm.pools {
		pools = append(pools, addr)
	}
	return pools
}

func isSOLQuote(poolInfo dex.PoolResult) bool {
	baseMint := poolInfo.Info.BaseMint
	quoteMint := poolInfo.Info.QuoteMint

	if quoteMint == constants.SOL || quoteMint == solana.SolMint {
		return true
	}
	if baseMint == constants.SOL || baseMint == solana.SolMint {
		return true
	}
	return false
}

func (pm *PoolManager) poll(ctx context.Context, poolAddress string) {
	poolInfo, err := dex.GetPoolInfo(ctx, poolAddress, pm.rpc)
	if err != nil {
		log.Printf("poll: failed to get pool info for %s: %v", poolAddress, err)
		return
	}

	solQuote := isSOLQuote(poolInfo)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tokenPrice, err := dex.GetTokenPrice(ctx, pm.rpc, poolInfo.Info)
			if err != nil {
				log.Printf("poll: failed to get price for %s: %v", poolAddress, err)
				continue
			}

			if solQuote {
				solUSD := pm.solPrice.USD()
				if solUSD <= 0 {
					continue
				}
				tokenPrice = tokenPrice * solUSD
			}

			msg := OutgoingMessage{
				Type:      "price",
				Pool:      poolAddress,
				Price:     tokenPrice,
				Timestamp: time.Now().UnixMilli(),
			}

			pm.Broadcast(poolAddress, msg)
		}
	}
}
