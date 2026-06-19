package dex

import (
	"github.com/portilho13/dex-backend/types"
	"github.com/portilho13/dex-backend/utils"
)

const (
	raydiumBaseMintOffset   = 400
	raydiumQuoteMintOffset  = 432
	raydiumBaseVaultOffset  = 336
	raydiumQuoteVaultOffset = 368
	raydiumBaseDecOffset    = 464
	raydiumQuoteDecOffset   = 465
	raydiumMinLen           = 752
)

func parseRaydiumV4(data []byte) (types.PoolInfo, error) {
	if len(data) < raydiumMinLen {
		return types.PoolInfo{}, nil
	}

	return types.PoolInfo{
		BaseMint:      utils.PubkeyAt(data, raydiumBaseMintOffset),
		QuoteMint:     utils.PubkeyAt(data, raydiumQuoteMintOffset),
		BaseVault:     utils.PubkeyAt(data, raydiumBaseVaultOffset),
		QuoteVault:    utils.PubkeyAt(data, raydiumQuoteVaultOffset),
		BaseDecimals:  data[raydiumBaseDecOffset],
		QuoteDecimals: data[raydiumQuoteDecOffset],
	}, nil
}
