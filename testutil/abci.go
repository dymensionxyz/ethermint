package testutil

import (
	"time"

	errorsmod "cosmossdk.io/errors"
	errorstypes "github.com/cosmos/cosmos-sdk/types/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"

	sdkmath "cosmossdk.io/math"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/evmos/ethermint/app"
	"github.com/evmos/ethermint/encoding"
	"github.com/evmos/ethermint/testutil/tx"
)

// DeliverTx delivers a cosmos tx for a given set of msgs
func DeliverTx(
	ctx sdk.Context,
	appEvmos *app.EthermintApp,
	priv cryptotypes.PrivKey,
	gasPrice *sdkmath.Int,
	msgs ...sdk.Msg,
) (abci.ResponseDeliverTx, error) {
	txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig
	tx, err := tx.PrepareCosmosTx(
		ctx,
		appEvmos,
		tx.CosmosTxArgs{
			TxCfg:    txConfig,
			Priv:     priv,
			ChainID:  ctx.ChainID(),
			Gas:      10_000_000,
			GasPrice: gasPrice,
			Msgs:     msgs,
		},
	)
	if err != nil {
		return abci.ResponseDeliverTx{}, err
	}
	return BroadcastTxBytes(appEvmos, txConfig.TxEncoder(), tx)
}

// DeliverEthTx generates and broadcasts a Cosmos Tx populated with MsgEthereumTx messages.
// If a private key is provided, it will attempt to sign all messages with the given private key,
// otherwise, it will assume the messages have already been signed.
func DeliverEthTx(
	appEvmos *app.EthermintApp,
	priv cryptotypes.PrivKey,
	msgs ...sdk.Msg,
) (abci.ResponseDeliverTx, error) {
	txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig

	tx, err := tx.PrepareEthTx(txConfig, appEvmos, priv, msgs...)
	if err != nil {
		return abci.ResponseDeliverTx{}, err
	}
	return BroadcastTxBytes(appEvmos, txConfig.TxEncoder(), tx)
}

// Commit commits a block at a given time. Reminder: At the end of each
// Tendermint Consensus round the following methods are run
//  1. BeginBlock
//  2. DeliverTx
//  3. EndBlock
//  4. Commit
func Commit(ctx sdk.Context, app *app.EthermintApp, t time.Duration, vs *tmtypes.ValidatorSet) (sdk.Context, error) {
	header := ctx.BlockHeader()

	if vs != nil {
		res := app.EndBlock(abci.RequestEndBlock{Height: header.Height})

		nextVals, err := applyValSetChanges(vs, res.ValidatorUpdates)
		if err != nil {
			return ctx, err
		}
		header.ValidatorsHash = vs.Hash()
		header.NextValidatorsHash = nextVals.Hash()
	} else {
		app.EndBlocker(ctx, abci.RequestEndBlock{Height: header.Height})
	}

	_ = app.Commit()

	header.Height++
	header.Time = header.Time.Add(t)
	header.AppHash = app.LastCommitID().Hash

	app.BeginBlock(abci.RequestBeginBlock{
		Header: header,
	})

	return ctx.WithBlockHeader(header), nil
}

// applyValSetChanges takes in tmtypes.ValidatorSet and []abci.ValidatorUpdate and will return a new tmtypes.ValidatorSet which has the
// provided validator updates applied to the provided validator set.
func applyValSetChanges(valSet *tmtypes.ValidatorSet, valUpdates []abci.ValidatorUpdate) (*tmtypes.ValidatorSet, error) {
	updates, err := tmtypes.PB2TM.ValidatorUpdates(valUpdates)
	if err != nil {
		return nil, err
	}

	// must copy since validator set will mutate with UpdateWithChangeSet
	newVals := valSet.Copy()
	err = newVals.UpdateWithChangeSet(updates)
	if err != nil {
		return nil, err
	}

	return newVals, nil
}

// BroadcastTxBytes encodes a transaction and calls DeliverTx on the app.
func BroadcastTxBytes(app *app.EthermintApp, txEncoder sdk.TxEncoder, tx sdk.Tx) (abci.ResponseDeliverTx, error) {
	// bz are bytes to be broadcasted over the network
	bz, err := txEncoder(tx)
	if err != nil {
		return abci.ResponseDeliverTx{}, err
	}

	req := abci.RequestDeliverTx{Tx: bz}
	res := app.BaseApp.DeliverTx(req)
	if res.Code != 0 {
		return abci.ResponseDeliverTx{}, errorsmod.Wrapf(errorstypes.ErrInvalidRequest, res.Log)
	}

	return res, nil
}
