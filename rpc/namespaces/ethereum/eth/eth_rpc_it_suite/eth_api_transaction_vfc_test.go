package demo

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/ethermint/integration_test_util"
	rpctypes "github.com/evmos/ethermint/rpc/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"math/big"
)

func (suite *EthRpcTestSuite) Test_AllGetTransaction_VFC() {
	deployer := suite.CITS.WalletAccounts.Number(1)
	tokenOwner := suite.CITS.WalletAccounts.Number(2)
	senderOfVfbc := suite.CITS.WalletAccounts.Number(3)
	receiver := suite.CITS.WalletAccounts.Number(4)

	suite.CITS.MintCoin(tokenOwner, suite.CITS.NewBaseCoin(10))
	suite.CITS.MintCoin(senderOfVfbc, suite.CITS.NewBaseCoin(10))

	normalErc20ContractAddress, _, _, err := suite.CITS.TxDeploy2WDymContract(deployer, tokenOwner)
	suite.Require().NoError(err)

	vfbcContractAddress, found := suite.App().EvmKeeper().GetVirtualFrontierBankContractAddressByDenom(suite.Ctx(), suite.CITS.ChainConstantsConfig.GetMinDenom())
	suite.Require().True(found)

	suite.CITS.WaitNextBlockOrCommit()

	txSendErc20, err := suite.CITS.TxTransferErc20TokenAsync(normalErc20ContractAddress, tokenOwner, receiver, 1000, 0)
	suite.Require().NoError(err)

	txSendVfbc, err := suite.CITS.TxTransferErc20TokenAsync(vfbcContractAddress, senderOfVfbc, receiver, 1000, 0)
	suite.Require().NoError(err)

	suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

	ethPublicApi := suite.GetEthPublicAPI()

	rpcTxSentErc20, err := ethPublicApi.GetTransactionByHash(txSendErc20.AsTransaction().Hash())
	suite.Require().NoError(err)
	suite.Require().NotNil(rpcTxSentErc20)

	rpcTxSentVfbc, err := ethPublicApi.GetTransactionByHash(txSendVfbc.AsTransaction().Hash())
	suite.Require().NoError(err)
	suite.Require().NotNil(rpcTxSentVfbc)

	suite.Require().Equal(rpcTxSentErc20.BlockNumber, rpcTxSentVfbc.BlockNumber, "both transactions expected to be delivered within the same block")
	suite.Require().NotEqual(common.Hash{}, rpcTxSentErc20.BlockHash)
	suite.Require().NotEqual(*rpcTxSentErc20.TransactionIndex, *rpcTxSentVfbc.TransactionIndex, "expected to have different transaction index")

	receiptTxSentErc20 := suite.GetTxReceipt(txSendErc20.AsTransaction().Hash())
	receiptTxSentVfbc := suite.GetTxReceipt(txSendVfbc.AsTransaction().Hash())
	suite.Require().NoError(err)

	tmp1 := suite.Equal(ethtypes.ReceiptStatusSuccessful, receiptTxSentErc20.Status, "tx transfer ERC-20 must be success")
	tmp2 := suite.Equal(ethtypes.ReceiptStatusSuccessful, receiptTxSentVfbc.Status, "tx transfer via VFB contract must be success")
	if !tmp1 || !tmp2 {
		return
	}

	compareFieldsOfRpcTransactions := func(rpcTxSentErc20, rpcTxSentVfbc *rpctypes.RPCTransaction) {
		suite.Equal(rpcTxSentErc20.BlockHash, rpcTxSentVfbc.BlockHash)
		suite.Equal(rpcTxSentErc20.BlockNumber, rpcTxSentVfbc.BlockNumber)
		suite.Equal(tokenOwner.GetEthAddress(), rpcTxSentErc20.From)
		suite.Equal(senderOfVfbc.GetEthAddress(), rpcTxSentVfbc.From)
		suite.Equal(rpcTxSentErc20.Gas, rpcTxSentVfbc.Gas)
		suite.Equal(rpcTxSentErc20.GasPrice, rpcTxSentVfbc.GasPrice)
		suite.Equal(rpcTxSentErc20.GasFeeCap, rpcTxSentVfbc.GasFeeCap)
		suite.Equal(rpcTxSentErc20.GasTipCap, rpcTxSentVfbc.GasTipCap)
		suite.Equal(rpcTxSentErc20.Input, rpcTxSentVfbc.Input, "same receiver, same amount, so expected to have same input")
		suite.Equal(normalErc20ContractAddress, *rpcTxSentErc20.To)
		suite.Equal(vfbcContractAddress, *rpcTxSentVfbc.To)
		if suite.NotNil(rpcTxSentErc20.TransactionIndex) && suite.NotNil(rpcTxSentVfbc.TransactionIndex) {
			gotTxSentErc20TxIndex := uint64(*rpcTxSentErc20.TransactionIndex)
			gotTxSentVfbcTxIndex := uint64(*rpcTxSentVfbc.TransactionIndex)
			suite.True(gotTxSentErc20TxIndex == 0 || gotTxSentVfbcTxIndex == 0)
			suite.True(gotTxSentErc20TxIndex == 1 || gotTxSentVfbcTxIndex == 1)
		}
		suite.Equal(rpcTxSentErc20.Value, rpcTxSentVfbc.Value)
		suite.Equal(rpcTxSentErc20.Type, rpcTxSentVfbc.Type)
		suite.Equal(rpcTxSentErc20.Accesses, rpcTxSentVfbc.Accesses)
		suite.Equal(rpcTxSentErc20.ChainID, rpcTxSentVfbc.ChainID)
		suite.Equal(rpcTxSentErc20.V == nil, rpcTxSentVfbc.V == nil)
		suite.Equal(rpcTxSentErc20.R == nil, rpcTxSentVfbc.R == nil)
		suite.Equal(rpcTxSentErc20.S == nil, rpcTxSentVfbc.S == nil)
	}

	suite.Run("GetTransactionByHash", func() {
		gotTxSentErc20, err := ethPublicApi.GetTransactionByHash(txSendErc20.AsTransaction().Hash())
		suite.Require().NoError(err)
		suite.Require().NotNil(gotTxSentErc20)

		gotTxSentVfbc, err := ethPublicApi.GetTransactionByHash(txSendVfbc.AsTransaction().Hash())
		suite.Require().NoError(err)
		suite.Require().NotNil(gotTxSentVfbc)

		// compare fields returned equally
		compareFieldsOfRpcTransactions(gotTxSentErc20, gotTxSentVfbc)
	})

	suite.Run("GetTransactionByBlockHashAndIndex", func() {
		gotTxSentErc20, err := ethPublicApi.GetTransactionByBlockHashAndIndex(*rpcTxSentErc20.BlockHash, hexutil.Uint(*rpcTxSentErc20.TransactionIndex))
		suite.Require().NoError(err)
		suite.Require().NotNil(gotTxSentErc20)

		gotTxSentVfbc, err := ethPublicApi.GetTransactionByBlockHashAndIndex(*gotTxSentErc20.BlockHash, hexutil.Uint(*rpcTxSentVfbc.TransactionIndex))
		suite.Require().NoError(err)
		suite.Require().NotNil(gotTxSentVfbc)

		// compare fields returned equally
		compareFieldsOfRpcTransactions(gotTxSentErc20, gotTxSentVfbc)
	})

	suite.Run("GetTransactionByBlockNumberAndIndex", func() {
		gotTxSentErc20, err := ethPublicApi.GetTransactionByBlockNumberAndIndex(rpctypes.NewBlockNumber(rpcTxSentErc20.BlockNumber.ToInt()), hexutil.Uint(*rpcTxSentErc20.TransactionIndex))
		suite.Require().NoError(err)
		suite.Require().NotNil(gotTxSentErc20)

		gotTxSentVfbc, err := ethPublicApi.GetTransactionByBlockNumberAndIndex(rpctypes.NewBlockNumber(gotTxSentErc20.BlockNumber.ToInt()), hexutil.Uint(*rpcTxSentVfbc.TransactionIndex))
		suite.Require().NoError(err)
		suite.Require().NotNil(gotTxSentVfbc)

		// compare fields returned equally
		compareFieldsOfRpcTransactions(gotTxSentErc20, gotTxSentVfbc)
	})

	suite.Run("GetBlockTransactionCountByNumber", func() {
		count := ethPublicApi.GetBlockTransactionCountByNumber(rpctypes.NewBlockNumber(rpcTxSentVfbc.BlockNumber.ToInt()))
		suite.Equal(uint(2), uint(*count))
	})

	suite.Run("GetBlockTransactionCountByHash", func() {
		count := ethPublicApi.GetBlockTransactionCountByHash(*rpcTxSentVfbc.BlockHash)
		suite.Equal(uint(2), uint(*count))
	})

	suite.Run("GetTransactionCount", func() {
		count1, err := ethPublicApi.GetTransactionCount(tokenOwner.GetEthAddress(), rpctypes.BlockNumberOrHash{BlockHash: rpcTxSentErc20.BlockHash})
		suite.Require().NoError(err)
		suite.Equal(hexutil.Uint64(1), *count1)

		count2, err := ethPublicApi.GetTransactionCount(senderOfVfbc.GetEthAddress(), rpctypes.BlockNumberOrHash{BlockHash: rpcTxSentVfbc.BlockHash})
		suite.Require().NoError(err)
		suite.Equal(hexutil.Uint64(1), *count2)
	})

	compareFieldsOfLogs := func(logOfSentErc20, logOfTxSentVfbc *ethtypes.Log) {
		suite.Equal(normalErc20ContractAddress, logOfSentErc20.Address)
		suite.Equal(vfbcContractAddress, logOfTxSentVfbc.Address)
		suite.Len(logOfSentErc20.Topics, 3)
		suite.Len(logOfTxSentVfbc.Topics, 3)
		if suite.Equal(logOfSentErc20.Topics[0], logOfTxSentVfbc.Topics[0], "must fires Transfer event") {
			suite.Equal(common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"), logOfTxSentVfbc.Topics[0], "must be transfer event")
		}
		suite.NotEqual(common.Hash{}, logOfSentErc20.Topics[1])
		suite.NotEqual(common.Hash{}, logOfTxSentVfbc.Topics[1])
		suite.Equal(logOfSentErc20.Topics[2], logOfTxSentVfbc.Topics[2], "same receiver so expected to have same topic value")
		suite.Equal(logOfSentErc20.Data, logOfTxSentVfbc.Data, "same amount so data must be the same")
		suite.Equal(logOfSentErc20.BlockHash, logOfTxSentVfbc.BlockHash)
		suite.Equal(logOfSentErc20.BlockNumber, logOfTxSentVfbc.BlockNumber)
		suite.Equal(rpcTxSentErc20.Hash, logOfSentErc20.TxHash)
		suite.Equal(rpcTxSentVfbc.Hash, logOfTxSentVfbc.TxHash)
		suite.Equal(uint64(*rpcTxSentErc20.TransactionIndex), uint64(logOfSentErc20.TxIndex))
		suite.Equal(uint64(*rpcTxSentVfbc.TransactionIndex), uint64(logOfTxSentVfbc.TxIndex))
		suite.Equal(uint64(*rpcTxSentErc20.TransactionIndex), uint64(logOfSentErc20.Index)) // single log so index equals to tx index
		suite.Equal(uint64(*rpcTxSentVfbc.TransactionIndex), uint64(logOfTxSentVfbc.Index)) // single log so index equals to tx index
		suite.False(logOfSentErc20.Removed)
		suite.False(logOfTxSentVfbc.Removed)
	}

	suite.Run("GetTransactionLogs", func() {
		gotTxSentErc20Logs, err := ethPublicApi.GetTransactionLogs(rpcTxSentErc20.Hash)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(gotTxSentErc20Logs)

		gotTxSentVfbcLogs, err := ethPublicApi.GetTransactionLogs(rpcTxSentVfbc.Hash)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(gotTxSentVfbcLogs)

		suite.Len(gotTxSentErc20Logs, 1)
		suite.Len(gotTxSentVfbcLogs, 1)

		// compare fields returned equally
		compareFieldsOfLogs(gotTxSentErc20Logs[0], gotTxSentVfbcLogs[0])
	})

	compareFieldsOfReceipts := func(receiptOfSentErc20, receiptOfTxSentVfbc map[string]interface{}) {
		suite.Equal(uint(ethtypes.ReceiptStatusSuccessful), uint(receiptOfSentErc20["status"].(hexutil.Uint)))
		suite.Equal(uint(ethtypes.ReceiptStatusSuccessful), uint(receiptOfTxSentVfbc["status"].(hexutil.Uint)))
		suite.NotZero(receiptOfSentErc20["cumulativeGasUsed"])
		suite.NotZero(receiptOfTxSentVfbc["cumulativeGasUsed"])
		suite.Equal(ethtypes.BytesToBloom(ethtypes.LogsBloom(receiptOfSentErc20["logs"].([]*ethtypes.Log))), receiptOfSentErc20["logsBloom"])
		suite.Equal(ethtypes.BytesToBloom(ethtypes.LogsBloom(receiptOfTxSentVfbc["logs"].([]*ethtypes.Log))), receiptOfTxSentVfbc["logsBloom"])
		suite.Len(receiptOfSentErc20["logs"], 1)
		suite.Len(receiptOfTxSentVfbc["logs"], 1)
		compareFieldsOfLogs(receiptOfSentErc20["logs"].([]*ethtypes.Log)[0], receiptOfTxSentVfbc["logs"].([]*ethtypes.Log)[0])
		suite.Equal(txSendErc20.AsTransaction().Hash(), receiptOfSentErc20["transactionHash"])
		suite.Equal(txSendVfbc.AsTransaction().Hash(), receiptOfTxSentVfbc["transactionHash"])
		suite.Nil(receiptOfSentErc20["contractAddress"])
		suite.Nil(receiptOfTxSentVfbc["contractAddress"])
		if suite.NotZero(receiptOfSentErc20["gasUsed"]) {
			suite.Less(uint64(receiptOfSentErc20["gasUsed"].(hexutil.Uint64)), txSendErc20.AsTransaction().Gas())
		}
		if suite.NotZero(receiptOfTxSentVfbc["gasUsed"]) {
			suite.Less(uint64(receiptOfTxSentVfbc["gasUsed"].(hexutil.Uint64)), txSendVfbc.AsTransaction().Gas())
		}
		suite.Equal(receiptOfSentErc20["blockHash"], receiptOfTxSentVfbc["blockHash"])
		suite.Equal(receiptOfSentErc20["blockNumber"], receiptOfTxSentVfbc["blockNumber"])
		if suite.NotNil(receiptOfSentErc20["transactionIndex"]) && suite.NotNil(receiptOfTxSentVfbc["transactionIndex"]) {
			gotTxSentErc20TxIndex := uint64(receiptOfSentErc20["transactionIndex"].(hexutil.Uint64))
			gotTxSentVfbcTxIndex := uint64(receiptOfTxSentVfbc["transactionIndex"].(hexutil.Uint64))
			suite.True(gotTxSentErc20TxIndex == 0 || gotTxSentVfbcTxIndex == 0)
			suite.True(gotTxSentErc20TxIndex == 1 || gotTxSentVfbcTxIndex == 1)
		}
		suite.Equal(tokenOwner.GetEthAddress(), receiptOfSentErc20["from"])
		suite.Equal(senderOfVfbc.GetEthAddress(), receiptOfTxSentVfbc["from"])
		suite.Equal(normalErc20ContractAddress, *(receiptOfSentErc20["to"].(*common.Address)))
		suite.Equal(vfbcContractAddress, *(receiptOfTxSentVfbc["to"].(*common.Address)))
		suite.Equal(receiptOfSentErc20["type"], receiptOfTxSentVfbc["type"])
	}

	suite.Run("GetTransactionReceipt", func() {
		gotTxSentErc20Receipt, err := ethPublicApi.GetTransactionReceipt(rpcTxSentErc20.Hash)
		suite.Require().NoError(err)
		suite.Require().NotNil(gotTxSentErc20Receipt)

		gotTxSentVfbcReceipt, err := ethPublicApi.GetTransactionReceipt(rpcTxSentVfbc.Hash)
		suite.Require().NoError(err)
		suite.Require().NotNil(gotTxSentVfbcReceipt)

		gotTxSentErc20Logs, err := ethPublicApi.GetTransactionLogs(rpcTxSentErc20.Hash)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(gotTxSentErc20Logs)

		gotTxSentVfbcLogs, err := ethPublicApi.GetTransactionLogs(rpcTxSentVfbc.Hash)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(gotTxSentVfbcLogs)

		suite.Equal(gotTxSentErc20Receipt["logs"], gotTxSentErc20Logs)
		suite.Equal(gotTxSentVfbcReceipt["logs"], gotTxSentVfbcLogs)

		// compare fields returned equally
		compareFieldsOfReceipts(gotTxSentErc20Receipt, gotTxSentVfbcReceipt)

		// check gas of single tx, per block, call to vfbc
		suite.Commit()
		txSendVfbc2, err := suite.CITS.TxTransferErc20TokenAsync(vfbcContractAddress, senderOfVfbc, receiver, 1000, 0)
		suite.Require().NoError(err)

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		gotTxSentVfbcReceipt2 := suite.GetTxReceipt(txSendVfbc2.AsTransaction().Hash())
		suite.Require().NotNil(gotTxSentVfbcReceipt2)

		suite.NotZero(gotTxSentVfbcReceipt2.CumulativeGasUsed)
		suite.NotZero(gotTxSentVfbcReceipt2.GasUsed)
		suite.Less(gotTxSentVfbcReceipt2.CumulativeGasUsed, txSendVfbc2.AsTransaction().Gas())
		suite.Less(gotTxSentVfbcReceipt2.GasUsed, txSendVfbc2.AsTransaction().Gas())
	})

	transferViaVbfc := func(gas uint64, amount *big.Int) *evmtypes.MsgEthereumTx {
		suite.CITS.WaitNextBlockOrCommit()

		inputCallData, err := integration_test_util.ERC20MinterBurnerDecimalsContract.ABI.Pack("transfer", receiver.GetEthAddress(), amount)
		suite.Require().NoError(err)

		from := senderOfVfbc.GetEthAddress()

		ctx := suite.Ctx()

		msgEthereumTx := evmtypes.NewTx(
			suite.CITS.ChainApp.EvmKeeper().ChainID(),
			suite.CITS.ChainApp.EvmKeeper().GetNonce(ctx, from),
			&vfbcContractAddress,
			nil,
			gas,
			nil,
			suite.CITS.ChainApp.FeeMarketKeeper().GetBaseFee(ctx),
			big.NewInt(1),
			inputCallData,
			&ethtypes.AccessList{},
		)
		msgEthereumTx.From = from.String()

		err = suite.CITS.DeliverEthTxAsync(senderOfVfbc, msgEthereumTx)
		suite.Require().NoError(err)

		suite.Commit()

		return msgEthereumTx
	}

	const intrinsicGas = 21572
	const exactGasConsumption = intrinsicGas + evmtypes.VFBCopgTransfer

	testsGas := []struct {
		name        string
		gas         uint64
		amount      *big.Int
		wantSuccess bool
		assertFunc  func(gas, gasUsed uint64)
	}{
		{
			name:        "current implementation, tx consumes exactly amount of gas",
			gas:         exactGasConsumption,
			amount:      big.NewInt(1),
			wantSuccess: true,
			assertFunc: func(_ uint64, gasUsed uint64) {
				suite.Equal(exactGasConsumption, gasUsed)
			},
		},
		{
			name:        "current implementation, tx consumes exactly amount of gas",
			gas:         exactGasConsumption + 1,
			amount:      big.NewInt(1),
			wantSuccess: true,
			assertFunc: func(_ uint64, gasUsed uint64) {
				suite.Equal(exactGasConsumption, gasUsed)
			},
		},
		{
			name:        "less gas than expected, tx must be failed",
			gas:         22000,
			amount:      big.NewInt(1),
			wantSuccess: false,
			assertFunc: func(gas uint64, gasUsed uint64) {
				suite.Equal(gas, gasUsed, "expected to consume all gas on this failed txs")
			},
		},
		{
			name:        "success, tx gas above enough a bit",
			gas:         uint64(float64(exactGasConsumption) * 1.5),
			amount:      big.NewInt(1),
			wantSuccess: true,
			assertFunc: func(gas uint64, gasUsed uint64) {
				suite.Less(gasUsed, gas, "consumed gas must be less than gas limit")
				suite.Greater(gasUsed, gas/2, "consumed gas expected to greater than minimum consumption")
			},
		},
		{
			name:        "success, tx gas over 2x of required",
			gas:         300000,
			amount:      big.NewInt(1),
			wantSuccess: true,
			assertFunc: func(gas uint64, gasUsed uint64) {
				suite.Less(gasUsed, gas, "consumed gas must be less than gas limit")
				suite.Equal(gas/2, gasUsed, "consumed gas expected to equals minimum consumption because actual tx consumption is less than half of max")
			},
		},
		{
			name:        "failed txs consumes at least half of gas, no matter how much it is",
			gas:         6_000_000,
			amount:      suite.CITS.QueryBalance(0, senderOfVfbc.GetCosmosAddress().String()).Amount.AddRaw(1).BigInt(), // more than balance
			wantSuccess: false,
			assertFunc: func(gas uint64, gasUsed uint64) {
				suite.Equal(gas/2, gasUsed, "expected to consume at least half of gas on failed txs")
			},
		},
	}

	for _, tt := range testsGas {
		suite.Run(tt.name, func() {
			tx := transferViaVbfc(tt.gas, tt.amount)

			gotTxSentVfbcReceipt3 := suite.GetTxReceipt(tx.AsTransaction().Hash())
			suite.Require().NotNil(gotTxSentVfbcReceipt3)

			if tt.wantSuccess {
				suite.Require().Equal(ethtypes.ReceiptStatusSuccessful, gotTxSentVfbcReceipt3.Status, "tx transfer via VFB contract must be success")
			} else {
				suite.Require().Equal(ethtypes.ReceiptStatusFailed, gotTxSentVfbcReceipt3.Status, "tx transfer via VFB contract must be failed")
			}

			suite.Require().NotZero(gotTxSentVfbcReceipt3.CumulativeGasUsed)
			suite.Require().NotZero(gotTxSentVfbcReceipt3.GasUsed)
			suite.Require().Equal(gotTxSentVfbcReceipt3.CumulativeGasUsed, gotTxSentVfbcReceipt3.GasUsed, "single tx, cumulative gas used must be equal to gas used")

			if !tt.wantSuccess {
				suite.Require().LessOrEqual(tt.gas/2, gotTxSentVfbcReceipt3.GasUsed, "expected to consume at least half of gas on failed txs")
			}

			tt.assertFunc(tt.gas, gotTxSentVfbcReceipt3.GasUsed)
		})
	}

	suite.Run("estimate gas", func() {
		tx := transferViaVbfc(100_000, common.Big1)
		ethTx := tx.AsTransaction()

		from := senderOfVfbc.GetEthAddress()
		input := hexutil.Bytes(ethTx.Data())
		chainId := hexutil.Big(*ethTx.ChainId())
		gas := hexutil.Uint64(ethTx.Gas())

		estimateGas, err := ethPublicApi.EstimateGas(evmtypes.TransactionArgs{
			From:                 &from,
			To:                   &vfbcContractAddress,
			Gas:                  &gas,
			GasPrice:             nil,
			MaxFeePerGas:         nil,
			MaxPriorityFeePerGas: nil,
			Value:                nil,
			Nonce:                nil,
			Input:                &input,
			AccessList:           nil,
			ChainID:              &chainId,
		}, nil)
		suite.Require().NoError(err)

		est := uint64(estimateGas)
		if suite.NotZero(est) {
			suite.Equal(exactGasConsumption, est)
		}
	})
}
