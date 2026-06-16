package dex

import (
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/portilho13/dex-backend/constants"
	"github.com/portilho13/dex-backend/types"
)

var (
	pumpFunDiscriminator       = 8
	pumpFunMinLen              = 49
	pumpCurveSeed              = []byte("bonding-curve")
	pumpFunSOLMint             = solana.SolMint
	pumpFunSOLDecimals   uint8 = 9
	pumpFunTokenDecimals uint8 = 6
)

func parsePumpFun(data []byte, extra ...solana.PublicKey) (types.PoolInfo, error) {
	if len(data) < pumpFunMinLen {
		return types.PoolInfo{}, fmt.Errorf("pumpfun: data too short (%d < %d)", len(data), pumpFunMinLen)
	}

	if len(extra) == 0 {
		return types.PoolInfo{}, fmt.Errorf("pumpfun: token mint required as extra[0]")
	}
	mint := extra[0]

	bondingCurvePDA, _, err := solana.FindProgramAddress(
		[][]byte{pumpCurveSeed, mint.Bytes()},
		constants.PumpFunProgramId,
	)
	if err != nil {
		return types.PoolInfo{}, fmt.Errorf("pumpfun: derive bonding curve PDA: %w", err)
	}

	tokenVault, _, err := solana.FindAssociatedTokenAddress(bondingCurvePDA, mint)
	if err != nil {
		return types.PoolInfo{}, fmt.Errorf("pumpfun: derive token vault: %w", err)
	}

	return types.PoolInfo{
		BaseMint:      mint,
		QuoteMint:     constants.SOL,
		BaseVault:     tokenVault,
		QuoteVault:    bondingCurvePDA,
		BaseDecimals:  pumpFunTokenDecimals,
		QuoteDecimals: pumpFunSOLDecimals,
	}, nil
}
