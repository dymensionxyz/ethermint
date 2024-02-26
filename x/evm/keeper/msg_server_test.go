package keeper_test

import (
	"math/big"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/evmos/ethermint/x/evm/types"
)

func (suite *KeeperTestSuite) TestEthereumTx() {
	var (
		err             error
		msg             *types.MsgEthereumTx
		signer          ethtypes.Signer
		vmdb            *statedb.StateDB
		chainCfg        *params.ChainConfig
		expectedGasUsed uint64
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   bool
	}{
		{
			"Deploy contract tx - insufficient gas",
			func() {
				msg, err = suite.createContractMsgTx(
					vmdb.GetNonce(suite.address),
					signer,
					chainCfg,
					big.NewInt(1),
				)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"Transfer funds tx",
			func() {
				msg, _, err = newEthMsgTx(
					vmdb.GetNonce(suite.address),
					suite.ctx.BlockHeight(),
					suite.address,
					chainCfg,
					suite.signer,
					signer,
					ethtypes.AccessListTxType,
					nil,
					nil,
				)
				suite.Require().NoError(err)
				expectedGasUsed = params.TxGas
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			keeperParams := suite.app.EvmKeeper.GetParams(suite.ctx)
			chainCfg = keeperParams.ChainConfig.EthereumConfig(suite.app.EvmKeeper.ChainID())
			signer = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())
			vmdb = suite.StateDB()

			tc.malleate()
			res, err := suite.app.EvmKeeper.EthereumTx(suite.ctx, msg)
			if tc.expErr {
				suite.Require().Error(err)
				return
			}
			suite.Require().NoError(err)
			suite.Require().Equal(expectedGasUsed, res.GasUsed)
			suite.Require().False(res.Failed())
		})
	}
}

func (suite *KeeperTestSuite) TestUpdateParams() {
	testCases := []struct {
		name                               string
		malleate                           func()
		request                            *types.MsgUpdateParams
		expectErr                          bool
		expectVirtualFrontierContractCount int
		postCheck                          func()
	}{
		{
			name:      "fail - invalid authority",
			request:   &types.MsgUpdateParams{Authority: "foobar"},
			expectErr: true,
		},
		{
			name: "pass - valid Update msg",
			request: &types.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params:    types.DefaultParams(),
			},
			expectErr: false,
		},
		{
			name: "pass - valid Update msg without virtual frontier contracts",
			malleate: func() {
				params := types.DefaultParams()
				params.VirtualFrontierContracts = []string{"0x405b96e2538ac85ee862e332fa634b158d013ae1", "0x9ede3180fae6322ea4fc946810152170e833ab1f"}
				suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Commit()
			},
			request: &types.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params:    types.DefaultParams(),
			},
			expectErr:                          false,
			expectVirtualFrontierContractCount: 2,
		},
		{
			name: "fail - invalid Update msg with virtual frontier contracts",
			request: &types.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params: func() types.Params {
					params := types.DefaultParams()
					params.VirtualFrontierContracts = []string{"0x405b96e2538ac85ee862e332fa634b158d013ae1", "0x9ede3180fae6322ea4fc946810152170e833ab1f"}
					return params
				}(),
			},
			expectErr:                          true,
			expectVirtualFrontierContractCount: 2,
		},
		{
			name: "pass - (again) valid Update msg without virtual frontier contracts, old virtual frontier contracts are preserved",
			malleate: func() {
				// ensure that the virtual frontier contracts are set
				currentParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				suite.Require().Equal(
					[]string{"0x405b96e2538ac85ee862e332fa634b158d013ae1", "0x9ede3180fae6322ea4fc946810152170e833ab1f"},
					currentParams.VirtualFrontierContracts,
					"virtual frontier contracts must be set before execution",
				)
			},
			request: &types.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params:    types.DefaultParams(),
			},
			expectErr:                          false,
			expectVirtualFrontierContractCount: 2,
			postCheck: func() {
				// ensure that the virtual frontier contracts are preserved
				newParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				suite.Require().Equal(
					[]string{"0x405b96e2538ac85ee862e332fa634b158d013ae1", "0x9ede3180fae6322ea4fc946810152170e833ab1f"},
					newParams.VirtualFrontierContracts,
					"virtual frontier contracts must be kept after execution",
				)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run("MsgUpdateParams", func() {
			if tc.malleate != nil {
				tc.malleate()
			}

			_, err := suite.app.EvmKeeper.UpdateParams(suite.ctx, tc.request)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)

				params := suite.app.EvmKeeper.GetParams(suite.ctx)
				suite.Require().Len(params.VirtualFrontierContracts, tc.expectVirtualFrontierContractCount)
			}

			if tc.postCheck != nil {
				tc.postCheck()
			}
		})
	}
}
