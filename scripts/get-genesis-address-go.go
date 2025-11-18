package main

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/go-bip39"
)

func main() {
	// Test mnemonic
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	
	// Generate seed from mnemonic
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		panic(err)
	}
	
	// Create master key and derive account
	masterPriv, ch := hd.ComputeMastersFromSeed(seed)
	derivedPriv, err := hd.DerivePrivateKeyForPath(masterPriv, ch, sdk.GetConfig().GetFullBIP44Path())
	if err != nil {
		panic(err)
	}
	
	// Create private key
	privKey := &secp256k1.PrivKey{Key: derivedPriv}
	
	// Get address with volnix prefix
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("volnix", "volnixpub")
	address := sdk.AccAddress(privKey.PubKey().Address()).String()
	
	fmt.Println("✅ Genesis аккаунт адрес:")
	fmt.Println(address)
	fmt.Println("")
	fmt.Println("💡 Мнемоника для подключения:")
	fmt.Println(mnemonic)
}
