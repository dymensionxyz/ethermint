package keeper

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/evmos/ethermint/x/evm/types"
	"github.com/pkg/errors"
	"math/big"
)

// evmCallVirtualFrontierBankContract handles EVM call to a virtual frontier bank contract.
func (k *Keeper) evmCallVirtualFrontierBankContract(
	ctx sdk.Context,
	stateDB vm.StateDB,
	sender common.Address, virtualFrontierContract *types.VirtualFrontierContract, calldata []byte, gas uint64, value *big.Int,
) *types.VFCExecutionResult {
	compiledVFContract := types.VFBankContract20

	if virtualFrontierContract.Type != types.VFC_TYPE_BANK {
		return types.NewExecVFCError(fmt.Errorf("not a bank contract"))
	}

	// prohibit normal transfer to the bank contract
	if len(calldata) < 1 {
		return types.NewExecVFCRevert(
			0, types.ErrProhibitedAccessingVirtualFrontierContract.Wrap("not allowed to receive"),
		)
	}

	// prohibit transfer native token to the VF contract
	if value != nil && value.Sign() != 0 {
		return types.NewExecVFCRevert(
			0, types.ErrProhibitedAccessingVirtualFrontierContract.Wrap("not allowed to receive"),
		)
	}

	var bankContractMetadata types.VFBankContractMetadata
	if err := k.cdc.Unmarshal(virtualFrontierContract.Metadata, &bankContractMetadata); err != nil {
		return types.NewExecVFCError(
			fmt.Errorf("failed to unmarshal virtual frontier bank contract metadata: %v", err),
		)
	}

	if len(bankContractMetadata.MinDenom) == 0 {
		return types.NewExecVFCError(
			fmt.Errorf("virtual frontier bank contract metadata, denom is empty"),
		)
	}

	bankDenomMetadata, found := k.bankKeeper.GetDenomMetaData(ctx, bankContractMetadata.MinDenom)
	if !found {
		return types.NewExecVFCError(
			fmt.Errorf("bank denom metadata not found for %s", bankContractMetadata.MinDenom),
		)
	}

	method, found := bankContractMetadata.GetMethodFromSignature(calldata)
	if !found {
		// treat as fallback function that does nothing
		return types.NewExecVFCSuccess([]byte{}, 0)
	}

	vfbcDenomMetadata, _ /*ignore invalid state of bank denom-metadata*/ := types.CollectMetadataForVirtualFrontierBankContract(bankDenomMetadata)

	switch method {
	case types.VFBCmName:
		const opGasCost = types.VFBCopgName
		const opGasCostOnRevert = types.VFBCopgName_Revert
		if gas < opGasCost {
			return types.NewExecVFCOutOfGas()
		}

		if len(calldata) != 4 {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("invalid call data"))
		}

		bz, err := compiledVFContract.PackOutput("name", vfbcDenomMetadata.Name)

		if err != nil {
			return types.NewExecVFCError(err)
		}

		return types.NewExecVFCSuccess(bz, opGasCost)
	case types.VFBCmSymbol:
		const opGasCost = types.VFBCopgSymbol
		const opGasCostOnRevert = types.VFBCopgSymbol_Revert
		if gas < opGasCost {
			return types.NewExecVFCOutOfGas()
		}

		if len(calldata) != 4 {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("invalid call data"))
		}

		bz, err := compiledVFContract.PackOutput("symbol", vfbcDenomMetadata.Symbol)

		if err != nil {
			return types.NewExecVFCError(err)
		}

		return types.NewExecVFCSuccess(bz, opGasCost)
	case types.VFBCmDecimals:
		const opGasCost = types.VFBCopgDecimals
		const opGasCostOnRevert = types.VFBCopgDecimals_Revert
		if gas < opGasCost {
			return types.NewExecVFCOutOfGas()
		}

		if len(calldata) != 4 {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("invalid call data"))
		}

		if !vfbcDenomMetadata.CanDecimalsUint8() {
			return types.NewExecVFCRevert(opGasCostOnRevert, fmt.Errorf("decimals overflow %d", vfbcDenomMetadata.Decimals))
		}

		bz, err := compiledVFContract.PackOutput("decimals", uint8(vfbcDenomMetadata.Decimals))

		if err != nil {
			return types.NewExecVFCError(err)
		}

		return types.NewExecVFCSuccess(bz, opGasCost)
	case types.VFBCmTotalSupply:
		const opGasCost = types.VFBCopgTotalSupply
		const opGasCostOnRevert = types.VFBCopgTotalSupply_Revert
		if gas < opGasCost {
			return types.NewExecVFCOutOfGas()
		}

		if len(calldata) != 4 {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("invalid call data"))
		}

		totalSupply := k.bankKeeper.GetSupply(ctx, bankContractMetadata.MinDenom)

		bz, err := compiledVFContract.PackOutput("totalSupply", totalSupply.Amount.BigInt())

		if err != nil {
			return types.NewExecVFCError(err)
		}

		return types.NewExecVFCSuccess(bz, opGasCost)
	case types.VFBCmBalanceOf:
		const opGasCost = types.VFBCopgBalanceOf
		const opGasCostOnRevert = types.VFBCopgBalanceOf_Revert
		if gas < opGasCost {
			return types.NewExecVFCOutOfGas()
		}

		if len(calldata) != 4 /*4bytes sig*/ +32 /*address*/ {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("invalid call data"))
		}

		// unpack the calldata
		inputs, err := compiledVFContract.UnpackInput("balanceOf", calldata[4:])

		if err != nil {
			return types.NewExecVFCError(err)
		}

		if len(inputs) != 1 {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("invalid input"))
		}

		receiverAddress, ok := inputs[0].(common.Address)
		if !ok {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("first input is not an address"))
		}

		// get the balance of the address
		balance := k.bankKeeper.GetBalance(ctx, receiverAddress.Bytes(), bankContractMetadata.MinDenom)

		// pack the output

		bz, err := compiledVFContract.PackOutput("balanceOf", balance.Amount.BigInt())

		if err != nil {
			return types.NewExecVFCError(err)
		}

		return types.NewExecVFCSuccess(bz, opGasCost)
	case types.VFBCmTransfer:
		const opGasCost = types.VFBCopgTransfer
		const opGasCostOnRevert = types.VFBCopgTransfer_Revert
		if gas < opGasCost {
			return types.NewExecVFCOutOfGas()
		}

		if len(calldata) != 4 /*4bytes sig*/ +32 /*address*/ +32 /*amount*/ {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("invalid call data"))
		}

		eventTransfer, foundEvent := compiledVFContract.ABI.Events["Transfer"]
		if !foundEvent {
			return types.NewExecVFCError(errors.New("event Transfer could not be found"))
		}

		// unpack the call-data
		inputs, err := compiledVFContract.UnpackInput("transfer", calldata[4:])

		if err != nil {
			return types.NewExecVFCError(err)
		}

		if len(inputs) != 2 {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("invalid input"))
		}

		to, ok := inputs[0].(common.Address)
		if !ok {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("first input is not an address"))
		}

		amount, ok := inputs[1].(*big.Int)
		if !ok {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("second input is not a number"))
		}

		receiver := sdk.AccAddress(to.Bytes())

		// prohibit transfer to some types of account

		// - module account
		accountI := k.accountKeeper.GetAccount(ctx, receiver)
		if accountI != nil {
			_, isModuleAccount := accountI.(authtypes.ModuleAccountI)
			if isModuleAccount {
				return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("can not transfer to module account"))
			}
		}
		// - VF contracts
		if k.IsVirtualFrontierContract(ctx, to) {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("not allowed to receive"))
		}

		senderBalance := k.bankKeeper.GetBalance(ctx, sender.Bytes(), bankContractMetadata.MinDenom)

		sendAmount := sdk.NewCoin(bankContractMetadata.MinDenom, sdk.NewIntFromBigInt(amount))
		/*
			The line above also checks if the amount is negative and if it has more than 256 bits.
			But let's do explicitly check just for safety, prevent any future issue due to SDK change,
			and also to make the code look more safety.
		*/
		if sendAmount.Amount.IsNegative() {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("transfer amount is negative"))
		}
		if sendAmount.Amount.BigInt().BitLen() > 256 {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("transfer amount exceeds 256 bits"))
		}

		if senderBalance.Amount.LT(sendAmount.Amount) {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New("ERC20: transfer amount exceeds balance"))
		}

		if err := k.bankKeeper.IsSendEnabledCoins(ctx, sendAmount); err != nil {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New(err.Error()))
		}

		if k.bankKeeper.BlockedAddr(receiver) {
			return types.NewExecVFCRevert(opGasCostOnRevert, errors.New(fmt.Sprintf("unauthorized, %s is not allowed to receive funds", to)))
		}

		// Prepare to fire the ERC-20 Transfer event
		bzData, err := abi.Arguments{
			eventTransfer.Inputs[2],
		}.Pack(amount)
		if err != nil {
			if err != nil {
				return types.NewExecVFCError(err)
			}
		}

		// The rest code are state changed, can not be reverted

		// transfer the amount
		if err := k.bankKeeper.SendCoins(ctx, sender.Bytes(), receiver, sdk.NewCoins(sendAmount)); err != nil {
			return types.NewExecVFCRevert(opGasCost, errors.Wrap(err, "failed to transfer"))
		}

		// Fire the ERC-20 Transfer event
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

		return types.NewExecVFCSuccessWithRetBool(true, opGasCost)
	case types.VFBCmApprove_NotSupported, types.VFBCmTransferFrom_NotSupported, types.VFBCmAllowance_NotSupported:
		const opGasCost = types.VFBCopgOtherNotSupported
		if gas < opGasCost {
			return types.NewExecVFCOutOfGas()
		}

		return types.NewExecVFCRevert(opGasCost, errors.New("not supported method"))
	default:
		panic("unreachable")
	}
}
