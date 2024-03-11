package keeper_test

import (
	"errors"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/evmos/ethermint/x/evm/keeper"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/evmos/ethermint/x/evm/types"
	"github.com/evmos/ethermint/x/evm/vm/geth"
)

// LogRecordHook records all the logs
type LogRecordHook struct {
	Logs []*ethtypes.Log
}

func (dh *LogRecordHook) PostTxProcessing(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt) error {
	dh.Logs = receipt.Logs
	return nil
}

// FailureHook always fail
type FailureHook struct{}

func (dh FailureHook) PostTxProcessing(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt) error {
	return errors.New("post tx processing failed")
}

func (suite *KeeperTestSuite) TestEvmHooks() {
	testCases := []struct {
		msg       string
		setupHook func() types.EvmHooks
		expFunc   func(hook types.EvmHooks, result error)
	}{
		{
			"log collect hook",
			func() types.EvmHooks {
				return &LogRecordHook{}
			},
			func(hook types.EvmHooks, result error) {
				suite.Require().NoError(result)
				suite.Require().Equal(1, len((hook.(*LogRecordHook).Logs)))
			},
		},
		{
			"always fail hook",
			func() types.EvmHooks {
				return &FailureHook{}
			},
			func(hook types.EvmHooks, result error) {
				suite.Require().Error(result)
			},
		},
	}

	for _, tc := range testCases {
		suite.SetupTest()
		hook := tc.setupHook()

		k := keeper.NewKeeper(
			suite.app.AppCodec(), suite.app.GetKey(types.StoreKey), suite.app.GetTKey(types.StoreKey), authtypes.NewModuleAddress(govtypes.ModuleName),
			suite.app.AccountKeeper, suite.app.BankKeeper, suite.app.StakingKeeper, suite.app.FeeMarketKeeper,
			nil, geth.NewEVM, "", suite.app.GetSubspace(types.ModuleName),
		)
		k.SetHooks(keeper.NewMultiEvmHooks(hook))

		ctx := suite.ctx
		txHash := common.BigToHash(big.NewInt(1))
		vmdb := statedb.New(ctx, k, statedb.NewTxConfig(
			common.BytesToHash(ctx.HeaderHash().Bytes()),
			txHash,
			0,
			0,
		))

		vmdb.AddLog(&ethtypes.Log{
			Topics:  []common.Hash{},
			Address: suite.address,
		})
		logs := vmdb.Logs()
		receipt := &ethtypes.Receipt{
			TxHash: txHash,
			Logs:   logs,
		}
		result := k.PostTxProcessing(ctx, ethtypes.Message{}, receipt)

		tc.expFunc(hook, result)
	}
}
