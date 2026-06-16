package dex

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/portilho13/dex-backend/constants"
	"github.com/portilho13/dex-backend/types"
)

func identifyDEX(owner solana.PublicKey) types.DEX {
	switch owner {
	case constants.RaydiumAMMV4:
		return types.RaydiumV4
	case constants.PumpFunProgramId:
		return types.PumpFunDEX
	case constants.OrcaWhirlpool:
		return types.Orca
	default:
		return types.Unknown
	}
}

func GetPoolInfo(ctx context.Context, poolAdress string, rpc *rpc.Client) error {

	poolAddressPubkey := solana.MustPublicKeyFromBase58(poolAdress)

	accountInfo, err := rpc.GetAccountInfo(ctx, poolAddressPubkey)
	if err != nil {
		return err
	}

	owner := accountInfo.Value.Owner

	dexType := identifyDEX(owner)

	fmt.Println(dexType)

	return nil
}

func getMintDecimals(ctx context.Context, rpc *rpc.Client, mint solana.PublicKey) (uint8, error) {
	info, err := rpc.GetAccountInfo(ctx, mint)
	if err != nil {
		return 0, err
	}

	data := info.Value.Data.GetBinary()
	if len(data) < 45 {
		return 0, fmt.Errorf("invalid mint account")
	}

	return data[44], nil
}

func isQuoteToken(mint solana.PublicKey) bool {
	return mint == constants.USDC || mint == constants.USDT || mint == constants.SOL
}

func getPrice(ctx context.Context, rpc *rpc.Client, pool types.PoolInfo) (float64, error) {
	baseRes, err := rpc.GetTokenAccountBalance(ctx, pool.BaseVault, "confirmed")
	if err != nil {
		return 0, err
	}

	quoteRes, err := rpc.GetTokenAccountBalance(ctx, pool.QuoteVault, "confirmed")
	if err != nil {
		return 0, err
	}

	baseAmount, _ := strconv.ParseFloat(baseRes.Value.UiAmountString, 64)
	quoteAmount, _ := strconv.ParseFloat(quoteRes.Value.UiAmountString, 64)

	if isQuoteToken(pool.BaseMint) {
		return (baseAmount / quoteAmount), nil
	} else {
		return (quoteAmount / baseAmount), nil
	}
}
