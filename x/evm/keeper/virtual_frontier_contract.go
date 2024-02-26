package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/ethermint/x/evm/types"
	"strings"
)

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

	if vfContract.Address != contractAddress.String() {
		return sdkerrors.ErrUnknownAddress.Wrapf("contract address %s does not match the address in the contract %s", contractAddress, vfContract.Address)
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

// DeployNewVirtualFrontierContract deploys a new virtual frontier contract into the store
func (k Keeper) DeployNewVirtualFrontierContract(ctx sdk.Context, contractAddress common.Address, vfContract *types.VirtualFrontierContract) error {
	// TODO VFC: check if the contract has code

	params := k.GetParams(ctx)
	for _, existingAddr := range params.VirtualFrontierContractsAddress() {
		if existingAddr == contractAddress {
			return sdkerrors.ErrInvalidRequest.Wrapf("virtual frontier contract %s is already registered in params", contractAddress)
		}
	}

	if k.GetVirtualFrontierContract(ctx, contractAddress) != nil {
		return sdkerrors.ErrInvalidRequest.Wrapf("virtual frontier contract %s already exists", contractAddress)
	}

	var err error

	// register new contract address to params store
	params.VirtualFrontierContracts = append(params.VirtualFrontierContracts, strings.ToLower(contractAddress.String()))

	err = k.SetParams(ctx, params)
	if err != nil {
		return err
	}

	// register new contract metadata to store
	err = k.SetVirtualFrontierContract(ctx, contractAddress, vfContract)
	if err != nil {
		return err
	}

	// TODO VFC: deploy contract code

	return nil
}
