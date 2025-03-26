// Copyright 2021 Evmos Foundation
// This file is part of Evmos' Ethermint library.
//
// The Ethermint library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Ethermint library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Ethermint library. If not, see https://github.com/evmos/ethermint/blob/main/LICENSE
package tests

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/evmos/ethermint/crypto/ethsecp256k1"
)

// NewAddrKey generates an Ethereum address and its corresponding private key.
func NewAddrKey() (common.Address, *ethsecp256k1.PrivKey) {
	privkey, _ := ethsecp256k1.GenerateKey()
	key, err := privkey.ToECDSA()
	if err != nil {
		return common.Address{}, nil
	}

	addr := crypto.PubkeyToAddress(key.PublicKey)

	return addr, privkey
}

// NewAccAddressAndKey generates a private key and its corresponding
// Cosmos SDK address.
func NewAccAddressAndKey() (sdk.AccAddress, *ethsecp256k1.PrivKey) {
	addr, privKey := NewAddrKey()
	return sdk.AccAddress(addr.Bytes()), privKey
}

// GenerateAddress generates an Ethereum address.
func GenerateAddress() common.Address {
	addr, _ := NewAddrKey()
	return addr
}
