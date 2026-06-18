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
	case constants.PumpFunProgramId: // bonding curve program
		return types.PumpFunDEX
	case constants.PumpFunAMMProgramId: // AMM program
		return types.PumpFunDEX
	case constants.OrcaWhirlpool:
		return types.Orca
	default:
		return types.Unknown
	}
}

func GetPoolInfo(ctx context.Context, poolAddress string, client *rpc.Client) (types.PoolInfo, error) {
	poolAddressPubkey := solana.MustPublicKeyFromBase58(poolAddress)

	accountInfo, err := client.GetAccountInfo(ctx, poolAddressPubkey)
	if err != nil {
		return types.PoolInfo{}, err
	}
	if accountInfo == nil || accountInfo.Value == nil {
		return types.PoolInfo{}, fmt.Errorf("account not found")
	}

	data := accountInfo.Value.Data.GetBinary()
	owner := accountInfo.Value.Owner

	fmt.Println(owner)

	dexType := identifyDEX(owner)

	switch dexType {
	case types.RaydiumV4:
		return ParsePoolInfo(types.RaydiumV4, data)

	case types.Orca:
		return ParsePoolInfo(types.Orca, data)

	case types.PumpFunDEX:
		// parsePumpFun detects bonding curve vs AMM via discriminator.
		// For bonding curve: pass mint + poolAddress as extras.
		// For AMM: extras are ignored, everything is read from data.
		return ParsePoolInfo(types.PumpFunDEX, data, poolAddressPubkey)

	default:
		return types.PoolInfo{}, fmt.Errorf("unknown DEX for owner %s", owner)
	}
}

func ParsePoolInfo(dex types.DEX, data []byte, extra ...solana.PublicKey) (types.PoolInfo, error) {
	switch dex {
	case types.PumpFunDEX:
		return parsePumpFun(data, extra...)
	default:
		return types.PoolInfo{}, fmt.Errorf("unknown DEX: %d", dex)
	}
}

func getMintDecimals(ctx context.Context, client *rpc.Client, mint solana.PublicKey) (uint8, error) {
	info, err := client.GetAccountInfo(ctx, mint)
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

func GetTokenPrice(ctx context.Context, client *rpc.Client, pool types.PoolInfo) (float64, error) {
	baseRes, err := client.GetTokenAccountBalance(ctx, pool.BaseVault, rpc.CommitmentConfirmed)
	if err != nil {
		return 0, err
	}

	quoteRes, err := client.GetTokenAccountBalance(ctx, pool.QuoteVault, rpc.CommitmentConfirmed)
	if err != nil {
		return 0, err
	}

	baseAmount, err := strconv.ParseFloat(baseRes.Value.UiAmountString, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid base vault amount: %w", err)
	}
	quoteAmount, err := strconv.ParseFloat(quoteRes.Value.UiAmountString, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid quote vault amount: %w", err)
	}

	if baseAmount == 0 || quoteAmount == 0 {
		return 0, fmt.Errorf("vault balance is zero")
	}

	if isQuoteToken(pool.BaseMint) {
		return baseAmount / quoteAmount, nil
	}
	return quoteAmount / baseAmount, nil
}
