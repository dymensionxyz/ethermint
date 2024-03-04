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
package keeper

import (
	"encoding/hex"
	"fmt"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/math"
	"math/big"

	tmtypes "github.com/tendermint/tendermint/types"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ethermint "github.com/evmos/ethermint/types"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/evmos/ethermint/x/evm/types"
	evm "github.com/evmos/ethermint/x/evm/vm"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// NewEVM generates a go-ethereum VM from the provided Message fields and the chain parameters
// (ChainConfig and module Params). It additionally sets the validator operator address as the
// coinbase address to make it available for the COINBASE opcode, even though there is no
// beneficiary of the coinbase transaction (since we're not mining).
//
// NOTE: the RANDOM opcode is currently not supported since it requires
// RANDAO implementation. See https://github.com/evmos/ethermint/pull/1520#pullrequestreview-1200504697
// for more information.

func (k *Keeper) NewEVM(
	ctx sdk.Context,
	msg core.Message,
	cfg *statedb.EVMConfig,
	tracer vm.EVMLogger,
	stateDB vm.StateDB,
) evm.EVM {
	blockCtx := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     k.GetHashFn(ctx),
		Coinbase:    cfg.CoinBase,
		GasLimit:    ethermint.BlockGasLimit(ctx),
		BlockNumber: big.NewInt(ctx.BlockHeight()),
		Time:        big.NewInt(ctx.BlockHeader().Time.Unix()),
		Difficulty:  big.NewInt(0), // unused. Only required in PoW context
		BaseFee:     cfg.BaseFee,
		Random:      nil, // not supported
	}

	txCtx := core.NewEVMTxContext(msg)
	if tracer == nil {
		tracer = k.Tracer(ctx, msg, cfg.ChainConfig)
	}
	vmConfig := k.VMConfig(ctx, msg, cfg, tracer)
	return k.evmConstructor(blockCtx, txCtx, stateDB, cfg.ChainConfig, vmConfig, k.customPrecompiles)
}

// GetHashFn implements vm.GetHashFunc for Ethermint. It handles 3 cases:
//  1. The requested height matches the current height from context (and thus same epoch number)
//  2. The requested height is from an previous height from the same chain epoch
//  3. The requested height is from a height greater than the latest one
func (k Keeper) GetHashFn(ctx sdk.Context) vm.GetHashFunc {
	return func(height uint64) common.Hash {
		h, err := ethermint.SafeInt64(height)
		if err != nil {
			k.Logger(ctx).Error("failed to cast height to int64", "error", err)
			return common.Hash{}
		}

		switch {
		case ctx.BlockHeight() == h:
			// Case 1: The requested height matches the one from the context so we can retrieve the header
			// hash directly from the context.
			// Note: The headerHash is only set at begin block, it will be nil in case of a query context
			headerHash := ctx.HeaderHash()
			if len(headerHash) != 0 {
				return common.BytesToHash(headerHash)
			}

			// only recompute the hash if not set (eg: checkTxState)
			contextBlockHeader := ctx.BlockHeader()
			header, err := tmtypes.HeaderFromProto(&contextBlockHeader)
			if err != nil {
				k.Logger(ctx).Error("failed to cast tendermint header from proto", "error", err)
				return common.Hash{}
			}

			headerHash = header.Hash()
			return common.BytesToHash(headerHash)

		case ctx.BlockHeight() > h:
			// Case 2: if the chain is not the current height we need to retrieve the hash from the store for the
			// current chain epoch. This only applies if the current height is greater than the requested height.
			histInfo, found := k.stakingKeeper.GetHistoricalInfo(ctx, h)
			if !found {
				k.Logger(ctx).Debug("historical info not found", "height", h)
				return common.Hash{}
			}

			header, err := tmtypes.HeaderFromProto(&histInfo.Header)
			if err != nil {
				k.Logger(ctx).Error("failed to cast tendermint header from proto", "error", err)
				return common.Hash{}
			}

			return common.BytesToHash(header.Hash())
		default:
			// Case 3: heights greater than the current one returns an empty hash.
			return common.Hash{}
		}
	}
}

// ApplyTransaction runs and attempts to perform a state transition with the given transaction (i.e Message), that will
// only be persisted (committed) to the underlying KVStore if the transaction does not fail.
//
// # Gas tracking
//
// Ethereum consumes gas according to the EVM opcodes instead of general reads and writes to store. Because of this, the
// state transition needs to ignore the SDK gas consumption mechanism defined by the GasKVStore and instead consume the
// amount of gas used by the VM execution. The amount of gas used is tracked by the EVM and returned in the execution
// result.
//
// Prior to the execution, the starting tx gas meter is saved and replaced with an infinite gas meter in a new context
// in order to ignore the SDK gas consumption config values (read, write, has, delete).
// After the execution, the gas used from the message execution will be added to the starting gas consumed, taking into
// consideration the amount of gas returned. Finally, the context is updated with the EVM gas consumed value prior to
// returning.
//
// For relevant discussion see: https://github.com/cosmos/cosmos-sdk/discussions/9072
func (k *Keeper) ApplyTransaction(ctx sdk.Context, msgEth *types.MsgEthereumTx) (*types.MsgEthereumTxResponse, error) {
	var (
		bloom        *big.Int
		bloomReceipt ethtypes.Bloom
	)

	cfg, err := k.EVMConfig(ctx, sdk.ConsAddress(ctx.BlockHeader().ProposerAddress), k.eip155ChainID)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to load evm config")
	}
	ethTx := msgEth.AsTransaction()
	txConfig := k.TxConfig(ctx, ethTx.Hash())

	// get the signer according to the chain rules from the config and block height
	signer := ethtypes.MakeSigner(cfg.ChainConfig, big.NewInt(ctx.BlockHeight()))
	msg, err := msgEth.AsMessage(signer, cfg.BaseFee)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to return ethereum transaction as core message")
	}

	// snapshot to contain the tx processing and post processing in same scope
	var commit func()
	tmpCtx := ctx
	if k.hooks != nil {
		// Create a cache context to revert state when tx hooks fails,
		// the cache context is only committed when both tx and hooks executed successfully.
		// Didn't use `Snapshot` because the context stack has exponential complexity on certain operations,
		// thus restricted to be used only inside `ApplyMessage`.
		tmpCtx, commit = ctx.CacheContext()
	}

	// pass true to commit the StateDB
	res, err := k.ApplyMessageWithConfig(tmpCtx, msg, nil, true, cfg, txConfig)
	if err != nil {
		// when a transaction contains multiple msg, as long as one of the msg fails
		// all gas will be deducted. so is not msg.Gas()
		k.ResetGasMeterAndConsumeGas(ctx, ctx.GasMeter().Limit())
		return nil, errorsmod.Wrap(err, "failed to apply ethereum core message")
	}

	logs := types.LogsToEthereum(res.Logs)

	// Compute block bloom filter
	if len(logs) > 0 {
		bloom = k.GetBlockBloomTransient(ctx)
		bloom.Or(bloom, big.NewInt(0).SetBytes(ethtypes.LogsBloom(logs)))
		bloomReceipt = ethtypes.BytesToBloom(bloom.Bytes())
	}

	cumulativeGasUsed := res.GasUsed
	if ctx.BlockGasMeter() != nil {
		limit := ctx.BlockGasMeter().Limit()
		cumulativeGasUsed += ctx.BlockGasMeter().GasConsumed()
		if cumulativeGasUsed > limit {
			cumulativeGasUsed = limit
		}
	}

	var contractAddr common.Address
	if msg.To() == nil {
		contractAddr = crypto.CreateAddress(msg.From(), msg.Nonce())
	}

	receipt := &ethtypes.Receipt{
		Type:              ethTx.Type(),
		PostState:         nil, // TODO: intermediate state root
		CumulativeGasUsed: cumulativeGasUsed,
		Bloom:             bloomReceipt,
		Logs:              logs,
		TxHash:            txConfig.TxHash,
		ContractAddress:   contractAddr,
		GasUsed:           res.GasUsed,
		BlockHash:         txConfig.BlockHash,
		BlockNumber:       big.NewInt(ctx.BlockHeight()),
		TransactionIndex:  txConfig.TxIndex,
	}

	if !res.Failed() {
		receipt.Status = ethtypes.ReceiptStatusSuccessful
		// Only call hooks if tx executed successfully.
		if err = k.PostTxProcessing(tmpCtx, msg, receipt); err != nil {
			// If hooks return error, revert the whole tx.
			res.VmError = types.ErrPostTxProcessing.Error()
			k.Logger(ctx).Error("tx post processing failed", "error", err)

			// If the tx failed in post processing hooks, we should clear the logs
			res.Logs = nil
		} else if commit != nil {
			// PostTxProcessing is successful, commit the tmpCtx
			commit()
			// Since the post-processing can alter the log, we need to update the result
			res.Logs = types.NewLogsFromEth(receipt.Logs)
			ctx.EventManager().EmitEvents(tmpCtx.EventManager().Events())
		}
	}

	// refund gas in order to match the Ethereum gas consumption instead of the default SDK one.
	if err = k.RefundGas(ctx, msg, msg.Gas()-res.GasUsed, cfg.Params.EvmDenom); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to refund gas leftover gas to sender %s", msg.From())
	}

	if len(receipt.Logs) > 0 {
		// Update transient block bloom filter
		k.SetBlockBloomTransient(ctx, receipt.Bloom.Big())
		k.SetLogSizeTransient(ctx, uint64(txConfig.LogIndex)+uint64(len(receipt.Logs)))
	}

	k.SetTxIndexTransient(ctx, uint64(txConfig.TxIndex)+1)

	totalGasUsed, err := k.AddTransientGasUsed(ctx, res.GasUsed)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to add transient gas used")
	}

	// reset the gas meter for current cosmos transaction
	k.ResetGasMeterAndConsumeGas(ctx, totalGasUsed)
	return res, nil
}

// ApplyMessage calls ApplyMessageWithConfig with an empty TxConfig.
func (k *Keeper) ApplyMessage(ctx sdk.Context, msg core.Message, tracer vm.EVMLogger, commit bool) (*types.MsgEthereumTxResponse, error) {
	cfg, err := k.EVMConfig(ctx, sdk.ConsAddress(ctx.BlockHeader().ProposerAddress), k.eip155ChainID)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to load evm config")
	}

	txConfig := statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))
	return k.ApplyMessageWithConfig(ctx, msg, tracer, commit, cfg, txConfig)
}

// ApplyMessageWithConfig computes the new state by applying the given message against the existing state.
// If the message fails, the VM execution error with the reason will be returned to the client
// and the transaction won't be committed to the store.
//
// # Reverted state
//
// The snapshot and rollback are supported by the `statedb.StateDB`.
//
// # Different Callers
//
// It's called in three scenarios:
// 1. `ApplyTransaction`, in the transaction processing flow.
// 2. `EthCall/EthEstimateGas` grpc query handler.
// 3. Called by other native modules directly.
//
// # Prechecks and Preprocessing
//
// All relevant state transition prechecks for the MsgEthereumTx are performed on the AnteHandler,
// prior to running the transaction against the state. The prechecks run are the following:
//
// 1. the nonce of the message caller is correct
// 2. caller has enough balance to cover transaction fee(gaslimit * gasprice)
// 3. the amount of gas required is available in the block
// 4. the purchased gas is enough to cover intrinsic usage
// 5. there is no overflow when calculating intrinsic gas
// 6. caller has enough balance to cover asset transfer for **topmost** call
//
// The preprocessing steps performed by the AnteHandler are:
//
// 1. set up the initial access list (iff fork > Berlin)
//
// # Tracer parameter
//
// It should be a `vm.Tracer` object or nil, if pass `nil`, it'll create a default one based on keeper options.
//
// # Commit parameter
//
// If commit is true, the `StateDB` will be committed, otherwise discarded.
func (k *Keeper) ApplyMessageWithConfig(ctx sdk.Context,
	msg core.Message,
	tracer vm.EVMLogger,
	commit bool,
	cfg *statedb.EVMConfig,
	txConfig statedb.TxConfig,
) (*types.MsgEthereumTxResponse, error) {
	var (
		ret   []byte // return bytes from evm execution
		vmErr error  // vm errors do not effect consensus and are therefore not assigned to err
	)

	// return error if contract creation or call are disabled through governance
	if !cfg.Params.EnableCreate && msg.To() == nil {
		return nil, errorsmod.Wrap(types.ErrCreateDisabled, "failed to create new contract")
	} else if !cfg.Params.EnableCall && msg.To() != nil {
		return nil, errorsmod.Wrap(types.ErrCallDisabled, "failed to call contract")
	}

	stateDB := statedb.New(ctx, k, txConfig)
	evm := k.NewEVM(ctx, msg, cfg, tracer, stateDB)

	leftoverGas := msg.Gas()

	// Allow the tracer captures the tx level events, mainly the gas consumption.
	vmCfg := evm.Config()
	if vmCfg.Debug {
		vmCfg.Tracer.CaptureTxStart(leftoverGas)
		defer func() {
			vmCfg.Tracer.CaptureTxEnd(leftoverGas)
		}()
	}

	sender := vm.AccountRef(msg.From())
	contractCreation := msg.To() == nil
	isLondon := cfg.ChainConfig.IsLondon(evm.Context().BlockNumber)

	intrinsicGas, err := k.GetEthIntrinsicGas(ctx, msg, cfg.ChainConfig, contractCreation)
	if err != nil {
		// should have already been checked on Ante Handler
		return nil, errorsmod.Wrap(err, "intrinsic gas failed")
	}

	// Should check again even if it is checked on Ante Handler, because eth_call don't go through Ante Handler.
	if leftoverGas < intrinsicGas {
		// eth_estimateGas will check for this exact error
		return nil, errorsmod.Wrap(core.ErrIntrinsicGas, "apply message")
	}
	leftoverGas -= intrinsicGas

	// access list preparation is moved from ante handler to here, because it's needed when `ApplyMessage` is called
	// under contexts where ante handlers are not run, for example `eth_call` and `eth_estimateGas`.
	if rules := cfg.ChainConfig.Rules(big.NewInt(ctx.BlockHeight()), cfg.ChainConfig.MergeNetsplitBlock != nil); rules.IsBerlin {
		stateDB.PrepareAccessList(msg.From(), msg.To(), evm.ActivePrecompiles(rules), msg.AccessList())
	}

	if contractCreation {
		// take over the nonce management from evm:
		// - reset sender's nonce to msg.Nonce() before calling evm.
		// - increase sender's nonce by one no matter the result.
		stateDB.SetNonce(sender.Address(), msg.Nonce())
		ret, _, leftoverGas, vmErr = evm.Create(sender, msg.Data(), leftoverGas, msg.Value())
		stateDB.SetNonce(sender.Address(), msg.Nonce()+1)
	} else {
		ret, leftoverGas, vmErr = k.proxiedEvmCall(ctx, evm, stateDB, sender, *msg.To(), msg.Data(), leftoverGas, msg.Value())
	}

	refundQuotient := params.RefundQuotient

	// After EIP-3529: refunds are capped to gasUsed / 5
	if isLondon {
		refundQuotient = params.RefundQuotientEIP3529
	}

	// calculate gas refund
	if msg.Gas() < leftoverGas {
		return nil, errorsmod.Wrap(types.ErrGasOverflow, "apply message")
	}
	// refund gas
	temporaryGasUsed := msg.Gas() - leftoverGas
	leftoverGas += GasToRefund(stateDB.GetRefund(), temporaryGasUsed, refundQuotient)

	// EVM execution error needs to be available for the JSON-RPC client
	var vmError string
	if vmErr != nil {
		vmError = vmErr.Error()
	}

	// The dirty states in `StateDB` is either committed or discarded after return
	if commit {
		if err := stateDB.Commit(); err != nil {
			return nil, errorsmod.Wrap(err, "failed to commit stateDB")
		}
	}

	// calculate a minimum amount of gas to be charged to sender if GasLimit
	// is considerably higher than GasUsed to stay more aligned with Tendermint gas mechanics
	// for more info https://github.com/evmos/ethermint/issues/1085
	gasLimit := sdk.NewDec(int64(msg.Gas()))
	minGasMultiplier := k.GetMinGasMultiplier(ctx)
	minimumGasUsed := gasLimit.Mul(minGasMultiplier)

	if !minimumGasUsed.TruncateInt().IsUint64() {
		return nil, errorsmod.Wrapf(types.ErrGasOverflow, "minimumGasUsed(%s) is not a uint64", minimumGasUsed.TruncateInt().String())
	}

	if msg.Gas() < leftoverGas {
		return nil, errorsmod.Wrapf(types.ErrGasOverflow, "message gas limit < leftover gas (%d < %d)", msg.Gas(), leftoverGas)
	}

	gasUsed := sdk.MaxDec(minimumGasUsed, sdk.NewDec(int64(temporaryGasUsed))).TruncateInt().Uint64()
	// reset leftoverGas, to be used by the tracer
	leftoverGas = msg.Gas() - gasUsed

	return &types.MsgEthereumTxResponse{
		GasUsed: gasUsed,
		VmError: vmError,
		Ret:     ret,
		Logs:    types.NewLogsFromEth(stateDB.Logs()),
		Hash:    txConfig.TxHash.Hex(),
	}, nil
}

// proxiedEvmCall is the proxied method of the EVM::Call method.
// It is used to intercept the call request, make decision before actual invoking call to the EVM::Call.
// If the target
func (k *Keeper) proxiedEvmCall(ctx sdk.Context, evm evm.EVM, stateDB vm.StateDB, caller vm.ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, vmErr error) {
	if k.IsVirtualFrontierContract(ctx, addr) {
		vfContract := k.GetVirtualFrontierContract(ctx, addr)
		if vfContract == nil {
			return nil, 0, types.ErrVMExecution.Wrapf("virtual frontier contract %s could not be found", addr)
		}

		if !vfContract.Active {
			return nil, 0, types.ErrVMExecution.Wrapf("the virtual frontier contract %s is not active", addr)
		}

		switch types.VirtualFrontierContractType(vfContract.Type) {
		case types.VirtualFrontierContractTypeBankContract:
			return k.evmCallVirtualFrontierBankContract(ctx, stateDB, caller.Address(), vfContract, input, gas, value)
		default:
			panic(fmt.Errorf("not implemented handler for VF contract %d", vfContract.Type))
		}
	}

	return evm.Call(caller, addr, input, gas, value)
}

// evmCallVirtualFrontierBankContract handles EVM call to a virtual frontier bank contract.
func (k *Keeper) evmCallVirtualFrontierBankContract(
	ctx sdk.Context,
	stateDB vm.StateDB,
	sender common.Address, virtualFrontierContract *types.VirtualFrontierContract, calldata []byte, gas uint64, value *big.Int,
) (ret []byte, leftOverGas uint64, vmErr error) {
	defer func() {
		if vmErr != nil {
			k.Logger(ctx).Debug("virtual frontier bank contract execution failed", "error", vmErr)
		}
	}()

	compiledVFContract := types.VFBankContract20

	/*
		For all the call to the virtual frontier bank contract, we charge flattened this gas amount,
		intrinsic gas cost (21k) are excluded.
	*/
	const flattenedGasCost uint64 = 59000

	defer func() {
		if vmErr != nil {
			// consume all gas if an error occurs
			leftOverGas = 0
		} else {
			// consume the flattened gas cost
			tmpLeftOver, overflow := math.SafeSub(gas, flattenedGasCost)
			if overflow {
				// if we can not sub, we consume all gas
				leftOverGas = 0
			} else {
				leftOverGas = tmpLeftOver
			}
		}
	}()

	if virtualFrontierContract.Type != uint32(types.VirtualFrontierContractTypeBankContract) {
		vmErr = types.ErrVMExecution.Wrapf("virtual frontier contract type %d is not a bank contract", virtualFrontierContract.Type)
		return
	}

	// prohibit normal transfer to the bank contract
	if len(calldata) < 1 {
		vmErr = types.ErrProhibitedAccessingVirtualFrontierContract.Wrap("can not transfer to virtual frontier bank contract")
		return
	}

	if len(calldata) < 4 {
		vmErr = types.ErrVMExecution.Wrap("invalid call data")
		return
	}

	// prohibit transfer native token to the VF contract
	if value != nil && value.Sign() != 0 {
		vmErr = types.ErrProhibitedAccessingVirtualFrontierContract.Wrap("cannot transfer to virtual frontier bank contract")
		return
	}

	var bankContractMetadata types.VFBankContractMetadata
	if err := k.cdc.Unmarshal(virtualFrontierContract.Metadata, &bankContractMetadata); err != nil {
		vmErr = types.ErrVMExecution.Wrapf("failed to unmarshal virtual frontier bank contract metadata: %v", err)
		return
	}

	method, found := bankContractMetadata.GetMethodFromSignature(calldata)
	if !found {
		if len(calldata) >= 4 {
			vmErr = types.ErrVMExecution.Wrapf("unknown method signature 0x%s", hex.EncodeToString(calldata[:4]))
		} else {
			vmErr = types.ErrVMExecution.Wrapf("unknown method signature 0x%s", hex.EncodeToString(calldata))
		}
		return
	}

	bankDenomMetadata, found := k.bankKeeper.GetDenomMetaData(ctx, bankContractMetadata.MinDenom)
	if !found {
		vmErr = types.ErrVMExecution.Wrapf("bank denom metadata not found for %s", bankContractMetadata.MinDenom)
		return
	}

	if flattenedGasCost > gas {
		vmErr = vm.ErrOutOfGas
		return
	}

	vfbcDenomMetadata, _ /*ignore invalid state of bank denom-metadata*/ := types.CollectMetadataForVirtualFrontierBankContract(bankDenomMetadata)

	switch method {
	case types.VFBCmName:
		bz, err := compiledVFContract.PackOutput("name", vfbcDenomMetadata.Name)

		if err != nil {
			vmErr = err
			return
		}

		ret = bz
		return
	case types.VFBCmSymbol:
		bz, err := compiledVFContract.PackOutput("symbol", vfbcDenomMetadata.Symbol)

		if err != nil {
			vmErr = err
			return
		}

		ret = bz
		return
	case types.VFBCmDecimals:
		if !vfbcDenomMetadata.CanDecimalsUint8() {
			vmErr = types.ErrVMExecution.Wrapf("decimals overflow %d", vfbcDenomMetadata.Decimals)
			return
		}

		bz, err := compiledVFContract.PackOutput("decimals", uint8(vfbcDenomMetadata.Decimals))

		if err != nil {
			vmErr = err
			return
		}

		ret = bz
		return
	case types.VFBCmTotalSupply:
		totalSupply := k.bankKeeper.GetSupply(ctx, bankContractMetadata.MinDenom)

		bz, err := compiledVFContract.PackOutput("totalSupply", new(big.Int).SetUint64(totalSupply.Amount.Uint64()))

		if err != nil {
			vmErr = err
			return
		}

		ret = bz
		return
	case types.VFBCmBalanceOf:
		if len(calldata) < 5 {
			vmErr = types.ErrVMExecution.Wrap("invalid call data")
			return
		}

		// unpack the calldata
		inputs, err := compiledVFContract.UnpackInput("balanceOf", calldata[4:])

		if err != nil {
			vmErr = err
			return
		}

		if len(inputs) != 1 {
			vmErr = types.ErrExecutionReverted
			return
		}

		receiverAddress, ok := inputs[0].(common.Address)
		if !ok {
			vmErr = types.ErrExecutionReverted.Wrap("first input is not a contract address")
			return
		}

		// get the balance of the address
		balance := k.bankKeeper.GetBalance(ctx, receiverAddress.Bytes(), bankContractMetadata.MinDenom)

		// pack the output

		bz, err := compiledVFContract.PackOutput("balanceOf", balance.Amount.BigInt())

		if err != nil {
			vmErr = err
			return
		}

		ret = bz
		return
	case types.VFBCmTransfer:
		if len(calldata) < 5 {
			vmErr = types.ErrVMExecution.Wrap("invalid call data")
			return
		}

		eventTransfer, foundEvent := compiledVFContract.ABI.Events["Transfer"]
		if !foundEvent {
			vmErr = types.ErrVMExecution.Wrap("event Transfer could not be found")
			return
		}

		// unpack the calldata
		inputs, err := compiledVFContract.UnpackInput("transfer", calldata[4:])

		if err != nil {
			vmErr = err
			return
		}

		if len(inputs) != 2 {
			vmErr = types.ErrExecutionReverted
			return
		}

		to, ok := inputs[0].(common.Address)
		if !ok {
			vmErr = types.ErrExecutionReverted
			return
		}

		receiver := sdk.AccAddress(to.Bytes())

		// prohibit transfer to some types of account

		// - module account
		accountI := k.accountKeeper.GetAccount(ctx, receiver)
		if accountI != nil {
			_, isModuleAccount := accountI.(authtypes.ModuleAccountI)
			if isModuleAccount {
				vmErr = types.ErrVMExecution.Wrap("can not transfer to module account")
				return
			}
		}
		// - VF contracts
		if k.IsVirtualFrontierContract(ctx, to) {
			vmErr = types.ErrProhibitedAccessingVirtualFrontierContract.Wrap("can not transfer to virtual frontier contract")
			return
		}

		amount, ok := inputs[1].(*big.Int)
		if !ok {
			vmErr = types.ErrExecutionReverted
			return
		}

		senderBalance := k.bankKeeper.GetBalance(ctx, sender.Bytes(), bankContractMetadata.MinDenom)
		sendAmount := sdk.NewCoin(bankContractMetadata.MinDenom, sdk.NewIntFromBigInt(amount))

		if senderBalance.Amount.LT(sendAmount.Amount) {
			vmErr = types.ErrVMExecution.Wrapf("insufficient balance %s < %s", senderBalance, sendAmount)
			return
		}

		if err := k.bankKeeper.IsSendEnabledCoins(ctx, sendAmount); err != nil {
			vmErr = types.ErrVMExecution.Wrap(err.Error())
			return
		}

		if k.bankKeeper.BlockedAddr(receiver) {
			vmErr = types.ErrVMExecution.Wrapf("unauthorized, %s is not allowed to receive funds", to)
			return
		}

		// transfer the amount
		if err := k.bankKeeper.SendCoins(ctx, sender.Bytes(), receiver, sdk.NewCoins(sendAmount)); err != nil {
			vmErr = types.ErrVMExecution.Wrapf("failed to transfer %s from %s to %s: %v", sendAmount.String(), sender, to, err)
			return
		}

		// Fire the ERC-20 Transfer event
		bzData, err := abi.Arguments{
			eventTransfer.Inputs[2],
		}.Pack(amount)
		if err != nil {
			vmErr = types.ErrVMExecution.Wrapf("failed to pack output data")
			return
		}

		stateDB.AddLog(&ethtypes.Log{
			Address: virtualFrontierContract.ContractAddress(),
			Topics: []common.Hash{
				common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"), // keccak256 of `Transfer(address,address,uint256)`
				sender.Hash(),
				to.Hash(),
			},
			Data:        bzData,
			BlockNumber: uint64(ctx.BlockHeight()),
		})

		return
	default:
		panic("unreachable")
	}

	return
}
