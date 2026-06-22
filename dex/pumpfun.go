package dex

import (
	"encoding/binary"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/portilho13/dex-backend/constants"
	"github.com/portilho13/dex-backend/types"
	"github.com/portilho13/dex-backend/utils"
)

var (
	pumpFunBondingCurveDiscriminator        = uint64(6966180631402821399)
	pumpFunAMMDiscriminator          uint64 = 13577703138238765809

	pumpCurveSeed              = []byte("bonding-curve")
	pumpFunSOLMint             = solana.SolMint
	pumpFunSOLDecimals   uint8 = 9
	pumpFunTokenDecimals uint8 = 6
)

const (
	pumpAMMBaseMintOffset   = 43
	pumpAMMQuoteMintOffset  = 75
	pumpAMMBaseVaultOffset  = 139
	pumpAMMQuoteVaultOffset = 171
	pumpAMMMinLen           = 203

	pumpBondingCurveMinLen = 49
)

func parsePumpFun(data []byte, extra ...solana.PublicKey) (types.PoolInfo, error) {
	if len(data) < 8 {
		return types.PoolInfo{}, fmt.Errorf("pumpfun: data too short")
	}

	discriminator := binary.LittleEndian.Uint64(data[0:8])

	switch discriminator {
	case pumpFunBondingCurveDiscriminator:
		return parsePumpFunBondingCurve(data, extra...)
	case pumpFunAMMDiscriminator:
		return parsePumpFunAMM(data)
	default:
		return types.PoolInfo{}, fmt.Errorf("pumpfun: unknown discriminator %d", discriminator)
	}
}

func parsePumpFunBondingCurve(data []byte, extra ...solana.PublicKey) (types.PoolInfo, error) {
	if len(data) < pumpBondingCurveMinLen {
		return types.PoolInfo{}, fmt.Errorf("pumpfun bonding curve: data too short (%d < %d)", len(data), pumpBondingCurveMinLen)
	}
	if len(extra) < 2 {
		return types.PoolInfo{}, fmt.Errorf("pumpfun bonding curve: requires extra[0]=mint, extra[1]=poolAddress")
	}

	mint := extra[0]
	poolAddress := extra[1]

	tokenVault, _, err := solana.FindAssociatedTokenAddress(poolAddress, mint)
	if err != nil {
		return types.PoolInfo{}, fmt.Errorf("pumpfun bonding curve: derive token vault: %w", err)
	}

	return types.PoolInfo{
		BaseMint:      mint,
		QuoteMint:     constants.SOL,
		BaseVault:     tokenVault,
		QuoteVault:    poolAddress,
		BaseDecimals:  pumpFunTokenDecimals,
		QuoteDecimals: pumpFunSOLDecimals,
	}, nil
}

func parsePumpFunAMM(data []byte) (types.PoolInfo, error) {
	if len(data) < pumpAMMMinLen {
		return types.PoolInfo{}, fmt.Errorf("pumpfun amm: data too short (%d < %d)", len(data), pumpAMMMinLen)
	}

	return types.PoolInfo{
		BaseMint:      utils.PubkeyAt(data, pumpAMMBaseMintOffset),
		QuoteMint:     utils.PubkeyAt(data, pumpAMMQuoteMintOffset),
		BaseVault:     utils.PubkeyAt(data, pumpAMMBaseVaultOffset),
		QuoteVault:    utils.PubkeyAt(data, pumpAMMQuoteVaultOffset),
		BaseDecimals:  pumpFunTokenDecimals,
		QuoteDecimals: pumpFunSOLDecimals,
	}, nil
}
