package types

import "github.com/gagliardetto/solana-go"

type PoolInfo struct {
	BaseMint      solana.PublicKey
	QuoteMint     solana.PublicKey
	BaseVault     solana.PublicKey
	QuoteVault    solana.PublicKey
	BaseDecimals  uint8
	QuoteDecimals uint8
}

type DEX int

const (
	Unknown DEX = iota
	RaydiumV4
	PumpFunDEX
	Orca
)
