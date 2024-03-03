package evm_test

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	etherminttypes "github.com/evmos/ethermint/types"
	"github.com/evmos/ethermint/x/evm"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/evmos/ethermint/x/evm/types"
)

func (suite *EvmTestSuite) TestInitGenesis() {
	privkey, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)

	address := common.HexToAddress(privkey.PubKey().Address().String())

	var vmdb *statedb.StateDB

	testCases := []struct {
		name     string
		malleate func()
		genState *types.GenesisState
		expPanic bool
	}{
		{
			"default",
			func() {},
			types.DefaultGenesisState(),
			false,
		},
		{
			"valid account",
			func() {
				vmdb.AddBalance(address, big.NewInt(1))
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
						Storage: types.Storage{
							{Key: common.BytesToHash([]byte("key")).String(), Value: common.BytesToHash([]byte("value")).String()},
						},
					},
				},
			},
			false,
		},
		{
			"account not found",
			func() {},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
					},
				},
			},
			true,
		},
		{
			"invalid account type",
			func() {
				acc := authtypes.NewBaseAccountWithAddress(address.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
					},
				},
			},
			true,
		},
		{
			"invalid code hash",
			func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, address.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
						Code:    "ffffffff",
					},
				},
			},
			true,
		},
		{
			"ignore empty account code checking",
			func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, address.Bytes())

				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
						Code:    "",
					},
				},
			},
			false,
		},
		{
			"ignore empty account code checking with non-empty codehash",
			func() {
				ethAcc := &etherminttypes.EthAccount{
					BaseAccount: authtypes.NewBaseAccount(address.Bytes(), nil, 0, 0),
					CodeHash:    common.BytesToHash([]byte{1, 2, 3}).Hex(),
				}

				suite.app.AccountKeeper.SetAccount(suite.ctx, ethAcc)
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
						Code:    "",
					},
				},
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset values
			vmdb = suite.StateDB()

			tc.malleate()
			vmdb.Commit()

			if tc.expPanic {
				suite.Require().Panics(
					func() {
						_ = evm.InitGenesis(suite.ctx, suite.app.EvmKeeper, suite.app.AccountKeeper, suite.app.BankKeeper, *tc.genState)
					},
				)
			} else {
				suite.Require().NotPanics(
					func() {
						_ = evm.InitGenesis(suite.ctx, suite.app.EvmKeeper, suite.app.AccountKeeper, suite.app.BankKeeper, *tc.genState)
					},
				)
			}
		})
	}

	originalCtx := suite.ctx
	testCasesProhibitEnableCreate := []struct {
		name      string
		chainId   string
		wantPanic bool
	}{
		{
			name:      "Dymension Mainnet",
			chainId:   "dymension_1100-1",
			wantPanic: true,
		},
		{
			name:      "Dymension Testnet Blumbus",
			chainId:   "blumbus_111-1",
			wantPanic: true,
		},
		{
			name:      "Dymension Devnet Froopyland",
			chainId:   "froopyland_100-1",
			wantPanic: true,
		},
		{
			name:      "Ethermint Devnet",
			chainId:   "ethermint_9000-1",
			wantPanic: false,
		},
	}
	for _, tt := range testCasesProhibitEnableCreate {
		suite.Run(tt.name, func() {
			suite.SetupTest()
			suite.ctx = suite.ctx.WithChainID(tt.chainId)

			genesisState := types.DefaultGenesisState()
			suite.Require().True(genesisState.Params.EnableCreate, "Enable create must be enabled by default")

			var ctx sdk.Context
			if tt.chainId != "" {
				ctx = suite.ctx.WithChainID(tt.chainId)
			} else {
				ctx = originalCtx
			}

			init := func() {
				_ = evm.InitGenesis(ctx, suite.app.EvmKeeper, suite.app.AccountKeeper, suite.app.BankKeeper, *genesisState)
			}

			if tt.wantPanic {
				defer func() {
					err := recover()
					suite.Require().NotNil(err, "expected panic")
					suite.Require().Contains(fmt.Sprintf("%v", err), "enable create is not allowed on Dymension chains")
				}()

				init()
			} else {
				suite.Require().NotPanics(init)
			}
		})
	}
}
