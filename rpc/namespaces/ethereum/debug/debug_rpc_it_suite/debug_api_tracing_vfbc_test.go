package demo

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/ethermint/integration_test_util"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"math"
	"math/big"
	"strings"
)

func (suite *DebugRpcTestSuite) TestTracingVirtualFrontierBankContract() {
	deployer := suite.CITS.WalletAccounts.Number(1)
	tokenOwner := suite.CITS.WalletAccounts.Number(2)
	senderOfVfbc := suite.CITS.WalletAccounts.Number(3)
	receiver := integration_test_util.NewTestAccount(suite.T(), nil)

	suite.CITS.MintCoin(tokenOwner, suite.CITS.NewBaseCoin(10))
	suite.CITS.MintCoin(senderOfVfbc, suite.CITS.NewBaseCoin(10))

	normalErc20ContractAddress, _, _, err := suite.CITS.TxDeploy2WDymContract(deployer, tokenOwner)
	suite.Require().NoError(err)

	vfbcContractAddress, found := suite.App().EvmKeeper().GetVirtualFrontierBankContractAddressByDenom(suite.Ctx(), suite.CITS.ChainConstantsConfig.GetMinDenom())
	suite.Require().True(found)

	suite.CITS.WaitNextBlockOrCommit()

	callTracer := func(hash common.Hash) callFrame {
		traceErc20, err := suite.GetDebugAPI().TraceTransaction(hash, &evmtypes.TraceConfig{
			Tracer: "callTracer",
		})
		suite.Require().NoError(err)
		suite.Require().NotNil(traceErc20)

		bz, err := json.Marshal(traceErc20)
		suite.Require().NoError(err)

		var callFrameRoot callFrame
		err = json.Unmarshal(bz, &callFrameRoot)
		suite.Require().NoError(err)

		return callFrameRoot
	}

	suite.Run("trace of success transfer(address,uint256)", func() {
		suite.CITS.WaitNextBlockOrCommit()

		txSendErc20, err := suite.CITS.TxTransferErc20TokenAsync(normalErc20ContractAddress, tokenOwner, receiver, 1000, 0)
		suite.Require().NoError(err)
		ethTxSendErc20 := txSendErc20.AsTransaction()

		txSendVfbc, err := suite.CITS.TxTransferErc20TokenAsync(vfbcContractAddress, senderOfVfbc, receiver, 1000, 0)
		suite.Require().NoError(err)
		ethTxSendVfbc := txSendVfbc.AsTransaction()

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		receiptSentErc20 := suite.GetTxReceipt(ethTxSendErc20.Hash())
		receiptSentVfbc := suite.GetTxReceipt(ethTxSendVfbc.Hash())

		suite.Require().Equal(receiptSentErc20.BlockHash, receiptSentVfbc.BlockHash, "both txs expected to be included in the same block")
		suite.Require().Equal(ethtypes.ReceiptStatusSuccessful, receiptSentErc20.Status)
		suite.Require().Equal(ethtypes.ReceiptStatusSuccessful, receiptSentVfbc.Status)

		callTracerSentErc20 := callTracer(ethTxSendErc20.Hash())
		callTracerSentVfbc := callTracer(ethTxSendVfbc.Hash())

		suite.Equal(callTracerSentErc20.Type, callTracerSentVfbc.Type)
		suite.Equal(strings.ToLower(tokenOwner.GetEthAddress().String()), callTracerSentErc20.From)
		suite.Equal(strings.ToLower(senderOfVfbc.GetEthAddress().String()), callTracerSentVfbc.From)
		suite.Equal(strings.ToLower(normalErc20ContractAddress.String()), callTracerSentErc20.To)
		suite.Equal(strings.ToLower(vfbcContractAddress.String()), callTracerSentVfbc.To)
		suite.Equal(callTracerSentErc20.Value, callTracerSentVfbc.Value)
		suite.Equal(callTracerSentErc20.Gas, callTracerSentVfbc.Gas, "input gas must be the same")
		suite.Equal(fmt.Sprintf("0x%x", 13735), callTracerSentErc20.GasUsed)
		suite.Equal(fmt.Sprintf("0x%x", evmtypes.VFBCopgTransfer), callTracerSentVfbc.GasUsed)
		suite.Equal(callTracerSentErc20.Input, callTracerSentVfbc.Input)
		suite.Equal(callTracerSentErc20.Output, callTracerSentVfbc.Output)
		suite.Empty(callTracerSentErc20.Error)
		suite.Empty(callTracerSentVfbc.Error)
		suite.Empty(callTracerSentErc20.Calls)
		suite.Empty(callTracerSentVfbc.Calls)
	})

	suite.Run("trace of fail transfer(address,uint256)", func() {
		suite.CITS.WaitNextBlockOrCommit()

		txSendErc20, err := suite.CITS.TxTransferErc20TokenAsync(normalErc20ContractAddress, tokenOwner, receiver, math.MaxUint16, 18)
		suite.Require().NoError(err)
		ethTxSendErc20 := txSendErc20.AsTransaction()

		txSendVfbc, err := suite.CITS.TxTransferErc20TokenAsync(vfbcContractAddress, senderOfVfbc, receiver, math.MaxUint16, 18)
		suite.Require().NoError(err)
		ethTxSendVfbc := txSendVfbc.AsTransaction()

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		receiptSentErc20 := suite.GetTxReceipt(ethTxSendErc20.Hash())
		receiptSentVfbc := suite.GetTxReceipt(ethTxSendVfbc.Hash())

		suite.Require().Equal(receiptSentErc20.BlockHash, receiptSentVfbc.BlockHash, "both txs expected to be included in the same block")
		suite.Require().Equal(ethtypes.ReceiptStatusFailed, receiptSentErc20.Status)
		suite.Require().Equal(ethtypes.ReceiptStatusFailed, receiptSentVfbc.Status)

		callTracerSentErc20 := callTracer(ethTxSendErc20.Hash())
		callTracerSentVfbc := callTracer(ethTxSendVfbc.Hash())

		suite.Equal(callTracerSentErc20.Type, callTracerSentVfbc.Type)
		suite.Equal(strings.ToLower(tokenOwner.GetEthAddress().String()), callTracerSentErc20.From)
		suite.Equal(strings.ToLower(senderOfVfbc.GetEthAddress().String()), callTracerSentVfbc.From)
		suite.Equal(strings.ToLower(normalErc20ContractAddress.String()), callTracerSentErc20.To)
		suite.Equal(strings.ToLower(vfbcContractAddress.String()), callTracerSentVfbc.To)
		suite.Equal(callTracerSentErc20.Value, callTracerSentVfbc.Value)
		suite.Equal(callTracerSentErc20.Gas, callTracerSentVfbc.Gas, "input gas must be the same")
		suite.Equal(fmt.Sprintf("0x%x", 3372), callTracerSentErc20.GasUsed)
		suite.Equal(fmt.Sprintf("0x%x", evmtypes.VFBCopgTransfer_Revert), callTracerSentVfbc.GasUsed)
		suite.Equal(callTracerSentErc20.Input, callTracerSentVfbc.Input)
		suite.Equal(callTracerSentErc20.Output, callTracerSentVfbc.Output)
		suite.Equal(callTracerSentErc20.Error, "execution reverted")
		suite.Equal(callTracerSentVfbc.Error, "execution reverted")
		suite.Empty(callTracerSentErc20.Calls)
		suite.Empty(callTracerSentVfbc.Calls)
	})

	sender := suite.CITS.WalletAccounts.Number(1)

	buildCustomTx := func(contract common.Address, _4bytes string, extraArgs ...string) *evmtypes.MsgEthereumTx {
		inputCallData, err := hex.DecodeString(_4bytes + strings.Join(extraArgs, ""))
		suite.Require().NoError(err)

		from := sender.GetEthAddress()

		ctx := suite.Ctx()

		msgEthereumTx := evmtypes.NewTx(
			suite.CITS.ChainApp.EvmKeeper().ChainID(),
			suite.CITS.ChainApp.EvmKeeper().GetNonce(ctx, from),
			&contract,
			nil,
			30_000,
			nil,
			suite.CITS.ChainApp.FeeMarketKeeper().GetBaseFee(ctx),
			big.NewInt(1),
			inputCallData,
			&ethtypes.AccessList{},
		)
		msgEthereumTx.From = from.String()

		return msgEthereumTx
	}

	suite.Run("trace of success decimals()", func() {
		suite.CITS.WaitNextBlockOrCommit()

		txErc20 := buildCustomTx(normalErc20ContractAddress, "313ce567")
		_, err := suite.CITS.DeliverEthTx(sender, txErc20)
		suite.Require().NoError(err)
		ethTxErc20 := txErc20.AsTransaction()

		txVfbc := buildCustomTx(vfbcContractAddress, "313ce567")
		_, err = suite.CITS.DeliverEthTx(sender, txVfbc)
		suite.Require().NoError(err)
		ethTxVfbc := txVfbc.AsTransaction()

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		receiptErc20 := suite.GetTxReceipt(ethTxErc20.Hash())
		receiptVfbc := suite.GetTxReceipt(ethTxVfbc.Hash())

		suite.Require().Equal(ethtypes.ReceiptStatusSuccessful, receiptErc20.Status)
		suite.Require().Equal(ethtypes.ReceiptStatusSuccessful, receiptVfbc.Status)

		callTracerErc20 := callTracer(ethTxErc20.Hash())
		callTracerVfbc := callTracer(ethTxVfbc.Hash())

		suite.Equal(callTracerErc20.Type, callTracerVfbc.Type)
		suite.Equal(strings.ToLower(sender.GetEthAddress().String()), callTracerErc20.From)
		suite.Equal(strings.ToLower(sender.GetEthAddress().String()), callTracerVfbc.From)
		suite.Equal(strings.ToLower(normalErc20ContractAddress.String()), callTracerErc20.To)
		suite.Equal(strings.ToLower(vfbcContractAddress.String()), callTracerVfbc.To)
		suite.Equal(callTracerErc20.Value, callTracerVfbc.Value)
		suite.Equal(callTracerErc20.Gas, callTracerVfbc.Gas, "input gas must be the same")
		suite.Equal(fmt.Sprintf("0x%x", 380), callTracerErc20.GasUsed)
		suite.Equal(fmt.Sprintf("0x%x", evmtypes.VFBCopgDecimals), callTracerVfbc.GasUsed)
		suite.Equal(callTracerErc20.Input, callTracerVfbc.Input)
		suite.Equal("0x0000000000000000000000000000000000000000000000000000000000000000", callTracerErc20.Output)
		suite.Equal( /*18*/ "0x0000000000000000000000000000000000000000000000000000000000000012", callTracerVfbc.Output)
		suite.Empty(callTracerErc20.Error)
		suite.Empty(callTracerVfbc.Error)
		suite.Empty(callTracerErc20.Calls)
		suite.Empty(callTracerVfbc.Calls)
	})

	suite.Run("trace of success name()", func() {
		suite.CITS.WaitNextBlockOrCommit()

		txErc20 := buildCustomTx(normalErc20ContractAddress, "06fdde03")
		_, err := suite.CITS.DeliverEthTx(sender, txErc20)
		suite.Require().NoError(err)
		ethTxErc20 := txErc20.AsTransaction()

		txVfbc := buildCustomTx(vfbcContractAddress, "06fdde03")
		_, err = suite.CITS.DeliverEthTx(sender, txVfbc)
		suite.Require().NoError(err)
		ethTxVfbc := txVfbc.AsTransaction()

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		receiptErc20 := suite.GetTxReceipt(ethTxErc20.Hash())
		receiptVfbc := suite.GetTxReceipt(ethTxVfbc.Hash())

		suite.Require().Equal(ethtypes.ReceiptStatusSuccessful, receiptErc20.Status)
		suite.Require().Equal(ethtypes.ReceiptStatusSuccessful, receiptVfbc.Status)

		callTracerErc20 := callTracer(ethTxErc20.Hash())
		callTracerVfbc := callTracer(ethTxVfbc.Hash())

		suite.Equal(callTracerErc20.Type, callTracerVfbc.Type)
		suite.Equal(strings.ToLower(sender.GetEthAddress().String()), callTracerErc20.From)
		suite.Equal(strings.ToLower(sender.GetEthAddress().String()), callTracerVfbc.From)
		suite.Equal(strings.ToLower(normalErc20ContractAddress.String()), callTracerErc20.To)
		suite.Equal(strings.ToLower(vfbcContractAddress.String()), callTracerVfbc.To)
		suite.Equal(callTracerErc20.Value, callTracerVfbc.Value)
		suite.Equal(callTracerErc20.Gas, callTracerVfbc.Gas, "input gas must be the same")
		suite.Equal(fmt.Sprintf("0x%x", 3455), callTracerErc20.GasUsed)
		suite.Equal(fmt.Sprintf("0x%x", evmtypes.VFBCopgName), callTracerVfbc.GasUsed)
		suite.Equal(callTracerErc20.Input, callTracerVfbc.Input)
		suite.Equal("0x000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000045744594d00000000000000000000000000000000000000000000000000000000", callTracerErc20.Output)
		suite.Equal("0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000344594d0000000000000000000000000000000000000000000000000000000000", callTracerVfbc.Output)
		suite.Empty(callTracerErc20.Error)
		suite.Empty(callTracerVfbc.Error)
		suite.Empty(callTracerErc20.Calls)
		suite.Empty(callTracerVfbc.Calls)
	})

	suite.Run("trace of success symbol()", func() {
		suite.CITS.WaitNextBlockOrCommit()

		txErc20 := buildCustomTx(normalErc20ContractAddress, "95d89b41")
		_, err := suite.CITS.DeliverEthTx(sender, txErc20)
		suite.Require().NoError(err)
		ethTxErc20 := txErc20.AsTransaction()

		txVfbc := buildCustomTx(vfbcContractAddress, "95d89b41")
		_, err = suite.CITS.DeliverEthTx(sender, txVfbc)
		suite.Require().NoError(err)
		ethTxVfbc := txVfbc.AsTransaction()

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		receiptErc20 := suite.GetTxReceipt(ethTxErc20.Hash())
		receiptVfbc := suite.GetTxReceipt(ethTxVfbc.Hash())

		suite.Require().Equal(ethtypes.ReceiptStatusSuccessful, receiptErc20.Status)
		suite.Require().Equal(ethtypes.ReceiptStatusSuccessful, receiptVfbc.Status)

		callTracerErc20 := callTracer(ethTxErc20.Hash())
		callTracerVfbc := callTracer(ethTxVfbc.Hash())

		suite.Equal(callTracerErc20.Type, callTracerVfbc.Type)
		suite.Equal(strings.ToLower(sender.GetEthAddress().String()), callTracerErc20.From)
		suite.Equal(strings.ToLower(sender.GetEthAddress().String()), callTracerVfbc.From)
		suite.Equal(strings.ToLower(normalErc20ContractAddress.String()), callTracerErc20.To)
		suite.Equal(strings.ToLower(vfbcContractAddress.String()), callTracerVfbc.To)
		suite.Equal(callTracerErc20.Value, callTracerVfbc.Value)
		suite.Equal(callTracerErc20.Gas, callTracerVfbc.Gas, "input gas must be the same")
		suite.Equal(fmt.Sprintf("0x%x", 3520), callTracerErc20.GasUsed)
		suite.Equal(fmt.Sprintf("0x%x", evmtypes.VFBCopgSymbol), callTracerVfbc.GasUsed)
		suite.Equal(callTracerErc20.Input, callTracerVfbc.Input)
		suite.Equal("0x000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000045744594d00000000000000000000000000000000000000000000000000000000", callTracerErc20.Output)
		suite.Equal("0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000344594d0000000000000000000000000000000000000000000000000000000000", callTracerVfbc.Output)
		suite.Empty(callTracerErc20.Error)
		suite.Empty(callTracerVfbc.Error)
		suite.Empty(callTracerErc20.Calls)
		suite.Empty(callTracerVfbc.Calls)
	})

	suite.Run("trace of success totalSupply()", func() {
		suite.CITS.WaitNextBlockOrCommit()

		txErc20 := buildCustomTx(normalErc20ContractAddress, "18160ddd")
		_, err := suite.CITS.DeliverEthTx(sender, txErc20)
		suite.Require().NoError(err)
		ethTxErc20 := txErc20.AsTransaction()

		txVfbc := buildCustomTx(vfbcContractAddress, "18160ddd")
		_, err = suite.CITS.DeliverEthTx(sender, txVfbc)
		suite.Require().NoError(err)
		ethTxVfbc := txVfbc.AsTransaction()

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		receiptErc20 := suite.GetTxReceipt(ethTxErc20.Hash())
		receiptVfbc := suite.GetTxReceipt(ethTxVfbc.Hash())

		suite.Require().Equal(ethtypes.ReceiptStatusSuccessful, receiptErc20.Status)
		suite.Require().Equal(ethtypes.ReceiptStatusSuccessful, receiptVfbc.Status)

		callTracerErc20 := callTracer(ethTxErc20.Hash())
		callTracerVfbc := callTracer(ethTxVfbc.Hash())

		suite.Equal(callTracerErc20.Type, callTracerVfbc.Type)
		suite.Equal(strings.ToLower(sender.GetEthAddress().String()), callTracerErc20.From)
		suite.Equal(strings.ToLower(sender.GetEthAddress().String()), callTracerVfbc.From)
		suite.Equal(strings.ToLower(normalErc20ContractAddress.String()), callTracerErc20.To)
		suite.Equal(strings.ToLower(vfbcContractAddress.String()), callTracerVfbc.To)
		suite.Equal(callTracerErc20.Value, callTracerVfbc.Value)
		suite.Equal(callTracerErc20.Gas, callTracerVfbc.Gas, "input gas must be the same")
		suite.Equal(fmt.Sprintf("0x%x", 2505), callTracerErc20.GasUsed)
		suite.Equal(fmt.Sprintf("0x%x", evmtypes.VFBCopgTotalSupply), callTracerVfbc.GasUsed)
		suite.Equal(callTracerErc20.Input, callTracerVfbc.Input)
		suite.NotEqual(common.Hash{}, common.HexToHash(callTracerErc20.Output))
		suite.NotEqual(common.Hash{}, common.HexToHash(callTracerVfbc.Output))
		suite.Empty(callTracerErc20.Error)
		suite.Empty(callTracerVfbc.Error)
		suite.Empty(callTracerErc20.Calls)
		suite.Empty(callTracerVfbc.Calls)
	})

	suite.Run("trace of success balanceOf(address)", func() {
		suite.CITS.WaitNextBlockOrCommit()

		txErc20 := buildCustomTx(normalErc20ContractAddress, "70a08231", "000000000000000000000000"+tokenOwner.GetEthAddress().String()[2:])
		_, err := suite.CITS.DeliverEthTx(sender, txErc20)
		suite.Require().NoError(err)
		ethTxErc20 := txErc20.AsTransaction()

		txVfbc := buildCustomTx(vfbcContractAddress, "70a08231", "000000000000000000000000"+senderOfVfbc.GetEthAddress().String()[2:])
		_, err = suite.CITS.DeliverEthTx(sender, txVfbc)
		suite.Require().NoError(err)
		ethTxVfbc := txVfbc.AsTransaction()

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		receiptErc20 := suite.GetTxReceipt(ethTxErc20.Hash())
		receiptVfbc := suite.GetTxReceipt(ethTxVfbc.Hash())

		suite.Require().Equal(ethtypes.ReceiptStatusSuccessful, receiptErc20.Status)
		suite.Require().Equal(ethtypes.ReceiptStatusSuccessful, receiptVfbc.Status)

		callTracerErc20 := callTracer(ethTxErc20.Hash())
		callTracerVfbc := callTracer(ethTxVfbc.Hash())

		suite.Equal(callTracerErc20.Type, callTracerVfbc.Type)
		suite.Equal(strings.ToLower(sender.GetEthAddress().String()), callTracerErc20.From)
		suite.Equal(strings.ToLower(sender.GetEthAddress().String()), callTracerVfbc.From)
		suite.Equal(strings.ToLower(normalErc20ContractAddress.String()), callTracerErc20.To)
		suite.Equal(strings.ToLower(vfbcContractAddress.String()), callTracerVfbc.To)
		suite.Equal(callTracerErc20.Value, callTracerVfbc.Value)
		suite.Equal(callTracerErc20.Gas, callTracerVfbc.Gas, "input gas must be the same")
		suite.Equal(fmt.Sprintf("0x%x", 2886), callTracerErc20.GasUsed)
		suite.Equal(fmt.Sprintf("0x%x", evmtypes.VFBCopgBalanceOf), callTracerVfbc.GasUsed)
		if suite.NotEqual(callTracerErc20.Input, callTracerVfbc.Input) {
			suite.Len(callTracerVfbc.Input[2:], 72)
		}
		suite.NotEqual(common.Hash{}, common.HexToHash(callTracerErc20.Output))
		suite.NotEqual(common.Hash{}, common.HexToHash(callTracerVfbc.Output))
		suite.Empty(callTracerErc20.Error)
		suite.Empty(callTracerVfbc.Error)
		suite.Empty(callTracerErc20.Calls)
		suite.Empty(callTracerVfbc.Calls)
	})
}

type callFrame struct {
	Type    string      `json:"type"`
	From    string      `json:"from"`
	To      string      `json:"to,omitempty"`
	Value   string      `json:"value,omitempty"`
	Gas     string      `json:"gas"`
	GasUsed string      `json:"gasUsed"`
	Input   string      `json:"input"`
	Output  string      `json:"output,omitempty"`
	Error   string      `json:"error,omitempty"`
	Calls   []callFrame `json:"calls,omitempty"`
}
