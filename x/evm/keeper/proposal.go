package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/ethermint/x/evm/types"
)

// UpdateVirtualFrontierBankContracts update the virtual frontier bank contracts
func (k Keeper) UpdateVirtualFrontierBankContracts(
	ctx sdk.Context,
	contracts ...types.VirtualFrontierBankContractProposalContent,
) ([]common.Address, error) {
	if len(contracts) == 0 {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("no contracts to update")
	}

	var uniqueTracker = make(map[common.Address]bool)

	for _, updateContent := range contracts {

		if err := updateContent.ValidateBasic(); err != nil {
			return nil, err
		}

		contractAddress := common.HexToAddress(updateContent.ContractAddress)

		if _, found := uniqueTracker[common.HexToAddress(updateContent.ContractAddress)]; found {
			return nil, sdkerrors.ErrInvalidRequest.Wrapf("duplicate update for contract address: %s", updateContent.ContractAddress)
		}
		uniqueTracker[contractAddress] = true

		vfContract := k.GetVirtualFrontierContract(ctx, contractAddress)
		if vfContract == nil {
			return nil, sdkerrors.ErrUnknownAddress.Wrapf("virtual frontier contract %s not found", contractAddress.String())
		}

		if vfContract.Type != types.VFC_TYPE_BANK {
			return nil, sdkerrors.ErrInvalidRequest.Wrapf("%s is not a virtual frontier bank contract", contractAddress.String())
		}

		var bankContractMeta types.VFBankContractMetadata
		if err := k.cdc.Unmarshal(vfContract.Metadata, &bankContractMeta); err != nil {
			return nil, sdkerrors.ErrUnpackAny.Wrapf("failed to unmarshal virtual frontier bank contract metadata for %s", contractAddress.String())
		}

		vfContract.Active = updateContent.Active

		bz, err := k.cdc.Marshal(&bankContractMeta)
		if err != nil {
			return nil, sdkerrors.ErrPackAny.Wrapf("failed to marshal virtual frontier bank contract metadata for %s", contractAddress.String())
		}
		vfContract.Metadata = bz

		if err := k.SetVirtualFrontierContract(ctx, contractAddress, vfContract); err != nil {
			return nil, err
		}
	}

	var updatedAddressList []common.Address

	for address := range uniqueTracker {
		updatedAddressList = append(updatedAddressList, address)
	}

	return updatedAddressList, nil
}
