package ante

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// RejectMessagesDecorator prevents invalid msg types from being executed
type RejectMessagesDecorator struct {
	disabledMsgTypeURLs []string
}

var _ sdk.AnteDecorator = RejectMessagesDecorator{}

// NewRejectMessagesDecorator creates a decorator to block vesting messages from reaching the mempool
func NewRejectMessagesDecorator(msgs []string) RejectMessagesDecorator {
	return RejectMessagesDecorator{
		disabledMsgTypeURLs: msgs,
	}
}

// AnteHandle rejects messages that requires ethereum-specific authentication.
// For example `MsgEthereumTx` requires fee to be deducted in the antehandler in
// order to perform the refund.
func (rmd RejectMessagesDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	for _, msg := range tx.GetMsgs() {
		typeURL := sdk.MsgTypeURL(msg)
		for _, disabledTypeURL := range rmd.disabledMsgTypeURLs {
			if typeURL == disabledTypeURL {
				return ctx, errorsmod.Wrapf(
					sdkerrors.ErrUnauthorized,
					"MsgTypeURL %s not supported",
					typeURL,
				)
			}
		}
	}
	return next(ctx, tx, simulate)
}
