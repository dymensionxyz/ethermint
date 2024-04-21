package types

import (
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"math/big"
)

// EvmTxArgs encapsulates all possible params to create all EVM txs types.
// This includes LegacyTx, DynamicFeeTx and AccessListTx
type EvmTxArgs struct {
	Nonce     uint64
	GasLimit  uint64
	Input     []byte
	GasFeeCap *big.Int
	GasPrice  *big.Int
	ChainID   *big.Int
	Amount    *big.Int
	GasTipCap *big.Int
	To        *common.Address
	Accesses  *ethtypes.AccessList
}

func (m EvmTxArgs) NewTx() *evmtypes.MsgEthereumTx {
	return evmtypes.NewTx(
		m.ChainID,   // chain-id
		m.Nonce,     // nonce
		m.To,        // to
		m.Amount,    // amount
		m.GasLimit,  // gas limit
		m.GasPrice,  // gas price
		m.GasFeeCap, // gas fee cap
		m.GasTipCap, // gas tip cap
		m.Input,     // input
		m.Accesses,  // access list
	)
}
