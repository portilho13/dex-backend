package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/portilho13/dex-backend/dex"
)

func main() {
	fmt.Println("Test")

	client := rpc.New("https://mainnet.helius-rpc.com/?api-key=7e293735-eb88-4947-88a2-2c28ce5e1edd")

	address := "Fhr8U71eGAwaUvNQdTysPjoBq7wMZGJ4Ej6yDEQDxs7D"

	err := dex.GetPoolInfo(context.TODO(), address, client)
	if err != nil {
		log.Fatal(err)
	}
}
