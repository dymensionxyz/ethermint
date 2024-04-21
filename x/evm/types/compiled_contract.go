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
package types

import (
	// embed compiled smart contract
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// HexString is a byte array that serializes to hex
type HexString []byte

// MarshalJSON serializes ByteArray to hex
func (s HexString) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%x", string(s)))
}

// UnmarshalJSON deserializes ByteArray to hex
func (s *HexString) UnmarshalJSON(data []byte) error {
	var x string
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	str, err := hex.DecodeString(x)
	if err != nil {
		return err
	}
	*s = str
	return nil
}

// CompiledContract contains compiled bytecode and abi
type CompiledContract struct {
	ABI abi.ABI
	Bin HexString
}

type jsonCompiledContract struct {
	ABI string
	Bin HexString
}

// MarshalJSON serializes ByteArray to hex
func (s CompiledContract) MarshalJSON() ([]byte, error) {
	abi1, err := json.Marshal(s.ABI)
	if err != nil {
		return nil, err
	}
	return json.Marshal(jsonCompiledContract{ABI: string(abi1), Bin: s.Bin})
}

// UnmarshalJSON deserializes ByteArray to hex
func (s *CompiledContract) UnmarshalJSON(data []byte) error {
	var x jsonCompiledContract
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}

	s.Bin = x.Bin
	if err := json.Unmarshal([]byte(x.ABI), &s.ABI); err != nil {
		return fmt.Errorf("failed to unmarshal ABI: %w", err)
	}

	return nil
}

var (
	//go:embed ERC20Contract.json
	erc20JSON []byte

	// ERC20Contract is the compiled test erc20 contract
	ERC20Contract CompiledContract

	//go:embed SimpleStorageContract.json
	simpleStorageJSON []byte

	// SimpleStorageContract is the compiled test simple storage contract
	SimpleStorageContract CompiledContract

	//go:embed TestMessageCall.json
	testMessageCallJSON []byte

	// TestMessageCall is the compiled message call benchmark contract
	TestMessageCall CompiledContract

	//go:embed VFBankContract20.json
	vfBankContract20JSON []byte

	// VFBankContract20 is the compiled virtual frontier bank contract
	VFBankContract20 CompiledContract
)

func init() {
	err := json.Unmarshal(erc20JSON, &ERC20Contract)
	if err != nil {
		panic(err)
	}

	if len(ERC20Contract.Bin) == 0 {
		panic("load contract failed")
	}

	err = json.Unmarshal(testMessageCallJSON, &TestMessageCall)
	if err != nil {
		panic(err)
	}

	if len(TestMessageCall.Bin) == 0 {
		panic("load contract failed")
	}

	err = json.Unmarshal(simpleStorageJSON, &SimpleStorageContract)
	if err != nil {
		panic(err)
	}

	if len(SimpleStorageContract.Bin) == 0 {
		panic("load contract failed")
	}

	err = json.Unmarshal(vfBankContract20JSON, &VFBankContract20)
	if err != nil {
		panic(err)
	}

	if len(VFBankContract20.Bin) == 0 {
		panic("load contract failed")
	}
}

func (s CompiledContract) PackOutput(methodName string, args ...interface{}) ([]byte, error) {
	method, exist := s.ABI.Methods[methodName]
	if !exist {
		return nil, sdkerrors.ErrPackAny.Wrapf("method '%s' not found in ABI", methodName)
	}

	bz, err := method.Outputs.Pack(args...)
	if err != nil {
		return nil, sdkerrors.ErrPackAny.Wrapf("failed to pack output for method %s()", methodName)
	}

	return bz, nil
}

func (s CompiledContract) UnpackInput(methodName string, data []byte) ([]interface{}, error) {
	method, exist := s.ABI.Methods[methodName]
	if !exist {
		return nil, sdkerrors.ErrUnpackAny.Wrapf("method '%s' not found in ABI", methodName)
	}

	inputs, err := method.Inputs.Unpack(data)
	if err != nil {
		return nil, sdkerrors.ErrUnpackAny.Wrapf("failed to unpack input for method %s()", methodName)
	}

	return inputs, nil
}
