package evm

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/evmos/ethermint/x/evm/keeper"
	"github.com/evmos/ethermint/x/evm/types"
	"strings"
)

// NewEvmProposalHandler creates a governance handler to manage new proposal types.
func NewEvmProposalHandler(k *keeper.Keeper) govv1beta1.Handler {
	return func(ctx sdk.Context, content govv1beta1.Content) error {
		switch c := content.(type) {
		case *types.UpdateVirtualFrontierBankContractsProposal:
			return handleUpdateVirtualFrontierBankContractsProposal(ctx, k, c)
		default:
			return errorsmod.Wrapf(errortypes.ErrUnknownRequest, "unrecognized %s proposal content type: %T", types.ModuleName, c)
		}
	}
}

// handleUpdateVirtualFrontierBankContractsProposal handles the proposal to update the virtual frontier bank contracts
func handleUpdateVirtualFrontierBankContractsProposal(
	ctx sdk.Context,
	k *keeper.Keeper,
	p *types.UpdateVirtualFrontierBankContractsProposal,
) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	contractsAddress, err := k.UpdateVirtualFrontierBankContracts(ctx, p.Contracts...)
	if err != nil {
		return err
	}

	for _, contractAddress := range contractsAddress {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeVirtualFrontierContract,
				sdk.NewAttribute(types.AttributeKeyVFAction, "update"),
				sdk.NewAttribute(types.AttributeKeyVFType, "bank"),
				sdk.NewAttribute(types.AttributeKeyVFAddress, strings.ToLower(contractAddress.String())),
			),
		)
	}

	return nil
}
