package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	"fmt"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/evmos/ethermint/utils"
	"math"
	"math/big"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/evmos/ethermint/tests"
	"github.com/evmos/ethermint/x/evm/keeper"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/evmos/ethermint/x/evm/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

func (suite *KeeperTestSuite) TestGetHashFn() {
	header := suite.ctx.BlockHeader()
	h, _ := tmtypes.HeaderFromProto(&header)
	hash := h.Hash()

	testCases := []struct {
		msg      string
		height   uint64
		malleate func()
		expHash  common.Hash
	}{
		{
			"case 1.1: context hash cached",
			uint64(suite.ctx.BlockHeight()),
			func() {
				suite.ctx = suite.ctx.WithHeaderHash(tmhash.Sum([]byte("header")))
			},
			common.BytesToHash(tmhash.Sum([]byte("header"))),
		},
		{
			"case 1.2: failed to cast Tendermint header",
			uint64(suite.ctx.BlockHeight()),
			func() {
				header := tmproto.Header{}
				header.Height = suite.ctx.BlockHeight()
				suite.ctx = suite.ctx.WithBlockHeader(header)
			},
			common.Hash{},
		},
		{
			"case 1.3: hash calculated from Tendermint header",
			uint64(suite.ctx.BlockHeight()),
			func() {
				suite.ctx = suite.ctx.WithBlockHeader(header)
			},
			common.BytesToHash(hash),
		},
		{
			"case 2.1: height lower than current one, hist info not found",
			1,
			func() {
				suite.ctx = suite.ctx.WithBlockHeight(10)
			},
			common.Hash{},
		},
		{
			"case 2.2: height lower than current one, invalid hist info header",
			1,
			func() {
				suite.app.StakingKeeper.SetHistoricalInfo(suite.ctx, 1, &stakingtypes.HistoricalInfo{})
				suite.ctx = suite.ctx.WithBlockHeight(10)
			},
			common.Hash{},
		},
		{
			"case 2.3: height lower than current one, calculated from hist info header",
			1,
			func() {
				histInfo := &stakingtypes.HistoricalInfo{
					Header: header,
				}
				suite.app.StakingKeeper.SetHistoricalInfo(suite.ctx, 1, histInfo)
				suite.ctx = suite.ctx.WithBlockHeight(10)
			},
			common.BytesToHash(hash),
		},
		{
			"case 3: height greater than current one",
			200,
			func() {},
			common.Hash{},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()

			hash := suite.app.EvmKeeper.GetHashFn(suite.ctx)(tc.height)
			suite.Require().Equal(tc.expHash, hash)
		})
	}
}

func (suite *KeeperTestSuite) TestGetCoinbaseAddress() {
	valOpAddr := tests.GenerateAddress()

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"validator not found",
			func() {
				header := suite.ctx.BlockHeader()
				header.ProposerAddress = []byte{}
				suite.ctx = suite.ctx.WithBlockHeader(header)
			},
			false,
		},
		{
			"success",
			func() {
				valConsAddr, privkey := tests.NewAddrKey()

				pkAny, err := codectypes.NewAnyWithValue(privkey.PubKey())
				suite.Require().NoError(err)

				validator := stakingtypes.Validator{
					OperatorAddress: sdk.ValAddress(valOpAddr.Bytes()).String(),
					ConsensusPubkey: pkAny,
				}

				suite.app.StakingKeeper.SetValidator(suite.ctx, validator)
				err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
				suite.Require().NoError(err)

				header := suite.ctx.BlockHeader()
				header.ProposerAddress = valConsAddr.Bytes()
				suite.ctx = suite.ctx.WithBlockHeader(header)

				_, found := suite.app.StakingKeeper.GetValidatorByConsAddr(suite.ctx, valConsAddr.Bytes())
				suite.Require().True(found)

				suite.Require().NotEmpty(suite.ctx.BlockHeader().ProposerAddress)
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			proposerAddress := suite.ctx.BlockHeader().ProposerAddress
			coinbase, err := suite.app.EvmKeeper.GetCoinbaseAddress(suite.ctx, sdk.ConsAddress(proposerAddress))
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(valOpAddr, coinbase)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestGetEthIntrinsicGas() {
	testCases := []struct {
		name               string
		data               []byte
		accessList         ethtypes.AccessList
		height             int64
		isContractCreation bool
		noError            bool
		expGas             uint64
	}{
		{
			"no data, no accesslist, not contract creation, not homestead, not istanbul",
			nil,
			nil,
			1,
			false,
			true,
			params.TxGas,
		},
		{
			"with one zero data, no accesslist, not contract creation, not homestead, not istanbul",
			[]byte{0},
			nil,
			1,
			false,
			true,
			params.TxGas + params.TxDataZeroGas*1,
		},
		{
			"with one non zero data, no accesslist, not contract creation, not homestead, not istanbul",
			[]byte{1},
			nil,
			1,
			true,
			true,
			params.TxGas + params.TxDataNonZeroGasFrontier*1,
		},
		{
			"no data, one accesslist, not contract creation, not homestead, not istanbul",
			nil,
			[]ethtypes.AccessTuple{
				{},
			},
			1,
			false,
			true,
			params.TxGas + params.TxAccessListAddressGas,
		},
		{
			"no data, one accesslist with one storageKey, not contract creation, not homestead, not istanbul",
			nil,
			[]ethtypes.AccessTuple{
				{StorageKeys: make([]common.Hash, 1)},
			},
			1,
			false,
			true,
			params.TxGas + params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas*1,
		},
		{
			"no data, no accesslist, is contract creation, is homestead, not istanbul",
			nil,
			nil,
			2,
			true,
			true,
			params.TxGasContractCreation,
		},
		{
			"with one zero data, no accesslist, not contract creation, is homestead, is istanbul",
			[]byte{1},
			nil,
			3,
			false,
			true,
			params.TxGas + params.TxDataNonZeroGasEIP2028*1,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			params := suite.app.EvmKeeper.GetParams(suite.ctx)
			ethCfg := params.ChainConfig.EthereumConfig(suite.app.EvmKeeper.ChainID())
			ethCfg.HomesteadBlock = big.NewInt(2)
			ethCfg.IstanbulBlock = big.NewInt(3)
			signer := ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())

			suite.ctx = suite.ctx.WithBlockHeight(tc.height)

			nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)
			m, err := newNativeMessage(
				nonce,
				suite.ctx.BlockHeight(),
				suite.address,
				ethCfg,
				suite.signer,
				signer,
				ethtypes.AccessListTxType,
				tc.data,
				tc.accessList,
			)
			suite.Require().NoError(err)

			gas, err := suite.app.EvmKeeper.GetEthIntrinsicGas(suite.ctx, m, ethCfg, tc.isContractCreation)
			if tc.noError {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

			suite.Require().Equal(tc.expGas, gas)
		})
	}
}

func (suite *KeeperTestSuite) TestGasToRefund() {
	testCases := []struct {
		name           string
		gasconsumed    uint64
		refundQuotient uint64
		expGasRefund   uint64
		expPanic       bool
	}{
		{
			"gas refund 5",
			5,
			1,
			5,
			false,
		},
		{
			"gas refund 10",
			10,
			1,
			10,
			false,
		},
		{
			"gas refund availableRefund",
			11,
			1,
			10,
			false,
		},
		{
			"gas refund quotient 0",
			11,
			0,
			0,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.mintFeeCollector = true
			suite.SetupTest() // reset
			vmdb := suite.StateDB()
			vmdb.AddRefund(10)

			if tc.expPanic {
				panicF := func() {
					keeper.GasToRefund(vmdb.GetRefund(), tc.gasconsumed, tc.refundQuotient)
				}
				suite.Require().Panics(panicF)
			} else {
				gr := keeper.GasToRefund(vmdb.GetRefund(), tc.gasconsumed, tc.refundQuotient)
				suite.Require().Equal(tc.expGasRefund, gr)
			}
		})
	}
	suite.mintFeeCollector = false
}

func (suite *KeeperTestSuite) TestRefundGas() {
	var (
		m   core.Message
		err error
	)

	testCases := []struct {
		name           string
		leftoverGas    uint64
		refundQuotient uint64
		noError        bool
		expGasRefund   uint64
		malleate       func()
	}{
		{
			name:           "leftoverGas more than tx gas limit",
			leftoverGas:    params.TxGas + 1,
			refundQuotient: params.RefundQuotient,
			noError:        false,
			expGasRefund:   params.TxGas + 1,
		},
		{
			name:           "leftoverGas equal to tx gas limit, insufficient fee collector account",
			leftoverGas:    params.TxGas,
			refundQuotient: params.RefundQuotient,
			noError:        true,
			expGasRefund:   0,
		},
		{
			name:           "leftoverGas less than to tx gas limit",
			leftoverGas:    params.TxGas - 1,
			refundQuotient: params.RefundQuotient,
			noError:        true,
			expGasRefund:   0,
		},
		{
			name:           "no leftoverGas, refund half used gas ",
			leftoverGas:    0,
			refundQuotient: params.RefundQuotient,
			noError:        true,
			expGasRefund:   params.TxGas / params.RefundQuotient,
		},
		{
			name:           "invalid Gas value in msg",
			leftoverGas:    0,
			refundQuotient: params.RefundQuotient,
			noError:        false,
			expGasRefund:   params.TxGas,
			malleate: func() {
				keeperParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				m, err = suite.createContractGethMsg(
					suite.StateDB().GetNonce(suite.address),
					ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID()),
					keeperParams.ChainConfig.EthereumConfig(suite.app.EvmKeeper.ChainID()),
					big.NewInt(-100),
				)
				suite.Require().NoError(err)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.mintFeeCollector = true
			suite.SetupTest() // reset

			keeperParams := suite.app.EvmKeeper.GetParams(suite.ctx)
			ethCfg := keeperParams.ChainConfig.EthereumConfig(suite.app.EvmKeeper.ChainID())
			signer := ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())
			vmdb := suite.StateDB()

			m, err = newNativeMessage(
				vmdb.GetNonce(suite.address),
				suite.ctx.BlockHeight(),
				suite.address,
				ethCfg,
				suite.signer,
				signer,
				ethtypes.AccessListTxType,
				nil,
				nil,
			)
			suite.Require().NoError(err)

			vmdb.AddRefund(params.TxGas)

			if tc.leftoverGas > m.Gas() {
				return
			}

			if tc.malleate != nil {
				tc.malleate()
			}

			gasUsed := m.Gas() - tc.leftoverGas
			refund := keeper.GasToRefund(vmdb.GetRefund(), gasUsed, tc.refundQuotient)
			suite.Require().Equal(tc.expGasRefund, refund)

			err = suite.app.EvmKeeper.RefundGas(suite.ctx, m, refund, "aphoton")
			if tc.noError {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
	suite.mintFeeCollector = false
}

func (suite *KeeperTestSuite) TestResetGasMeterAndConsumeGas() {
	testCases := []struct {
		name        string
		gasConsumed uint64
		gasUsed     uint64
		expPanic    bool
	}{
		{
			"gas consumed 5, used 5",
			5,
			5,
			false,
		},
		{
			"gas consumed 5, used 10",
			5,
			10,
			false,
		},
		{
			"gas consumed 10, used 10",
			10,
			10,
			false,
		},
		{
			"gas consumed 11, used 10, NegativeGasConsumed panic",
			11,
			10,
			true,
		},
		{
			"gas consumed 1, used 10, overflow panic",
			1,
			math.MaxUint64,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			panicF := func() {
				gm := sdk.NewGasMeter(10)
				gm.ConsumeGas(tc.gasConsumed, "")
				ctx := suite.ctx.WithGasMeter(gm)
				suite.app.EvmKeeper.ResetGasMeterAndConsumeGas(ctx, tc.gasUsed)
			}

			if tc.expPanic {
				suite.Require().Panics(panicF)
			} else {
				suite.Require().NotPanics(panicF)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestEVMConfig() {
	proposerAddress := suite.ctx.BlockHeader().ProposerAddress
	cfg, err := suite.app.EvmKeeper.EVMConfig(suite.ctx, proposerAddress, big.NewInt(9000))
	suite.Require().NoError(err)
	suite.Require().Equal(types.DefaultParams(), cfg.Params)
	// london hardfork is enabled by default
	suite.Require().Equal(big.NewInt(0), cfg.BaseFee)
	suite.Require().Equal(suite.address, cfg.CoinBase)
	suite.Require().Equal(types.DefaultParams().ChainConfig.EthereumConfig(big.NewInt(9000)), cfg.ChainConfig)
}

func (suite *KeeperTestSuite) TestContractDeployment() {
	contractAddress := suite.DeployTestContract(suite.T(), suite.address, big.NewInt(10000000000000))
	db := suite.StateDB()
	suite.Require().Greater(db.GetCodeSize(contractAddress), 0)
}

func (suite *KeeperTestSuite) TestApplyMessage() {
	expectedGasUsed := params.TxGas
	var msg core.Message

	proposerAddress := suite.ctx.BlockHeader().ProposerAddress
	config, err := suite.app.EvmKeeper.EVMConfig(suite.ctx, proposerAddress, big.NewInt(9000))
	suite.Require().NoError(err)

	keeperParams := suite.app.EvmKeeper.GetParams(suite.ctx)
	chainCfg := keeperParams.ChainConfig.EthereumConfig(suite.app.EvmKeeper.ChainID())
	signer := ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())
	tracer := suite.app.EvmKeeper.Tracer(suite.ctx, msg, config.ChainConfig)
	vmdb := suite.StateDB()

	msg, err = newNativeMessage(
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

	res, err := suite.app.EvmKeeper.ApplyMessage(suite.ctx, msg, tracer, true)

	suite.Require().NoError(err)
	suite.Require().Equal(expectedGasUsed, res.GasUsed)
	suite.Require().False(res.Failed())
}

func (suite *KeeperTestSuite) TestApplyMessageWithConfig() {
	var (
		msg          core.Message
		err          error
		config       *statedb.EVMConfig
		keeperParams types.Params
		signer       ethtypes.Signer
		vmdb         *statedb.StateDB
		txConfig     statedb.TxConfig
		chainCfg     *params.ChainConfig
	)

	vfbcContractAddr, found := suite.app.EvmKeeper.GetVirtualFrontierBankContractAddressByDenom(suite.ctx, suite.denom)
	suite.Require().True(found, "require setup for virtual frontier bank contract of evm native denom")

	randomVFBCSenderAddress := common.BytesToAddress([]byte{0x01, 0x01, 0x39, 0x40})
	randomVFBCReceiverAddress := common.BytesToAddress([]byte{0x02, 0x02, 0x39, 0x40})
	vfbcTransferAmount := new(big.Int).Mul(big.NewInt(8), big.NewInt(int64(math.Pow(10, 18)))) // 8 * 10^18
	vfbcSenderInitialBalance := new(big.Int).SetUint64(math.MaxUint64)

	mintToVFBCSender := func(amount sdkmath.Int) {
		coins := sdk.NewCoins(sdk.NewCoin(suite.denom, amount))
		suite.app.BankKeeper.MintCoins(suite.ctx, minttypes.ModuleName, coins)
		suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, minttypes.ModuleName, randomVFBCSenderAddress.Bytes(), coins)
	}

	feeCompute := func(msg core.Message, gasUsed uint64) *big.Int {
		// tx fee was designed to be deducted in AnteHandler so this place does not cost fee
		return common.Big0
	}

	testCases := []struct {
		name           string
		malleate       func()
		expErr         bool
		testOnNonError func(core.Message, *types.MsgEthereumTxResponse)
	}{
		{
			name: "message applied ok",
			malleate: func() {
				msg, err = newNativeMessage(
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
			},
			expErr: false,
			testOnNonError: func(_ core.Message, res *types.MsgEthereumTxResponse) {
				suite.Require().False(res.Failed())
				suite.Require().Equal(params.TxGas, res.GasUsed)
			},
		},
		{
			name: "call contract tx with config param EnableCall = false",
			malleate: func() {
				config.Params.EnableCall = false
				msg, err = newNativeMessage(
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
			},
			expErr: true,
		},
		{
			name: "create contract tx with config param EnableCreate = false",
			malleate: func() {
				msg, err = suite.createContractGethMsg(vmdb.GetNonce(suite.address), signer, chainCfg, big.NewInt(1))
				suite.Require().NoError(err)
				config.Params.EnableCreate = false
			},
			expErr: true,
		},
		{
			name: "fix panic when minimumGasUsed is not uint64",
			malleate: func() {
				msg, err = newNativeMessage(
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
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasMultiplier = sdk.NewDec(math.MaxInt64).MulInt64(100)
				err = suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
			},
			expErr: true,
		},
		{
			name: "message transfer native via VFBC",
			malleate: func() {
				mintToVFBCSender(sdkmath.NewIntFromBigInt(vfbcSenderInitialBalance))

				callData := append(
					// transfer 1 to random receiver
					[]byte{0xa9, 0x05, 0x9c, 0xbb},
					append(
						common.BytesToHash(randomVFBCReceiverAddress.Bytes()).Bytes(),
						common.BytesToHash(vfbcTransferAmount.Bytes()).Bytes()...,
					)...,
				)

				msg = ethtypes.NewMessage(
					randomVFBCSenderAddress,                // from
					&vfbcContractAddr,                      // to
					vmdb.GetNonce(randomVFBCSenderAddress), // nonce
					nil,                                    // amount
					40_000,                                 // gas limit
					big.NewInt(7*int64(math.Pow(10, 9))),   // gas price
					big.NewInt(1),                          // gas fee cap
					big.NewInt(1),                          // gas tip cap
					callData,                               // call data
					nil,                                    // access list
					false,                                  // is fake
				)
			},
			expErr: false,
			testOnNonError: func(msg core.Message, res *types.MsgEthereumTxResponse) {
				if !suite.False(res.Failed()) {
					fmt.Printf("expect pass but got %s, desc %s\n", res.VmError, utils.MustAbiDecodeString(res.Ret[4:]))
					return
				}
				if suite.Len(res.Logs, 1, "should fire the Transfer event") {
					suite.Equal("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef", res.Logs[0].Topics[0])
				}
				suite.Equal(common.BytesToHash([]byte{0x01}).Bytes(), res.Ret) // ABI encoded of the boolean 'true'
				suite.Empty(res.VmError)
				if suite.Greater(res.GasUsed, uint64(21000)) {
					suite.Equal(uint64(35140), res.GasUsed)
				}

				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress)
				suite.Equal(vfbcTransferAmount, receiverBalance)

				senderFinalBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCSenderAddress)

				deductedFromSender := new(big.Int).Sub(vfbcSenderInitialBalance, senderFinalBalance)
				suite.Equal(vfbcTransferAmount, new(big.Int).Sub(deductedFromSender, feeCompute(msg, res.GasUsed)))
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest()
			proposerAddress := suite.ctx.BlockHeader().ProposerAddress
			config, err = suite.app.EvmKeeper.EVMConfig(suite.ctx, proposerAddress, big.NewInt(9000))
			suite.Require().NoError(err)

			keeperParams = suite.app.EvmKeeper.GetParams(suite.ctx)
			chainCfg = keeperParams.ChainConfig.EthereumConfig(suite.app.EvmKeeper.ChainID())
			signer = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())
			vmdb = suite.StateDB()
			txConfig = suite.app.EvmKeeper.TxConfig(suite.ctx, common.Hash{})

			tc.malleate()
			res, err := suite.app.EvmKeeper.ApplyMessageWithConfig(suite.ctx, msg, nil, true, config, txConfig)

			if tc.expErr {
				suite.Require().Error(err)
				return
			}

			suite.Require().NoError(err)
			suite.Require().NotNil(tc.testOnNonError, "test is required")
			tc.testOnNonError(msg, res)
		})
	}
}

func (suite *KeeperTestSuite) TestApplyMessageWithConfig_VFC_Ops() {
	vfbcContractAddrOfNative, found := suite.app.EvmKeeper.GetVirtualFrontierBankContractAddressByDenom(suite.ctx, suite.denom)
	suite.Require().True(found, "require setup for virtual frontier bank contract of evm native denom")

	randomVFBCSenderAddress := common.BytesToAddress([]byte{0x01, 0x01, 0x39, 0x40})
	randomVFBCReceiverAddress := common.BytesToAddress([]byte{0x02, 0x02, 0x39, 0x40})
	vfbcSenderInitialBalance := new(big.Int).SetUint64(math.MaxUint64)

	vfc := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, vfbcContractAddrOfNative)
	suite.Require().NotNil(vfc)

	bankDenomMetadataOfNative, found := suite.app.BankKeeper.GetDenomMetaData(suite.ctx, suite.denom)
	suite.Require().True(found)
	vfbcDenomMetadataOfNative, valid := types.CollectMetadataForVirtualFrontierBankContract(bankDenomMetadataOfNative)
	suite.Require().True(valid)

	senderNonce := func() uint64 {
		return suite.app.EvmKeeper.GetNonce(suite.ctx, randomVFBCSenderAddress)
	}

	feeCompute := func(msg core.Message, gasUsed uint64) *big.Int {
		// tx fee was designed to be deducted in AnteHandler so this place does not cost fee
		return common.Big0
	}

	callDataName := []byte{0x06, 0xfd, 0xde, 0x03}
	callDataSymbol := []byte{0x95, 0xd8, 0x9b, 0x41}
	callDataDecimals := []byte{0x31, 0x3c, 0xe5, 0x67}
	callDataTotalSupply := []byte{0x18, 0x16, 0x0d, 0xdd}
	callDataBalanceOf := []byte{0x70, 0xa0, 0x82, 0x31}
	callDataTransferSig := []byte{0xa9, 0x05, 0x9c, 0xbb}
	callDataTransferTo := func(receiver common.Address, amount *big.Int) (ret []byte) {
		ret = append(callDataTransferSig, common.BytesToHash(receiver.Bytes()).Bytes()...)
		ret = append(ret, common.BytesToHash(amount.Bytes()).Bytes()...)
		return ret
	}
	callDataTransfer := func(amount *big.Int) (ret []byte) {
		return callDataTransferTo(randomVFBCReceiverAddress, amount)
	}

	computeIntrinsicGas := func(msg core.Message) uint64 {
		gas, err := core.IntrinsicGas(msg.Data(), msg.AccessList(), msg.To() == nil, true, true)
		if err != nil {
			panic(err)
		}
		return gas
	}

	bytesOfAbiEncodedTrue := func() []byte {
		b := make([]byte, 32)
		b[31] = 0x1
		return b
	}()

	tests := []struct {
		name                              string
		prepare                           func() core.Message
		wantExecError                     bool
		wantVmError                       bool
		wantSenderBalanceAdjustmentOrZero *big.Int
		testOnNonExecError                func(core.Message, *types.MsgEthereumTxResponse)
	}{
		{
			name: "prohibit transfer to VFBC contract (no call data)",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					nil,                       // call data
					nil,                       // access list
					false,                     // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "not allowed to receive")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg), response.GasUsed, "exec revert consume no gas except intrinsic gas")
			},
		},
		{
			name: "prohibit transfer to VFBC contract (non-zero value)",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					common.Big1,               // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataName,              // call data
					nil,                       // access list
					false,                     // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "not allowed to receive")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg), response.GasUsed, "exec revert consume no gas except intrinsic gas")
			},
		},
		{
			name: "method signature does not exists",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,    // from
					&vfbcContractAddrOfNative,  // to
					senderNonce(),              // nonce
					nil,                        // amount
					40_000,                     // gas limit
					big.NewInt(1),              // gas price
					big.NewInt(1),              // gas fee cap
					big.NewInt(1),              // gas tip cap
					[]byte{0x1, 0x2, 0x3, 0x4}, // call data
					nil,                        // access list
					false,                      // is fake
				)
			},
			wantExecError: false,
			wantVmError:   false,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed(), "execution should not fail")
				suite.Empty(response.Ret)
				suite.Equal(computeIntrinsicGas(msg), response.GasUsed, "should consume no gas except intrinsic gas")
			},
		},
		{
			name: "name() but lacking gas",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,          // from
					&vfbcContractAddrOfNative,        // to
					senderNonce(),                    // nonce
					nil,                              // amount
					params.TxGas+types.VFBCopgName-1, // gas limit
					big.NewInt(1),                    // gas price
					big.NewInt(1),                    // gas fee cap
					big.NewInt(1),                    // gas tip cap
					callDataName,                     // call data
					nil,                              // access list
					false,                            // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "out of gas")
				suite.Equal(vm.ErrOutOfGas.Error(), response.VmError)
				suite.Equal(msg.Gas(), response.GasUsed, "out of gas consume all gas")
			},
		},
		{
			name: "name() but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					append(callDataName, 0x1), // call data
					nil,                       // access list
					false,                     // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgName_Revert, response.GasUsed)
			},
		},
		{
			name: "name()",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataName,              // call data
					nil,                       // access list
					false,                     // is fake
				)
			},
			wantExecError: false,
			wantVmError:   false,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				suite.Equal(vfbcDenomMetadataOfNative.Name, utils.MustAbiDecodeString(response.Ret))
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgName, response.GasUsed)
			},
		},
		{
			name: "symbol() but lacking gas",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,            // from
					&vfbcContractAddrOfNative,          // to
					senderNonce(),                      // nonce
					nil,                                // amount
					params.TxGas+types.VFBCopgSymbol-1, // gas limit
					big.NewInt(1),                      // gas price
					big.NewInt(1),                      // gas fee cap
					big.NewInt(1),                      // gas tip cap
					callDataSymbol,                     // call data
					nil,                                // access list
					false,                              // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "out of gas")
				suite.Equal(vm.ErrOutOfGas.Error(), response.VmError)
				suite.Equal(msg.Gas(), response.GasUsed, "out of gas consume all gas")
			},
		},
		{
			name: "symbol() but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,     // from
					&vfbcContractAddrOfNative,   // to
					senderNonce(),               // nonce
					nil,                         // amount
					40_000,                      // gas limit
					big.NewInt(1),               // gas price
					big.NewInt(1),               // gas fee cap
					big.NewInt(1),               // gas tip cap
					append(callDataSymbol, 0x1), // call data
					nil,                         // access list
					false,                       // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgSymbol_Revert, response.GasUsed)
			},
		},
		{
			name: "symbol()",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataSymbol,            // call data
					nil,                       // access list
					false,                     // is fake
				)
			},
			wantExecError: false,
			wantVmError:   false,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				suite.Equal(vfbcDenomMetadataOfNative.Symbol, utils.MustAbiDecodeString(response.Ret))
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgSymbol, response.GasUsed)
			},
		},
		{
			name: "decimals() but lacking gas",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,              // from
					&vfbcContractAddrOfNative,            // to
					senderNonce(),                        // nonce
					nil,                                  // amount
					params.TxGas+types.VFBCopgDecimals-1, // gas limit
					big.NewInt(1),                        // gas price
					big.NewInt(1),                        // gas fee cap
					big.NewInt(1),                        // gas tip cap
					callDataDecimals,                     // call data
					nil,                                  // access list
					false,                                // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "out of gas")
				suite.Equal(vm.ErrOutOfGas.Error(), response.VmError)
				suite.Equal(msg.Gas(), response.GasUsed, "out of gas consume all gas")
			},
		},
		{
			name: "decimals() but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,       // from
					&vfbcContractAddrOfNative,     // to
					senderNonce(),                 // nonce
					nil,                           // amount
					40_000,                        // gas limit
					big.NewInt(1),                 // gas price
					big.NewInt(1),                 // gas fee cap
					big.NewInt(1),                 // gas tip cap
					append(callDataDecimals, 0x1), // call data
					nil,                           // access list
					false,                         // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgDecimals_Revert, response.GasUsed)
			},
		},
		{
			name: "decimals()",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataDecimals,          // call data
					nil,                       // access list
					false,                     // is fake
				)
			},
			wantExecError: false,
			wantVmError:   false,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				suite.Equal(vfbcDenomMetadataOfNative.Decimals, uint32(new(big.Int).SetBytes(response.Ret).Uint64()))
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgDecimals, response.GasUsed)
			},
		},
		{
			name: "totalSupply() but lacking gas",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					params.TxGas+types.VFBCopgTotalSupply-1, // gas limit
					big.NewInt(1),       // gas price
					big.NewInt(1),       // gas fee cap
					big.NewInt(1),       // gas tip cap
					callDataTotalSupply, // call data
					nil,                 // access list
					false,               // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "out of gas")
				suite.Equal(vm.ErrOutOfGas.Error(), response.VmError)
				suite.Equal(msg.Gas(), response.GasUsed, "out of gas consume all gas")
			},
		},
		{
			name: "totalSupply() but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,          // from
					&vfbcContractAddrOfNative,        // to
					senderNonce(),                    // nonce
					nil,                              // amount
					40_000,                           // gas limit
					big.NewInt(1),                    // gas price
					big.NewInt(1),                    // gas fee cap
					big.NewInt(1),                    // gas tip cap
					append(callDataTotalSupply, 0x1), // call data
					nil,                              // access list
					false,                            // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTotalSupply_Revert, response.GasUsed)
			},
		},
		{
			name: "totalSupply()",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataTotalSupply,       // call data
					nil,                       // access list
					false,                     // is fake
				)
			},
			wantExecError: false,
			wantVmError:   false,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				retBigInt := new(big.Int).SetBytes(response.Ret)
				totalSupply := suite.app.BankKeeper.GetSupply(suite.ctx, suite.denom).Amount.BigInt()
				suite.Truef(retBigInt.Cmp(totalSupply) == 0, "total supply mismatch, want %s, got %s", totalSupply, retBigInt)
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTotalSupply, response.GasUsed)
			},
		},
		{
			name: "balanceOf(address) but lacking gas",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					params.TxGas+types.VFBCopgBalanceOf-1, // gas limit
					big.NewInt(1), // gas price
					big.NewInt(1), // gas fee cap
					big.NewInt(1), // gas tip cap
					append(callDataBalanceOf, common.BytesToHash(randomVFBCSenderAddress.Bytes()).Bytes()...), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "out of gas")
				suite.Equal(vm.ErrOutOfGas.Error(), response.VmError)
				suite.Equal(msg.Gas(), response.GasUsed, "out of gas consume all gas")
			},
		},
		{
			name: "balanceOf(address) but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,        // from
					&vfbcContractAddrOfNative,      // to
					senderNonce(),                  // nonce
					nil,                            // amount
					40_000,                         // gas limit
					big.NewInt(1),                  // gas price
					big.NewInt(1),                  // gas fee cap
					big.NewInt(1),                  // gas tip cap
					append(callDataBalanceOf, 0x1), // call data
					nil,                            // access list
					false,                          // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgBalanceOf_Revert, response.GasUsed)
			},
		},
		{
			name: "balanceOf(address) but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					append(callDataBalanceOf, make([]byte, 33)...), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgBalanceOf_Revert, response.GasUsed)
			},
		},
		{
			name: "balanceOf(address) but invalid address, take only last 20 bytes",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					append(callDataBalanceOf, func() []byte {
						invalidAddress := common.BytesToHash(randomVFBCSenderAddress.Bytes()).Bytes()
						for i := 0; i < 12; i++ {
							invalidAddress[i] = 0xff
						}
						return invalidAddress
					}()...), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError: false,
			wantVmError:   false,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				retBigInt := new(big.Int).SetBytes(response.Ret)
				balance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCSenderAddress)
				suite.Truef(retBigInt.Cmp(balance) == 0, "balance mismatch, want %s, got %s", balance, retBigInt)
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgBalanceOf, response.GasUsed)
			},
		},
		{
			name: "balanceOf(address)",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					append(callDataBalanceOf, common.BytesToHash(randomVFBCSenderAddress.Bytes()).Bytes()...), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError: false,
			wantVmError:   false,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				retBigInt := new(big.Int).SetBytes(response.Ret)
				balance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCSenderAddress)
				suite.Truef(retBigInt.Cmp(balance) == 0, "balance mismatch, want %s, got %s", balance, retBigInt)
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgBalanceOf, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256) but lacking gas",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,              // from
					&vfbcContractAddrOfNative,            // to
					senderNonce(),                        // nonce
					nil,                                  // amount
					params.TxGas+types.VFBCopgTransfer-1, // gas limit
					big.NewInt(1),                        // gas price
					big.NewInt(1),                        // gas fee cap
					big.NewInt(1),                        // gas tip cap
					callDataTransfer(common.Big1),        // call data
					nil,                                  // access list
					false,                                // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "out of gas")
				suite.Equal(vm.ErrOutOfGas.Error(), response.VmError)
				suite.Equal(msg.Gas(), response.GasUsed, "out of gas consume all gas")
			},
		},
		{
			name: "transfer(address,uint256) but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,          // from
					&vfbcContractAddrOfNative,        // to
					senderNonce(),                    // nonce
					nil,                              // amount
					40_000,                           // gas limit
					big.NewInt(1),                    // gas price
					big.NewInt(1),                    // gas fee cap
					big.NewInt(1),                    // gas tip cap
					append(callDataTransferSig, 0x1), // call data
					nil,                              // access list
					false,                            // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer_Revert, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256) but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					append(callDataTransferSig, make([]byte, 32+32+1)...), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer_Revert, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), transfer more than balance",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataTransfer(new(big.Int).Add(vfbcSenderInitialBalance, common.Big1)), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       true,
			wantSenderBalanceAdjustmentOrZero: common.Big0,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().True(response.Failed())
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress)
				suite.Truef(receiverBalance.Sign() == 0, "receiver should receives nothing but got %s", receiverBalance)
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "transfer amount exceeds balance")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer_Revert, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), transfer 1",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,       // from
					&vfbcContractAddrOfNative,     // to
					senderNonce(),                 // nonce
					nil,                           // amount
					40_000,                        // gas limit
					big.NewInt(1),                 // gas price
					big.NewInt(1),                 // gas fee cap
					big.NewInt(1),                 // gas tip cap
					callDataTransfer(common.Big1), // call data
					nil,                           // access list
					false,                         // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       false,
			wantSenderBalanceAdjustmentOrZero: common.Big1,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				suite.Equal(bytesOfAbiEncodedTrue, response.Ret)
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress)
				suite.Truef(receiverBalance.Cmp(common.Big1) == 0, "balance mismatch, want %s, got %s", common.Big1, receiverBalance)
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), transfer zero",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,       // from
					&vfbcContractAddrOfNative,     // to
					senderNonce(),                 // nonce
					nil,                           // amount
					40_000,                        // gas limit
					big.NewInt(1),                 // gas price
					big.NewInt(1),                 // gas fee cap
					big.NewInt(1),                 // gas tip cap
					callDataTransfer(common.Big0), // call data
					nil,                           // access list
					false,                         // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       false,
			wantSenderBalanceAdjustmentOrZero: common.Big0,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				suite.Equal(bytesOfAbiEncodedTrue, response.Ret)
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress)
				suite.Truef(receiverBalance.Sign() == 0, "balance mismatch, want %s, got %s", common.Big0, receiverBalance)
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), transfer value equals to max uint64",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataTransfer(new(big.Int).SetUint64(math.MaxUint64)), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       false,
			wantSenderBalanceAdjustmentOrZero: new(big.Int).SetUint64(math.MaxUint64),
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				suite.Equal(bytesOfAbiEncodedTrue, response.Ret)
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress)
				wantReceiverBalance := new(big.Int).SetUint64(math.MaxUint64)
				suite.Truef(receiverBalance.Cmp(wantReceiverBalance) == 0, "balance mismatch, want %s, got %s", new(big.Int).SetUint64(math.MaxUint64), receiverBalance)
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), but invalid address, take the first 20 bytes",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					func() []byte {
						callData := callDataTransfer(common.Big1)
						for i := 5; i < 15; i++ {
							callData[i] = 0xff
						}
						return callData
					}(), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       false,
			wantSenderBalanceAdjustmentOrZero: common.Big1,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				suite.Equal(bytesOfAbiEncodedTrue, response.Ret)
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress)
				suite.Truef(receiverBalance.Cmp(common.Big1) == 0, "balance mismatch, want %s, got %s", common.Big1, receiverBalance)
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), can not transfer to module account",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataTransferTo(types.VirtualFrontierContractDeployerAddress, common.Big1), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       true,
			wantSenderBalanceAdjustmentOrZero: common.Big0,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().True(response.Failed())
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, types.VirtualFrontierContractDeployerAddress)
				suite.Truef(receiverBalance.Sign() == 0, "receiver should receives nothing but got %s", receiverBalance)
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "can not transfer to module account")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer_Revert, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), can not transfer to VFC contract",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataTransferTo(vfbcContractAddrOfNative, common.Big1), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       true,
			wantSenderBalanceAdjustmentOrZero: common.Big0,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().True(response.Failed())
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, vfbcContractAddrOfNative)
				suite.Truef(receiverBalance.Sign() == 0, "receiver should receives nothing but got %s", receiverBalance)
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "not allowed to receive")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer_Revert, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), can not transfer coin that not send-able",
			prepare: func() core.Message {
				bankParams := suite.app.BankKeeper.GetParams(suite.ctx)
				bankParams.SendEnabled = []*banktypes.SendEnabled{{
					Denom:   suite.denom,
					Enabled: false,
				}}
				suite.app.BankKeeper.SetParams(suite.ctx, bankParams)

				return ethtypes.NewMessage(
					randomVFBCSenderAddress,       // from
					&vfbcContractAddrOfNative,     // to
					senderNonce(),                 // nonce
					nil,                           // amount
					40_000,                        // gas limit
					big.NewInt(1),                 // gas price
					big.NewInt(1),                 // gas fee cap
					big.NewInt(1),                 // gas tip cap
					callDataTransfer(common.Big1), // call data
					nil,                           // access list
					false,                         // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       true,
			wantSenderBalanceAdjustmentOrZero: common.Big0,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().True(response.Failed())
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress)
				suite.Truef(receiverBalance.Sign() == 0, "receiver should receives nothing but got %s", receiverBalance)
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "transfers are currently disabled")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer_Revert, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), can not transfer if contract was de-activated",
			prepare: func() core.Message {
				contract := *vfc
				contract.Active = false
				suite.app.EvmKeeper.SetVirtualFrontierContract(suite.ctx, vfbcContractAddrOfNative, &contract)

				return ethtypes.NewMessage(
					randomVFBCSenderAddress,       // from
					&vfbcContractAddrOfNative,     // to
					senderNonce(),                 // nonce
					nil,                           // amount
					40_000,                        // gas limit
					big.NewInt(1),                 // gas price
					big.NewInt(1),                 // gas fee cap
					big.NewInt(1),                 // gas tip cap
					callDataTransfer(common.Big1), // call data
					nil,                           // access list
					false,                         // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       true,
			wantSenderBalanceAdjustmentOrZero: common.Big0,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().True(response.Failed())
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress)
				suite.Truef(receiverBalance.Sign() == 0, "receiver should receives nothing but got %s", receiverBalance)
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "is not active")
				suite.Contains(response.VmError, "is not active")
				suite.Equal(msg.Gas(), response.GasUsed, "error tx consumes all gas")
			},
		},
	}
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			// minting

			coins := sdk.NewCoins(sdk.NewCoin(suite.denom, sdkmath.NewIntFromBigInt(vfbcSenderInitialBalance)))
			suite.app.BankKeeper.MintCoins(suite.ctx, minttypes.ModuleName, coins)
			suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, minttypes.ModuleName, randomVFBCSenderAddress.Bytes(), coins)

			// prepare exec ctx

			proposerAddress := suite.ctx.BlockHeader().ProposerAddress
			config, err := suite.app.EvmKeeper.EVMConfig(suite.ctx, proposerAddress, big.NewInt(9000))
			suite.Require().NoError(err)

			txConfig := suite.app.EvmKeeper.TxConfig(suite.ctx, common.Hash{})

			msg := tc.prepare()

			preExecNonce := suite.app.EvmKeeper.GetNonce(suite.ctx, randomVFBCSenderAddress)

			res, err := suite.app.EvmKeeper.ApplyMessageWithConfig(suite.ctx, msg, nil, true, config, txConfig)

			suite.Equal(preExecNonce, suite.app.EvmKeeper.GetNonce(suite.ctx, randomVFBCSenderAddress), "nonce increment must be handled by AnteHandler")

			if tc.wantExecError {
				suite.Require().Error(err)
				suite.Equal(vfbcSenderInitialBalance, suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCSenderAddress), "balance should not change on failed txs")
				return
			}

			suite.Require().NoError(err)

			wantBalanceAdjustmentOfSender := tc.wantSenderBalanceAdjustmentOrZero
			if wantBalanceAdjustmentOfSender == nil {
				wantBalanceAdjustmentOfSender = common.Big0
			}

			actualBalanceAdjustmentOfSender := new(big.Int).Sub(vfbcSenderInitialBalance, suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCSenderAddress))
			actualBalanceAdjustmentOfSender = new(big.Int).Sub(actualBalanceAdjustmentOfSender, feeCompute(msg, res.GasUsed))

			suite.Truef(wantBalanceAdjustmentOfSender.Cmp(actualBalanceAdjustmentOfSender) == 0, "balance adjustment mismatch, want %s, got %s", wantBalanceAdjustmentOfSender, actualBalanceAdjustmentOfSender)

			if tc.wantVmError {
				suite.Require().True(res.Failed())
				suite.Require().NotEmpty(res.VmError)
				suite.True(suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress).Sign() == 0, "receiver should receives nothing")
			} else {
				suite.Require().False(res.Failed())
				suite.Require().Empty(res.VmError)
			}

			suite.Require().NotNil(tc.testOnNonExecError, "post-test is required")
			tc.testOnNonExecError(msg, res)
		})
	}
}

func (suite *KeeperTestSuite) createContractGethMsg(nonce uint64, signer ethtypes.Signer, cfg *params.ChainConfig, gasPrice *big.Int) (core.Message, error) {
	ethMsg, err := suite.createContractMsgTx(nonce, signer, cfg, gasPrice)
	if err != nil {
		return nil, err
	}

	msgSigner := ethtypes.MakeSigner(cfg, big.NewInt(suite.ctx.BlockHeight()))
	return ethMsg.AsMessage(msgSigner, nil)
}

func (suite *KeeperTestSuite) createContractMsgTx(nonce uint64, signer ethtypes.Signer, cfg *params.ChainConfig, gasPrice *big.Int) (*types.MsgEthereumTx, error) {
	contractCreateTx := &ethtypes.AccessListTx{
		GasPrice: gasPrice,
		Gas:      params.TxGasContractCreation,
		To:       nil,
		Data:     []byte("contract_data"),
		Nonce:    nonce,
	}
	ethTx := ethtypes.NewTx(contractCreateTx)
	ethMsg := &types.MsgEthereumTx{}
	ethMsg.FromEthereumTx(ethTx)
	ethMsg.From = suite.address.Hex()

	return ethMsg, ethMsg.Sign(signer, suite.signer)
}

func (suite *KeeperTestSuite) TestGetProposerAddress() {
	var a sdk.ConsAddress
	address := sdk.ConsAddress(suite.address.Bytes())
	proposerAddress := sdk.ConsAddress(suite.ctx.BlockHeader().ProposerAddress)
	testCases := []struct {
		msg    string
		adr    sdk.ConsAddress
		expAdr sdk.ConsAddress
	}{
		{
			"proposer address provided",
			address,
			address,
		},
		{
			"nil proposer address provided",
			nil,
			proposerAddress,
		},
		{
			"typed nil proposer address provided",
			a,
			proposerAddress,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.Require().Equal(tc.expAdr, keeper.GetProposerAddress(suite.ctx, tc.adr))
		})
	}
}
