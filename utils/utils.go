package utils

import "github.com/gagliardetto/solana-go"

func PubkeyAt(data []byte, offset int) solana.PublicKey {
	var pk solana.PublicKey
	copy(pk[:], data[offset:offset+32])
	return pk
}
