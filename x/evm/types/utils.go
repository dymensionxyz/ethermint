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
	"fmt"
	"math/big"

	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
)

// DefaultPriorityReduction is the default amount of price values required for 1 unit of priority.
// Because priority is `int64` while price is `big.Int`, it's necessary to scale down the range to keep it more pratical.
// The default value is the same as the `sdk.DefaultPowerReduction`.
var DefaultPriorityReduction = sdk.DefaultPowerReduction

var EmptyCodeHash = crypto.Keccak256(nil)

// VFBCCode is the deployed bytecode of the VFBC contract.
// It was copied from state db of the VFBC contract which was deployed via the EVM module.
var VFBCCode = []byte{96, 128, 96, 64, 82, 52, 128, 21, 97, 0, 16, 87, 96, 0, 128, 253, 91, 80, 96, 4, 54, 16, 97, 0, 147, 87, 96, 0, 53, 96, 224, 28, 128, 99, 49, 60, 229, 103, 17, 97, 0, 102, 87, 128, 99, 49, 60, 229, 103, 20, 97, 1, 52, 87, 128, 99, 112, 160, 130, 49, 20, 97, 1, 82, 87, 128, 99, 149, 216, 155, 65, 20, 97, 1, 130, 87, 128, 99, 169, 5, 156, 187, 20, 97, 1, 160, 87, 128, 99, 221, 98, 237, 62, 20, 97, 1, 208, 87, 97, 0, 147, 86, 91, 128, 99, 6, 253, 222, 3, 20, 97, 0, 152, 87, 128, 99, 9, 94, 167, 179, 20, 97, 0, 182, 87, 128, 99, 24, 22, 13, 221, 20, 97, 0, 230, 87, 128, 99, 35, 184, 114, 221, 20, 97, 1, 4, 87, 91, 96, 0, 128, 253, 91, 97, 0, 160, 97, 2, 0, 86, 91, 96, 64, 81, 97, 0, 173, 145, 144, 97, 4, 155, 86, 91, 96, 64, 81, 128, 145, 3, 144, 243, 91, 97, 0, 208, 96, 4, 128, 54, 3, 129, 1, 144, 97, 0, 203, 145, 144, 97, 5, 86, 86, 91, 97, 2, 154, 86, 91, 96, 64, 81, 97, 0, 221, 145, 144, 97, 5, 177, 86, 91, 96, 64, 81, 128, 145, 3, 144, 243, 91, 97, 0, 238, 97, 2, 174, 86, 91, 96, 64, 81, 97, 0, 251, 145, 144, 97, 5, 219, 86, 91, 96, 64, 81, 128, 145, 3, 144, 243, 91, 97, 1, 30, 96, 4, 128, 54, 3, 129, 1, 144, 97, 1, 25, 145, 144, 97, 5, 246, 86, 91, 97, 2, 191, 86, 91, 96, 64, 81, 97, 1, 43, 145, 144, 97, 5, 177, 86, 91, 96, 64, 81, 128, 145, 3, 144, 243, 91, 97, 1, 60, 97, 2, 212, 86, 91, 96, 64, 81, 97, 1, 73, 145, 144, 97, 6, 101, 86, 91, 96, 64, 81, 128, 145, 3, 144, 243, 91, 97, 1, 108, 96, 4, 128, 54, 3, 129, 1, 144, 97, 1, 103, 145, 144, 97, 6, 128, 86, 91, 97, 2, 243, 86, 91, 96, 64, 81, 97, 1, 121, 145, 144, 97, 5, 219, 86, 91, 96, 64, 81, 128, 145, 3, 144, 243, 91, 97, 1, 138, 97, 3, 6, 86, 91, 96, 64, 81, 97, 1, 151, 145, 144, 97, 4, 155, 86, 91, 96, 64, 81, 128, 145, 3, 144, 243, 91, 97, 1, 186, 96, 4, 128, 54, 3, 129, 1, 144, 97, 1, 181, 145, 144, 97, 5, 86, 86, 91, 97, 3, 160, 86, 91, 96, 64, 81, 97, 1, 199, 145, 144, 97, 5, 177, 86, 91, 96, 64, 81, 128, 145, 3, 144, 243, 91, 97, 1, 234, 96, 4, 128, 54, 3, 129, 1, 144, 97, 1, 229, 145, 144, 97, 6, 173, 86, 91, 97, 3, 180, 86, 91, 96, 64, 81, 97, 1, 247, 145, 144, 97, 5, 219, 86, 91, 96, 64, 81, 128, 145, 3, 144, 243, 91, 96, 96, 97, 2, 10, 97, 3, 200, 86, 91, 96, 0, 128, 84, 97, 2, 23, 144, 97, 7, 28, 86, 91, 128, 96, 31, 1, 96, 32, 128, 145, 4, 2, 96, 32, 1, 96, 64, 81, 144, 129, 1, 96, 64, 82, 128, 146, 145, 144, 129, 129, 82, 96, 32, 1, 130, 128, 84, 97, 2, 67, 144, 97, 7, 28, 86, 91, 128, 21, 97, 2, 144, 87, 128, 96, 31, 16, 97, 2, 101, 87, 97, 1, 0, 128, 131, 84, 4, 2, 131, 82, 145, 96, 32, 1, 145, 97, 2, 144, 86, 91, 130, 1, 145, 144, 96, 0, 82, 96, 32, 96, 0, 32, 144, 91, 129, 84, 129, 82, 144, 96, 1, 1, 144, 96, 32, 1, 128, 131, 17, 97, 2, 115, 87, 130, 144, 3, 96, 31, 22, 130, 1, 145, 91, 80, 80, 80, 80, 80, 144, 80, 144, 86, 91, 96, 0, 97, 2, 164, 97, 3, 200, 86, 91, 96, 0, 144, 80, 146, 145, 80, 80, 86, 91, 96, 0, 97, 2, 184, 97, 3, 200, 86, 91, 96, 0, 144, 80, 144, 86, 91, 96, 0, 97, 2, 201, 97, 3, 200, 86, 91, 96, 0, 144, 80, 147, 146, 80, 80, 80, 86, 91, 96, 0, 97, 2, 222, 97, 3, 200, 86, 91, 96, 2, 96, 0, 144, 84, 144, 97, 1, 0, 10, 144, 4, 96, 255, 22, 144, 80, 144, 86, 91, 96, 0, 97, 2, 253, 97, 3, 200, 86, 91, 96, 0, 144, 80, 145, 144, 80, 86, 91, 96, 96, 97, 3, 16, 97, 3, 200, 86, 91, 96, 1, 128, 84, 97, 3, 29, 144, 97, 7, 28, 86, 91, 128, 96, 31, 1, 96, 32, 128, 145, 4, 2, 96, 32, 1, 96, 64, 81, 144, 129, 1, 96, 64, 82, 128, 146, 145, 144, 129, 129, 82, 96, 32, 1, 130, 128, 84, 97, 3, 73, 144, 97, 7, 28, 86, 91, 128, 21, 97, 3, 150, 87, 128, 96, 31, 16, 97, 3, 107, 87, 97, 1, 0, 128, 131, 84, 4, 2, 131, 82, 145, 96, 32, 1, 145, 97, 3, 150, 86, 91, 130, 1, 145, 144, 96, 0, 82, 96, 32, 96, 0, 32, 144, 91, 129, 84, 129, 82, 144, 96, 1, 1, 144, 96, 32, 1, 128, 131, 17, 97, 3, 121, 87, 130, 144, 3, 96, 31, 22, 130, 1, 145, 91, 80, 80, 80, 80, 80, 144, 80, 144, 86, 91, 96, 0, 97, 3, 170, 97, 3, 200, 86, 91, 96, 0, 144, 80, 146, 145, 80, 80, 86, 91, 96, 0, 97, 3, 190, 97, 3, 200, 86, 91, 96, 0, 144, 80, 146, 145, 80, 80, 86, 91, 96, 0, 97, 4, 9, 87, 96, 64, 81, 127, 8, 195, 121, 160, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 129, 82, 96, 4, 1, 97, 4, 0, 144, 97, 7, 153, 86, 91, 96, 64, 81, 128, 145, 3, 144, 253, 91, 86, 91, 96, 0, 129, 81, 144, 80, 145, 144, 80, 86, 91, 96, 0, 130, 130, 82, 96, 32, 130, 1, 144, 80, 146, 145, 80, 80, 86, 91, 96, 0, 91, 131, 129, 16, 21, 97, 4, 69, 87, 128, 130, 1, 81, 129, 132, 1, 82, 96, 32, 129, 1, 144, 80, 97, 4, 42, 86, 91, 96, 0, 132, 132, 1, 82, 80, 80, 80, 80, 86, 91, 96, 0, 96, 31, 25, 96, 31, 131, 1, 22, 144, 80, 145, 144, 80, 86, 91, 96, 0, 97, 4, 109, 130, 97, 4, 11, 86, 91, 97, 4, 119, 129, 133, 97, 4, 22, 86, 91, 147, 80, 97, 4, 135, 129, 133, 96, 32, 134, 1, 97, 4, 39, 86, 91, 97, 4, 144, 129, 97, 4, 81, 86, 91, 132, 1, 145, 80, 80, 146, 145, 80, 80, 86, 91, 96, 0, 96, 32, 130, 1, 144, 80, 129, 129, 3, 96, 0, 131, 1, 82, 97, 4, 181, 129, 132, 97, 4, 98, 86, 91, 144, 80, 146, 145, 80, 80, 86, 91, 96, 0, 128, 253, 91, 96, 0, 115, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 130, 22, 144, 80, 145, 144, 80, 86, 91, 96, 0, 97, 4, 237, 130, 97, 4, 194, 86, 91, 144, 80, 145, 144, 80, 86, 91, 97, 4, 253, 129, 97, 4, 226, 86, 91, 129, 20, 97, 5, 8, 87, 96, 0, 128, 253, 91, 80, 86, 91, 96, 0, 129, 53, 144, 80, 97, 5, 26, 129, 97, 4, 244, 86, 91, 146, 145, 80, 80, 86, 91, 96, 0, 129, 144, 80, 145, 144, 80, 86, 91, 97, 5, 51, 129, 97, 5, 32, 86, 91, 129, 20, 97, 5, 62, 87, 96, 0, 128, 253, 91, 80, 86, 91, 96, 0, 129, 53, 144, 80, 97, 5, 80, 129, 97, 5, 42, 86, 91, 146, 145, 80, 80, 86, 91, 96, 0, 128, 96, 64, 131, 133, 3, 18, 21, 97, 5, 109, 87, 97, 5, 108, 97, 4, 189, 86, 91, 91, 96, 0, 97, 5, 123, 133, 130, 134, 1, 97, 5, 11, 86, 91, 146, 80, 80, 96, 32, 97, 5, 140, 133, 130, 134, 1, 97, 5, 65, 86, 91, 145, 80, 80, 146, 80, 146, 144, 80, 86, 91, 96, 0, 129, 21, 21, 144, 80, 145, 144, 80, 86, 91, 97, 5, 171, 129, 97, 5, 150, 86, 91, 130, 82, 80, 80, 86, 91, 96, 0, 96, 32, 130, 1, 144, 80, 97, 5, 198, 96, 0, 131, 1, 132, 97, 5, 162, 86, 91, 146, 145, 80, 80, 86, 91, 97, 5, 213, 129, 97, 5, 32, 86, 91, 130, 82, 80, 80, 86, 91, 96, 0, 96, 32, 130, 1, 144, 80, 97, 5, 240, 96, 0, 131, 1, 132, 97, 5, 204, 86, 91, 146, 145, 80, 80, 86, 91, 96, 0, 128, 96, 0, 96, 96, 132, 134, 3, 18, 21, 97, 6, 15, 87, 97, 6, 14, 97, 4, 189, 86, 91, 91, 96, 0, 97, 6, 29, 134, 130, 135, 1, 97, 5, 11, 86, 91, 147, 80, 80, 96, 32, 97, 6, 46, 134, 130, 135, 1, 97, 5, 11, 86, 91, 146, 80, 80, 96, 64, 97, 6, 63, 134, 130, 135, 1, 97, 5, 65, 86, 91, 145, 80, 80, 146, 80, 146, 80, 146, 86, 91, 96, 0, 96, 255, 130, 22, 144, 80, 145, 144, 80, 86, 91, 97, 6, 95, 129, 97, 6, 73, 86, 91, 130, 82, 80, 80, 86, 91, 96, 0, 96, 32, 130, 1, 144, 80, 97, 6, 122, 96, 0, 131, 1, 132, 97, 6, 86, 86, 91, 146, 145, 80, 80, 86, 91, 96, 0, 96, 32, 130, 132, 3, 18, 21, 97, 6, 150, 87, 97, 6, 149, 97, 4, 189, 86, 91, 91, 96, 0, 97, 6, 164, 132, 130, 133, 1, 97, 5, 11, 86, 91, 145, 80, 80, 146, 145, 80, 80, 86, 91, 96, 0, 128, 96, 64, 131, 133, 3, 18, 21, 97, 6, 196, 87, 97, 6, 195, 97, 4, 189, 86, 91, 91, 96, 0, 97, 6, 210, 133, 130, 134, 1, 97, 5, 11, 86, 91, 146, 80, 80, 96, 32, 97, 6, 227, 133, 130, 134, 1, 97, 5, 11, 86, 91, 145, 80, 80, 146, 80, 146, 144, 80, 86, 91, 127, 78, 72, 123, 113, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 96, 0, 82, 96, 34, 96, 4, 82, 96, 36, 96, 0, 253, 91, 96, 0, 96, 2, 130, 4, 144, 80, 96, 1, 130, 22, 128, 97, 7, 52, 87, 96, 127, 130, 22, 145, 80, 91, 96, 32, 130, 16, 129, 3, 97, 7, 71, 87, 97, 7, 70, 97, 6, 237, 86, 91, 91, 80, 145, 144, 80, 86, 91, 127, 73, 32, 97, 109, 32, 97, 32, 114, 111, 99, 107, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 96, 0, 130, 1, 82, 80, 86, 91, 96, 0, 97, 7, 131, 96, 11, 131, 97, 4, 22, 86, 91, 145, 80, 97, 7, 142, 130, 97, 7, 77, 86, 91, 96, 32, 130, 1, 144, 80, 145, 144, 80, 86, 91, 96, 0, 96, 32, 130, 1, 144, 80, 129, 129, 3, 96, 0, 131, 1, 82, 97, 7, 178, 129, 97, 7, 118, 86, 91, 144, 80, 145, 144, 80, 86, 254, 162, 100, 105, 112, 102, 115, 88, 34, 18, 32, 250, 47, 81, 118, 119, 131, 214, 27, 0, 143, 219, 7, 3, 251, 243, 174, 7, 93, 193, 219, 4, 199, 171, 23, 50, 51, 225, 46, 117, 170, 22, 153, 100, 115, 111, 108, 99, 67, 0, 8, 17, 0, 51}

// VFBCCodeHash is the code hash of the VFBC contract, corresponding to the VFBCCode.
var VFBCCodeHash = crypto.Keccak256(VFBCCode)

// DecodeTxResponse decodes an protobuf-encoded byte slice into TxResponse
func DecodeTxResponse(in []byte) (*MsgEthereumTxResponse, error) {
	var txMsgData sdk.TxMsgData
	if err := proto.Unmarshal(in, &txMsgData); err != nil {
		return nil, err
	}

	if len(txMsgData.MsgResponses) == 0 {
		return &MsgEthereumTxResponse{}, nil
	}

	var res MsgEthereumTxResponse
	if err := proto.Unmarshal(txMsgData.MsgResponses[0].Value, &res); err != nil {
		return nil, errorsmod.Wrap(err, "failed to unmarshal tx response message data")
	}

	return &res, nil
}

// EncodeTransactionLogs encodes TransactionLogs slice into a protobuf-encoded byte slice.
func EncodeTransactionLogs(res *TransactionLogs) ([]byte, error) {
	return proto.Marshal(res)
}

// DecodeTransactionLogs decodes an protobuf-encoded byte slice into TransactionLogs
func DecodeTransactionLogs(data []byte) (TransactionLogs, error) {
	var logs TransactionLogs
	err := proto.Unmarshal(data, &logs)
	if err != nil {
		return TransactionLogs{}, err
	}
	return logs, nil
}

// UnwrapEthereumMsg extract MsgEthereumTx from wrapping sdk.Tx
func UnwrapEthereumMsg(tx *sdk.Tx, ethHash common.Hash) (*MsgEthereumTx, error) {
	if tx == nil {
		return nil, fmt.Errorf("invalid tx: nil")
	}

	for _, msg := range (*tx).GetMsgs() {
		ethMsg, ok := msg.(*MsgEthereumTx)
		if !ok {
			return nil, fmt.Errorf("invalid tx type: %T", tx)
		}
		txHash := ethMsg.AsTransaction().Hash()
		ethMsg.Hash = txHash.Hex()
		if txHash == ethHash {
			return ethMsg, nil
		}
	}

	return nil, fmt.Errorf("eth tx not found: %s", ethHash)
}

// BinSearch execute the binary search and hone in on an executable gas limit
func BinSearch(lo, hi uint64, executable func(uint64) (bool, *MsgEthereumTxResponse, error)) (uint64, error) {
	for lo+1 < hi {
		mid := (hi + lo) / 2
		failed, _, err := executable(mid)
		// If the error is not nil(consensus error), it means the provided message
		// call or transaction will never be accepted no matter how much gas it is
		// assigned. Return the error directly, don't struggle any more.
		if err != nil {
			return 0, err
		}
		if failed {
			lo = mid
		} else {
			hi = mid
		}
	}
	return hi, nil
}

// EffectiveGasPrice compute the effective gas price based on eip-1159 rules
// `effectiveGasPrice = min(baseFee + tipCap, feeCap)`
func EffectiveGasPrice(baseFee *big.Int, feeCap *big.Int, tipCap *big.Int) *big.Int {
	return math.BigMin(new(big.Int).Add(tipCap, baseFee), feeCap)
}
