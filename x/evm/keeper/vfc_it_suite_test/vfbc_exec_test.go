package vfc_it_suite_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/ethermint/integration_test_util"
	itutiltypes "github.com/evmos/ethermint/integration_test_util/types"
	"github.com/evmos/ethermint/rpc/namespaces/ethereum/eth"
	rpctypes "github.com/evmos/ethermint/rpc/types"
	"github.com/evmos/ethermint/testutil"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"github.com/tendermint/tendermint/libs/log"
	"math/big"
)

func (suite *VfcITSuite) TestExecVirtualFrontierBankContract() {
	deployer := suite.CITS.WalletAccounts.Number(1)
	tokenOwner := suite.CITS.WalletAccounts.Number(2)
	senderOfVfbc := suite.CITS.WalletAccounts.Number(3)
	lazySender := suite.CITS.WalletAccounts.Number(4)
	receiver := integration_test_util.NewTestAccount(suite.T(), nil)

	suite.CITS.MintCoin(tokenOwner, suite.CITS.NewBaseCoin(30))
	suite.CITS.MintCoin(senderOfVfbc, suite.CITS.NewBaseCoin(30))
	suite.CITS.MintCoin(lazySender, suite.CITS.NewBaseCoin(30))

	normalErc20ContractAddress, _, _, err := suite.CITS.TxDeploy2WDymContract(deployer, tokenOwner)
	suite.Require().NoError(err)

	vfbcContractAddress, found := suite.App().EvmKeeper().GetVirtualFrontierBankContractAddressByDenom(suite.Ctx(), suite.CITS.ChainConstantsConfig.GetMinDenom())
	suite.Require().True(found)

	suite.CITS.WaitNextBlockOrCommit()

	// prepare some ERC-20 funds for lazySender
	txPrepare, _, err := suite.CITS.TxTransferErc20Token(normalErc20ContractAddress, tokenOwner, lazySender, 1000, 0)
	suite.Require().NoError(err)
	suite.Commit() // commit to passive trigger EVM Tx indexer
	suite.Require().Equal(ethtypes.ReceiptStatusSuccessful, suite.GetTxReceipt(txPrepare.AsTransaction().Hash()).Status)

	type targetContract uint8
	const (
		erc20Contract targetContract = iota + 1
		vfbcContract
	)

	tests := []struct {
		name            string
		inputFunc       func() []byte
		targetContract  targetContract
		overrideSender  *itutiltypes.TestAccount
		wantSuccess     bool
		wantErrContains string
	}{
		{
			name: "pass - ERC-20 name()",
			inputFunc: func() []byte {
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("name")
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  erc20Contract,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "pass - VFBC name()",
			inputFunc: func() []byte {
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("name")
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  vfbcContract,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "pass - ERC-20 symbol()",
			inputFunc: func() []byte {
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("symbol")
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  erc20Contract,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "pass - VFBC symbol()",
			inputFunc: func() []byte {
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("symbol")
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  vfbcContract,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "pass - ERC-20 decimals()",
			inputFunc: func() []byte {
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("decimals")
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  erc20Contract,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "pass - VFBC decimals()",
			inputFunc: func() []byte {
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("decimals")
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  vfbcContract,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "pass - ERC-20 totalSupply()",
			inputFunc: func() []byte {
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("totalSupply")
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  erc20Contract,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "pass - VFBC totalSupply()",
			inputFunc: func() []byte {
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("totalSupply")
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  vfbcContract,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "pass - ERC-20 balanceOf(address)",
			inputFunc: func() []byte {
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("balanceOf", receiver.GetEthAddress())
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  erc20Contract,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "pass - VFBC balanceOf(address)",
			inputFunc: func() []byte {
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("balanceOf", receiver.GetEthAddress())
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  vfbcContract,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "pass - ERC-20 transfer(address,uint256)",
			inputFunc: func() []byte {
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("transfer", receiver.GetEthAddress(), common.Big1)
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  erc20Contract,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "pass - VFBC transfer(address,uint256)",
			inputFunc: func() []byte {
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("transfer", receiver.GetEthAddress(), common.Big1)
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  vfbcContract,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "pass - ERC-20 approve(address,uint256)",
			inputFunc: func() []byte {
				// lazySender approve tokenOwner to spend max 100 tokens on behalf of lazySender
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("approve", tokenOwner.GetEthAddress(), big.NewInt(100))
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  erc20Contract,
			overrideSender:  lazySender,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "fail - not support - VFBC approve(address,uint256)",
			inputFunc: func() []byte {
				// lazySender approve senderOfVfbc to spend max 100 tokens on behalf of lazySender
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("approve", senderOfVfbc.GetEthAddress(), big.NewInt(100))
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  vfbcContract,
			overrideSender:  lazySender,
			wantSuccess:     false,
			wantErrContains: "execution reverted",
		},
		{
			name: "pass - ERC-20 transferFrom(address,address,uint256)",
			inputFunc: func() []byte {
				// tokenOwner transfer 1 token from lazySender to receiver
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("transferFrom", lazySender.GetEthAddress(), receiver.GetEthAddress(), common.Big1)
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  erc20Contract,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "fail - not support - VFBC transferFrom(address,address,uint256)",
			inputFunc: func() []byte {
				// senderOfVfbc transfer 1 token from lazySender to receiver
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("transferFrom", lazySender.GetEthAddress(), receiver.GetEthAddress(), common.Big1)
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  vfbcContract,
			wantSuccess:     false,
			wantErrContains: "execution reverted",
		},
		{
			name: "pass - ERC-20 allowance(address,uint256)",
			inputFunc: func() []byte {
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("allowance", lazySender.GetEthAddress(), tokenOwner.GetEthAddress())
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  erc20Contract,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "fail - not support - VFBC allowance(address,uint256)",
			inputFunc: func() []byte {
				inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("allowance", lazySender.GetEthAddress(), tokenOwner.GetEthAddress())
				suite.Require().NoError(err)
				return inputCallData
			},
			targetContract:  vfbcContract,
			wantSuccess:     false,
			wantErrContains: "execution reverted",
		},
		{
			name: "pass - not ERC-20 method then do fallback",
			inputFunc: func() []byte {
				return append([]byte{0x01, 0x02, 0x03, 0x04}, ethtypes.EmptyRootHash.Bytes()...)
			},
			targetContract:  erc20Contract,
			wantSuccess:     true,
			wantErrContains: "",
		},
		{
			name: "pass - not VFBC ERC-20 method then do fallback",
			inputFunc: func() []byte {
				return append([]byte{0x01, 0x02, 0x03, 0x04}, ethtypes.EmptyRootHash.Bytes()...)
			},
			targetContract:  vfbcContract,
			wantSuccess:     true,
			wantErrContains: "",
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			inputCallData := tt.inputFunc()

			var sender *itutiltypes.TestAccount
			var from, to common.Address
			if tt.targetContract == erc20Contract {
				sender = tokenOwner
				to = normalErc20ContractAddress
			} else if tt.targetContract == vfbcContract {
				sender = senderOfVfbc
				to = vfbcContractAddress
			} else {
				panic("unknown target contract")
			}

			if tt.overrideSender != nil {
				sender = tt.overrideSender
			}

			from = sender.GetEthAddress()

			ctx := suite.Ctx()

			msgEthereumTx := evmtypes.NewTx(
				suite.CITS.ChainApp.EvmKeeper().ChainID(),
				suite.CITS.ChainApp.EvmKeeper().GetNonce(ctx, from),
				&to,
				nil,
				100_000, // gas limit
				nil,
				suite.CITS.ChainApp.FeeMarketKeeper().GetBaseFee(ctx),
				big.NewInt(1),
				inputCallData,
				&ethtypes.AccessList{},
			)
			msgEthereumTx.From = from.String()

			resp, err := suite.CITS.DeliverEthTx(sender, msgEthereumTx)

			suite.Commit()

			if tt.wantSuccess {
				suite.Require().NoError(err)
				suite.Require().NotNil(resp)

				receipt := suite.GetTxReceipt(common.HexToHash(resp.EthTxHash))
				suite.Require().NotNil(receipt)

				suite.Equal(ethtypes.ReceiptStatusSuccessful, receipt.Status, "tx must be successful")
				suite.Empty(resp.EvmError)
			} else {
				suite.Require().Error(err)

				receipt := suite.GetTxReceipt(common.HexToHash(resp.EthTxHash))
				suite.Require().NotNil(receipt)

				suite.Equal(ethtypes.ReceiptStatusFailed, receipt.Status, "tx must be failed")
				suite.Require().NotEmpty(tt.wantErrContains, "mis-configuration")
				suite.Contains(err.Error(), tt.wantErrContains)

				if suite.NotNil(resp) {
					suite.Contains(resp.EvmError, tt.wantErrContains)
				}
			}
		})
	}

	suite.Run("Transfer IBC token via VFBC", func() {
		// prepare metadata and contracts
		metaIbcAtom := testutil.NewBankDenomMetadata("ibc/C4CFF46FD6DE35CA4CF4CE031E643C8FDC9BA4B99AE598E9B0ED98FE3A2319F9", 6)
		metaIbcTia := testutil.NewBankDenomMetadata("ibc/45D6B52CAD911A15BD9C2F5FFDA80E26AFCB05C7CD520070790ABC86D2B24229", 6)

		suite.App().BankKeeper().SetDenomMetaData(suite.Ctx(), metaIbcAtom)
		suite.App().BankKeeper().SetDenomMetaData(suite.Ctx(), metaIbcTia)

		suite.CITS.WaitNextBlockOrCommit()

		vfbcAtom, foundAtom := suite.App().EvmKeeper().GetVirtualFrontierBankContractAddressByDenom(suite.Ctx(), metaIbcAtom.Base)
		suite.Require().True(foundAtom)
		vfbcTia, foundTia := suite.App().EvmKeeper().GetVirtualFrontierBankContractAddressByDenom(suite.Ctx(), metaIbcTia.Base)
		suite.Require().True(foundTia)

		// prepare funds

		const originalBalance = 10_000_000

		suite.CITS.MintCoin(senderOfVfbc, sdk.NewCoin(metaIbcAtom.Base, sdk.NewInt(originalBalance)))
		suite.CITS.MintCoin(senderOfVfbc, sdk.NewCoin(metaIbcTia.Base, sdk.NewInt(originalBalance)))

		suite.CITS.WaitNextBlockOrCommit()

		send := func(contract common.Address, amount uint64) {
			from := senderOfVfbc.GetEthAddress()

			inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("transfer", receiver.GetEthAddress(), new(big.Int).SetUint64(amount))
			suite.Require().NoError(err)

			ctx := suite.Ctx()

			msgEthereumTx := evmtypes.NewTx(
				suite.CITS.ChainApp.EvmKeeper().ChainID(),
				suite.CITS.ChainApp.EvmKeeper().GetNonce(ctx, from),
				&contract,
				nil,
				100_000, // gas limit
				nil,
				suite.CITS.ChainApp.FeeMarketKeeper().GetBaseFee(ctx),
				big.NewInt(1),
				inputCallData,
				&ethtypes.AccessList{},
			)
			msgEthereumTx.From = from.String()

			resp, err := suite.CITS.DeliverEthTx(senderOfVfbc, msgEthereumTx)
			suite.Require().NoError(err)

			suite.Commit()

			receipt := suite.GetTxReceipt(common.HexToHash(resp.EthTxHash))
			suite.Require().NotNil(receipt)
			suite.Equal(ethtypes.ReceiptStatusSuccessful, receipt.Status, "tx must be successful")
		}

		send(vfbcAtom, 1_000_001)
		send(vfbcTia, 2_000_002)

		suite.Equal(
			int64(8_999_999),
			suite.CITS.QueryBalanceByDenom(0, senderOfVfbc.GetCosmosAddress().String(), metaIbcAtom.Base).Amount.Int64(),
		)
		suite.Equal(
			int64(1_000_001),
			suite.CITS.QueryBalanceByDenom(0, receiver.GetCosmosAddress().String(), metaIbcAtom.Base).Amount.Int64(),
		)
		suite.Equal(
			int64(7_999_998),
			suite.CITS.QueryBalanceByDenom(0, senderOfVfbc.GetCosmosAddress().String(), metaIbcTia.Base).Amount.Int64(),
		)
		suite.Equal(
			int64(2_000_002),
			suite.CITS.QueryBalanceByDenom(0, receiver.GetCosmosAddress().String(), metaIbcTia.Base).Amount.Int64(),
		)
	})

	suite.Run("Normal smart contract interactive must work if does not change VFBC account status and storage", func() {
		getBalance := func(holder common.Address) *big.Int {
			inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("balanceOf", holder)
			suite.Require().NoError(err)

			input := hexutil.Bytes(inputCallData)

			bz, err := eth.NewPublicAPI(log.NewNopLogger(), suite.CITS.RpcBackendAt(0)).Call(evmtypes.TransactionArgs{
				To: &normalErc20ContractAddress,
				Gas: func() *hexutil.Uint64 {
					gas := hexutil.Uint64(100_000)
					return &gas
				}(),
				GasPrice: func() *hexutil.Big {
					gasPrice := hexutil.Big(*suite.CITS.ChainApp.FeeMarketKeeper().GetBaseFee(suite.Ctx()))
					return &gasPrice
				}(),
				Nonce: func() *hexutil.Uint64 {
					nonce := hexutil.Uint64(suite.App().EvmKeeper().GetNonce(suite.Ctx(), holder))
					return &nonce
				}(),
				Input: &input,
				ChainID: func() *hexutil.Big {
					chainID := hexutil.Big(*suite.CITS.ChainApp.EvmKeeper().ChainID())
					return &chainID
				}(),
			}, rpctypes.BlockNumberOrHash{
				BlockNumber: func() *rpctypes.BlockNumber {
					blk := rpctypes.EthLatestBlockNumber
					return &blk
				}(),
			}, nil)

			suite.Require().NoError(err)

			return new(big.Int).SetBytes(bz)
		}

		sender := tokenOwner.GetEthAddress()
		receiver := vfbcContractAddress

		originalSenderBalance := getBalance(sender)
		suite.Require().NotZero(originalSenderBalance.Sign())

		originalReceiverBalance := getBalance(receiver)

		const sendAmount int64 = 10000

		inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("transfer", receiver, big.NewInt(sendAmount))
		suite.Require().NoError(err)
		ctx := suite.Ctx()

		msgEthereumTx := evmtypes.NewTx(
			suite.CITS.ChainApp.EvmKeeper().ChainID(),
			suite.CITS.ChainApp.EvmKeeper().GetNonce(ctx, sender),
			&normalErc20ContractAddress,
			nil,
			100_000, // gas limit
			nil,
			suite.CITS.ChainApp.FeeMarketKeeper().GetBaseFee(ctx),
			big.NewInt(1),
			inputCallData,
			&ethtypes.AccessList{},
		)
		msgEthereumTx.From = sender.String()

		// The transaction is expected to be successful because:
		// 1. It modifies store of the normal ERC-20 contract
		// 2. It does not modify any store of the VFBC contract
		// 3. It does not change the VFBC account state like nonce, native coin balance, code, storage

		resp, err := suite.CITS.DeliverEthTx(tokenOwner, msgEthereumTx)
		suite.Require().NoError(err)

		suite.Commit()

		receipt := suite.GetTxReceipt(common.HexToHash(resp.EthTxHash))
		suite.Require().NotNil(receipt)

		suite.Equal(ethtypes.ReceiptStatusSuccessful, receipt.Status, "tx must be successful")
		suite.Equal(new(big.Int).Sub(originalSenderBalance, big.NewInt(sendAmount)), getBalance(sender))
		suite.Equal(new(big.Int).Add(originalReceiverBalance, big.NewInt(sendAmount)), getBalance(receiver))
	})

	suite.Run("replay protection, nonce must be increased after success tx", func() {
		from := senderOfVfbc.GetEthAddress()

		originalNonce := suite.CITS.ChainApp.EvmKeeper().GetNonce(suite.Ctx(), from)

		inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("transfer", receiver.GetEthAddress(), common.Big1)
		suite.Require().NoError(err)

		buildTx := func(nonce uint64) *evmtypes.MsgEthereumTx {
			msgEthereumTx := evmtypes.NewTx(
				suite.CITS.ChainApp.EvmKeeper().ChainID(),
				originalNonce,
				&vfbcContractAddress,
				nil,
				100_000, // gas limit
				nil,
				suite.CITS.ChainApp.FeeMarketKeeper().GetBaseFee(suite.Ctx()),
				common.Big1,
				inputCallData,
				&ethtypes.AccessList{},
			)
			msgEthereumTx.From = from.String()

			return msgEthereumTx
		}

		resp1, err := suite.CITS.DeliverEthTx(senderOfVfbc, buildTx(originalNonce))
		suite.Commit() // commit to passive trigger EVM Tx indexer

		suite.Require().NoError(err)
		suite.Require().NotNil(resp1)

		receipt1 := suite.GetTxReceipt(common.HexToHash(resp1.EthTxHash))
		suite.Require().NotNil(receipt1)
		suite.Equal(ethtypes.ReceiptStatusSuccessful, receipt1.Status, "tx must be successful")

		suite.Equal(originalNonce+1, suite.CITS.ChainApp.EvmKeeper().GetNonce(suite.Ctx(), from), "nonce must be increased")

		// now send again with the same nonce
		resp2, err := suite.CITS.DeliverEthTx(senderOfVfbc, buildTx(originalNonce))
		suite.Commit() // commit to passive trigger EVM Tx indexer

		suite.Require().Error(err)
		suite.Require().NotNil(resp2)

		receipt2, err := suite.CITS.RpcBackend.GetTransactionReceipt(common.HexToHash(resp2.EthTxHash))
		suite.Require().NoError(err)
		suite.Require().Nil(receipt2, "tx must not be executed")
	})

	suite.Run("replay protection, nonce must be increased after failed tx", func() {
		from := senderOfVfbc.GetEthAddress()

		originalNonce := suite.CITS.ChainApp.EvmKeeper().GetNonce(suite.Ctx(), from)

		inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack(
			"transfer",
			receiver.GetEthAddress(),
			suite.App().BankKeeper().GetSupply(suite.Ctx(), suite.CITS.ChainConstantsConfig.GetMinDenom()).Amount.BigInt(), // no-way enough
		)
		suite.Require().NoError(err)

		buildTx := func(nonce uint64) *evmtypes.MsgEthereumTx {
			msgEthereumTx := evmtypes.NewTx(
				suite.CITS.ChainApp.EvmKeeper().ChainID(),
				originalNonce,
				&vfbcContractAddress,
				nil,
				100_000, // gas limit
				nil,
				suite.CITS.ChainApp.FeeMarketKeeper().GetBaseFee(suite.Ctx()),
				common.Big1,
				inputCallData,
				&ethtypes.AccessList{},
			)
			msgEthereumTx.From = from.String()

			return msgEthereumTx
		}

		resp1, err := suite.CITS.DeliverEthTx(senderOfVfbc, buildTx(originalNonce))
		suite.Commit() // commit to passive trigger EVM Tx indexer

		suite.Require().Error(err)
		suite.Require().NotNil(resp1)

		receipt1 := suite.GetTxReceipt(common.HexToHash(resp1.EthTxHash))
		suite.Require().NotNil(receipt1)
		suite.Equal(ethtypes.ReceiptStatusFailed, receipt1.Status, "tx must be failed")

		suite.Equal(originalNonce+1, suite.CITS.ChainApp.EvmKeeper().GetNonce(suite.Ctx(), from), "nonce must be increased")

		// now send again with the same nonce
		resp2, err := suite.CITS.DeliverEthTx(senderOfVfbc, buildTx(originalNonce))
		suite.Commit() // commit to passive trigger EVM Tx indexer

		suite.Require().Error(err)
		suite.Require().NotNil(resp2)

		receipt2, err := suite.CITS.RpcBackend.GetTransactionReceipt(common.HexToHash(resp2.EthTxHash))
		suite.Require().NoError(err)
		suite.Require().Nil(receipt2, "tx must not be executed")
	})

	suite.Run("replay protection, chain-id must match", func() {
		from := senderOfVfbc.GetEthAddress()

		originalNonce := suite.CITS.ChainApp.EvmKeeper().GetNonce(suite.Ctx(), from)

		inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("transfer", receiver.GetEthAddress(), common.Big1)
		suite.Require().NoError(err)

		buildTx := func(nonce uint64) *evmtypes.MsgEthereumTx {
			msgEthereumTx := evmtypes.NewTx(
				new(big.Int).Add(suite.CITS.ChainApp.EvmKeeper().ChainID(), common.Big1),
				originalNonce,
				&vfbcContractAddress,
				nil,
				100_000, // gas limit
				nil,
				suite.CITS.ChainApp.FeeMarketKeeper().GetBaseFee(suite.Ctx()),
				common.Big1,
				inputCallData,
				&ethtypes.AccessList{},
			)
			msgEthereumTx.From = from.String()

			return msgEthereumTx
		}

		tx := buildTx(originalNonce)
		resp, err := suite.CITS.DeliverEthTx(senderOfVfbc, tx)
		suite.Commit() // commit to passive trigger EVM Tx indexer

		suite.Require().Error(err)
		suite.Require().Nil(resp)

		suite.Equal(originalNonce, suite.CITS.ChainApp.EvmKeeper().GetNonce(suite.Ctx(), from), "nonce must be kept, as tx will not be executed")

		receipt, err := suite.CITS.RpcBackend.GetTransactionReceipt(tx.AsTransaction().Hash())
		suite.Require().NoError(err)
		suite.Require().Nil(receipt, "tx must not be executed")
	})
}
