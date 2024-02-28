package keeper

import (
	"bytes"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/evmos/ethermint/x/evm/types"
	"strings"
)

// IsVirtualFrontierContract returns true if the address is a virtual frontier contract address
func (k Keeper) IsVirtualFrontierContract(ctx sdk.Context, address common.Address) bool {
	store := ctx.KVStore(k.storeKey)

	key := types.VirtualFrontierContractKey(address)

	return store.Has(key)
}

// GetVirtualFrontierContract returns the virtual frontier contract from the store, or nil if not found
func (k Keeper) GetVirtualFrontierContract(ctx sdk.Context, contractAddress common.Address) *types.VirtualFrontierContract {
	store := ctx.KVStore(k.storeKey)

	key := types.VirtualFrontierContractKey(contractAddress)

	bz := store.Get(key)
	if len(bz) == 0 {
		return nil
	}

	var vfContract types.VirtualFrontierContract
	k.cdc.MustUnmarshal(bz, &vfContract)

	return &vfContract
}

// SetVirtualFrontierContract registers/override a virtual frontier contract into the store
func (k Keeper) SetVirtualFrontierContract(ctx sdk.Context, contractAddress common.Address, vfContract *types.VirtualFrontierContract) error {
	if err := vfContract.ValidateBasic(k.cdc); err != nil {
		return err
	}

	if vfContract.Address != strings.ToLower(contractAddress.String()) {
		return sdkerrors.ErrUnknownAddress.Wrapf("contract address %s does not match the address in the contract %s", strings.ToLower(contractAddress.String()), vfContract.Address)
	}

	store := ctx.KVStore(k.storeKey)

	bz, err := k.cdc.Marshal(vfContract)
	if err != nil {
		return err
	}

	key := types.VirtualFrontierContractKey(contractAddress)

	store.Set(key, bz)
	return nil
}

// GetVirtualFrontierBankContractAddressByDenom returns the virtual frontier bank contract address by denom or nil if not found.
func (k Keeper) GetVirtualFrontierBankContractAddressByDenom(ctx sdk.Context, minDenom string) (contractAddress common.Address, found bool) {
	if minDenom == "" {
		panic("invalid parameter")
	}

	store := ctx.KVStore(k.storeKey)

	key := types.VirtualFrontierBankContractAddressByDenomKey(minDenom)

	bz := store.Get(key)
	if len(bz) == 0 {
		found = false
		return
	}

	contractAddress = common.BytesToAddress(bz)
	found = true
	return
}

// SetMappingVirtualFrontierBankContractAddressByDenom registers a virtual frontier contract into the store.
// Override is not allowed and returns error.
func (k Keeper) SetMappingVirtualFrontierBankContractAddressByDenom(ctx sdk.Context, minDenom string, contractAddress common.Address) error {
	if minDenom == "" || contractAddress == (common.Address{}) {
		panic("invalid parameter")
	}

	existingContractAddress, found := k.GetVirtualFrontierBankContractAddressByDenom(ctx, minDenom)
	if found {
		return sdkerrors.ErrConflict.Wrapf("mapping virtual frontier bank contract for denom %s had been registered before at %s", minDenom, existingContractAddress)
	}

	store := ctx.KVStore(k.storeKey)

	key := types.VirtualFrontierBankContractAddressByDenomKey(minDenom)

	store.Set(key, contractAddress.Bytes())
	return nil
}

// DeployNewVirtualFrontierBankContract deploys a new virtual frontier bank contract into the store
func (k Keeper) DeployNewVirtualFrontierBankContract(ctx sdk.Context, vfContract *types.VirtualFrontierContract, bankMeta *types.VFBankContractMetadata) (common.Address, error) {
	vfContract.Type = uint32(types.VirtualFrontierContractTypeBankContract)
	vfContract.Metadata = k.cdc.MustMarshal(bankMeta)

	callData, err := PrepareBytecodeForVirtualFrontierBankContractDeployment(bankMeta.DisplayName, uint8(bankMeta.Exponent))
	if err != nil {
		return common.Address{}, err
	}

	contractAddress, err := k.DeployNewVirtualFrontierContract(ctx, vfContract, callData)
	if err != nil {
		return common.Address{}, err
	}

	// register mapping by denom
	err = k.SetMappingVirtualFrontierBankContractAddressByDenom(ctx, bankMeta.MinDenom, contractAddress)
	if err != nil {
		return common.Address{}, err
	}

	return contractAddress, nil
}

func PrepareBytecodeForVirtualFrontierBankContractDeployment(displayName string, exponent uint8) ([]byte, error) {
	// method is exposed to be re-use in test
	ctorArgs, err := types.VFBankContract20.ABI.Pack(
		"",
		displayName,
		displayName,
		exponent,
	)

	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrPackAny, "failed to pack bytecode %s", err.Error())
	}

	contractBytecode := types.VFBankContract20.Bin

	var callData []byte
	callData = append(callData, contractBytecode...)
	callData = append(callData, ctorArgs...)

	return callData, nil
}

// DeployNewVirtualFrontierContract deploys a new virtual frontier contract into the store
func (k Keeper) DeployNewVirtualFrontierContract(ctx sdk.Context, vfContract *types.VirtualFrontierContract, callData []byte) (contractAddress common.Address, err error) {
	defer func() {
		if err != nil {
			contractAddress = common.Address{}
		}
	}()

	if len(vfContract.Address) > 0 {
		err = sdkerrors.ErrInvalidRequest.Wrapf("input contract address must be empty")
		return
	}

	deployerModuleAccount := k.accountKeeper.GetModuleAccount(ctx, types.ModuleVirtualFrontierContractDeployerName)
	if deployerModuleAccount == nil {
		err = sdkerrors.ErrNotFound.Wrapf("module account %s does not exist", types.ModuleVirtualFrontierContractDeployerName)
		return
	}

	nonce := deployerModuleAccount.GetSequence()
	contractAddress = crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, nonce)
	contractAccount := k.GetAccount(ctx, contractAddress)
	if contractAccount != nil {
		if len(contractAccount.CodeHash) > 0 {
			if bytes.Equal(contractAccount.CodeHash, types.EmptyCodeHash) {
				err = sdkerrors.ErrInvalidRequest.Wrapf("contract address already exists at %s", contractAddress)
				return
			}
		}
	}

	if k.IsVirtualFrontierContract(ctx, contractAddress) {
		err = sdkerrors.ErrInvalidRequest.Wrapf("virtual frontier contract %s already exists", contractAddress)
		return
	}

	// deploy pseudo bytecode for virtual frontier contract.
	//
	// The VF contract is not accessible from the EVM,
	// can only be accessed by calling it directly via ETH wallets or other means.
	// But to make the state consistency and prevent as much of the potential issues,
	// we still need to deploy a pseudo set of bytecode by asking the EVM to deploy it so the contract is actually exists.
	//
	// The pseudo bytecode is actual EVM bytecode, compiled from some real solidity contracts,
	// can read the contracts by checking the corresponding file:
	//  - VF Bank contract: x/evm/types/VFBankContract20.sol (the code is real ERC-20 interface, but the implementation always returns error upon invoking any function)
	if ctx.BlockHeight() == 0 {
		// can not deploy contract code in genesis, so we just store the contract metadata
		// and increase the sequence number of the deployer account so next deployment will generate different address.
		deployerModuleAccount.SetSequence(nonce + 1)
		k.accountKeeper.SetAccount(ctx, deployerModuleAccount)
	} else {
		if len(callData) == 0 {
			err = sdkerrors.ErrInvalidRequest.Wrapf("input call data must not be empty")
			return
		}

		msg := ethtypes.NewMessage(
			types.VirtualFrontierContractDeployerAddress,
			nil,
			nonce,
			common.Big0, // amount
			3_000_000,   // gasLimit
			common.Big0, // gasPrice
			common.Big0, // gasFeeCap
			common.Big0, // gasTipCap
			callData,
			ethtypes.AccessList{},
			false,
		)

		cfg, errGetEvmConfig := k.EVMConfig(ctx, ctx.BlockHeader().ProposerAddress, k.eip155ChainID)
		if errGetEvmConfig != nil {
			err = errorsmod.Wrapf(types.ErrVMExecution, "failed to load evm config: %v", errGetEvmConfig)
			return
		}
		if !cfg.Params.EnableCreate {
			// enable contract creation for this run in-case of disabled, this change is not persisted
			copiedParams := cfg.Params
			copiedParams.EnableCreate = true
			cfg.Params = copiedParams
		}

		txConfig := statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))

		res, errApplyMsg := k.ApplyMessageWithConfig(ctx, msg, types.NewNoOpTracer(), true, cfg, txConfig)
		if errApplyMsg != nil {
			err = errorsmod.Wrap(types.ErrVMExecution, errApplyMsg.Error())
			return
		}

		if res.Failed() {
			err = errorsmod.Wrap(types.ErrVMExecution, res.VmError)
			return
		}
	}

	// register new contract metadata to store
	vfContract.Address = strings.ToLower(contractAddress.String())
	err = k.SetVirtualFrontierContract(ctx, contractAddress, vfContract)
	if err != nil {
		return
	}

	// fire Tendermint events
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeVirtualFrontierContract,
			sdk.NewAttribute(types.AttributeKeyVFAction, "deploy"),
			sdk.NewAttribute(types.AttributeKeyVFType, vfContract.GetTypeName()),
			sdk.NewAttribute(types.AttributeKeyVFAddress, strings.ToLower(contractAddress.String())),
		),
	)

	return
}
