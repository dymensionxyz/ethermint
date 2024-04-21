package types

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v6/modules/core/keeper"
	"github.com/cosmos/ibc-go/v6/testing/types"
	chainapp "github.com/evmos/ethermint/app"
	"github.com/evmos/ethermint/encoding"
	abci "github.com/tendermint/tendermint/abci/types"
)

func (c chainAppImp) Info(info abci.RequestInfo) abci.ResponseInfo {
	return c.app.BaseApp.Info(info)
}

func (c chainAppImp) SetOption(option abci.RequestSetOption) abci.ResponseSetOption {
	return c.app.BaseApp.SetOption(option)
}

func (c chainAppImp) Query(query abci.RequestQuery) abci.ResponseQuery {
	return c.app.BaseApp.Query(query)
}

func (c chainAppImp) CheckTx(tx abci.RequestCheckTx) abci.ResponseCheckTx {
	return c.app.BaseApp.CheckTx(tx)
}

func (c chainAppImp) InitChain(chain abci.RequestInitChain) abci.ResponseInitChain {
	return c.app.BaseApp.InitChain(chain)
}

func (c chainAppImp) BeginBlock(block abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return c.app.BaseApp.BeginBlock(block)
}

func (c chainAppImp) DeliverTx(tx abci.RequestDeliverTx) abci.ResponseDeliverTx {
	return c.app.BaseApp.DeliverTx(tx)
}

func (c chainAppImp) EndBlock(block abci.RequestEndBlock) abci.ResponseEndBlock {
	return c.app.BaseApp.EndBlock(block)
}

func (c chainAppImp) Commit() abci.ResponseCommit {
	return c.app.BaseApp.Commit()
}

func (c chainAppImp) ListSnapshots(snapshots abci.RequestListSnapshots) abci.ResponseListSnapshots {
	return c.app.BaseApp.ListSnapshots(snapshots)
}

func (c chainAppImp) OfferSnapshot(snapshot abci.RequestOfferSnapshot) abci.ResponseOfferSnapshot {
	return c.app.BaseApp.OfferSnapshot(snapshot)
}

func (c chainAppImp) LoadSnapshotChunk(chunk abci.RequestLoadSnapshotChunk) abci.ResponseLoadSnapshotChunk {
	return c.app.BaseApp.LoadSnapshotChunk(chunk)
}

func (c chainAppImp) ApplySnapshotChunk(chunk abci.RequestApplySnapshotChunk) abci.ResponseApplySnapshotChunk {
	return c.app.BaseApp.ApplySnapshotChunk(chunk)
}

func (c chainAppImp) GetBaseApp() *baseapp.BaseApp {
	return c.app.BaseApp
}

func (c chainAppImp) GetStakingKeeper() types.StakingKeeper {
	return c.app.StakingKeeper
}

func (c chainAppImp) GetIBCKeeper() *ibckeeper.Keeper {
	return c.app.IBCKeeper
}

func (c chainAppImp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return c.app.ScopedIBCKeeper
}

func (c chainAppImp) GetTxConfig() client.TxConfig {
	return encoding.MakeConfig(chainapp.ModuleBasics).TxConfig
}

func (c chainAppImp) AppCodec() codec.Codec {
	return c.app.AppCodec()
}

func (c chainAppImp) LastCommitID() storetypes.CommitID {
	return c.app.LastCommitID()
}

func (c chainAppImp) LastBlockHeight() int64 {
	return c.app.LastBlockHeight()
}
