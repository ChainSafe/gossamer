package main

import (
	"errors"
	"fmt"
	"github.com/ChainSafe/gossamer/lib/keystore"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
)

/*
	Some Notes
- only valid transactions initially
- initially just worry about transfer events (start small and iterate)
- think I should have 2 transaction sets (to counteract each other)

	Steps for transaction testing
1) retrieve initial state (for now just node balances)
2) execute transactions
3) calculate the expected state
4) retrieve new state
5) new state compared to expected state
*/

/* Account balances info for reference
On Polkadot, four different balance types indicate whether your balance can be used for transfers, to pay fees, or must remain frozen and unused due to an on-chain requirement.

The AccountData struct defines the balance types in Substrate. The four types of balances include free, reserved, misc_frozen (miscFrozen in camel-case), and fee_frozen (feeFrozen in camel-case).

In general, the usable balance of the account is the amount that is free minus any funds that are considered frozen (either misc_frozen or fee_frozen) and depend on the reason for which the
funds are to be used. If the funds are to be used for transfers, then the usable amount is the free amount minus any misc_frozen funds. However, if the funds are to be used to pay transaction fees,
the usable amount would be the free funds minus fee_frozen.

The total balance of the account is considered to be the sum of free and reserved funds in the account. Reserved funds are held due to on-chain requirements and can usually be freed by
taking some on-chain action. For example, the "Identity" pallet reserves funds while an on-chain identity is registered, but by clearing the identity, you can unreserve the funds and make them free again.
*/

func getAccountInfo(api *gsrpc.SubstrateAPI, key types.StorageKey) (types.AccountInfo, error) {
	var accInfo types.AccountInfo
	ok, err := api.RPC.State.GetStorageLatest(key, &accInfo)
	if !ok {
		return types.AccountInfo{}, errors.New("unable to get storageLatest: value is empty")
	}
	if err != nil {
		return types.AccountInfo{}, err
	}
	return accInfo, err
}

func main() {
	// TODO CLI flag for address?
	api, err := gsrpc.NewSubstrateAPI("ws://127.0.0.1:8546")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Connected to gossamer API")

	// Try to get account info
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		fmt.Println(err)
	}

	aliceKey, err := types.CreateStorageKey(meta, "System", "Account", signature.TestKeyringPairAlice.PublicKey, nil)
	if err != nil {
		fmt.Println(err)
	}

	accountInfo, err := getAccountInfo(api, aliceKey)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(accountInfo)

	// Try to transfer money to Bob

	// Create a call, transferring 12345 units to Bob
	keyring, err := keystore.NewSr25519Keyring()
	if err != nil {
		panic(err)
	}
	bobPub := keyring.Bob().Public().Hex()

	bob, err := types.NewMultiAddressFromHexAccountID(bobPub)
	if err != nil {
		panic(err)
	}

	amount := types.NewUCompactFromUInt(12345)

	const balanceTransfer = "Balances.transfer"
	c, err := types.NewCall(meta, balanceTransfer, bob, amount)
	if err != nil {
		panic(err)
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		panic(err)
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		panic(err)
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		panic(err)
	}

	nonce := uint32(accountInfo.Nonce)

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	fmt.Printf("Sending %v from %#x to %#x with nonce %v\n", amount, signature.TestKeyringPairAlice.PublicKey, bob.AsID, nonce)
	// Sign the transaction using Alice's default account
	err = ext.Sign(signature.TestKeyringPairAlice, o)
	if err != nil {
		panic(err)
	}

	fmt.Println("Signed!")

	// Send the extrinsic
	//hash, err := api.RPC.Author.SubmitExtrinsic(ext)
	//if err != nil {
	//	panic(err)
	//}
	//
	//fmt.Printf("Transfer sent with hash %#x\n", hash)

	//// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		panic(err)
	}
	defer sub.Unsubscribe()

	for {
		status := <-sub.Chan()
		fmt.Printf("Transaction status: %#v\n", status)

		if status.IsInBlock {
			fmt.Printf("Completed at block hash: %#x\n", status.AsInBlock)
			return
		}
	}

}
