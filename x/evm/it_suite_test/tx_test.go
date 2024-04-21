package vfc_it_suite_test

import (
	sdkmath "cosmossdk.io/math"
	"encoding/json"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"math/big"
)

func (suite *EvmITSuite) TestTransfer() {
	sender := suite.CITS.WalletAccounts.Number(1)
	receiver := suite.CITS.WalletAccounts.Number(2)

	extremeBalance := new(big.Int).SetBytes(func() []byte {
		b := make([]byte, 32)
		b[0] = 0xFF
		return b
	}())
	if extremeBalance.Sign() < 1 {
		extremeBalance = new(big.Int).Abs(extremeBalance)
	}

	suite.CITS.MintCoin(sender, sdk.Coin{
		Denom:  suite.CITS.ChainConstantsConfig.GetMinDenom(),
		Amount: sdkmath.NewIntFromBigInt(extremeBalance),
	})

	balance := func(address common.Address) *big.Int {
		return suite.App().EvmKeeper().GetBalance(suite.Ctx(), address)
	}

	tests := []struct {
		name        string
		modifier    func(amount, gasPrice, gasFeeCap, gasTipCap *big.Int) (modifiedAmount, modifiedGasPrice, modifiedGasFeeCap, modifiedGasTipCap *big.Int)
		wantSuccess bool
	}{
		{
			name: "send small amount",
			modifier: func(amount, gasPrice, gasFeeCap, gasTipCap *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int) {
				amount = big.NewInt(10)
				return amount, gasPrice, gasFeeCap, gasTipCap
			},
			wantSuccess: true,
		},
		{
			name: "send all balance",
			modifier: func(amount, gasPrice, gasFeeCap, gasTipCap *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int) {
				amount = balance(sender.GetEthAddress())
				return amount, gasPrice, gasFeeCap, gasTipCap
			},
			wantSuccess: false,
		},
		{
			name: "send amount more than have",
			modifier: func(amount, gasPrice, gasFeeCap, gasTipCap *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int) {
				amount = new(big.Int).Add(common.Big1, balance(sender.GetEthAddress()))
				return amount, gasPrice, gasFeeCap, gasTipCap
			},
			wantSuccess: false,
		},
		{
			name: "send zero amount",
			modifier: func(amount, gasPrice, gasFeeCap, gasTipCap *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int) {
				amount = common.Big0
				return amount, gasPrice, gasFeeCap, gasTipCap
			},
			wantSuccess: true,
		},
		{
			name: "send negative amount",
			modifier: func(amount, gasPrice, gasFeeCap, gasTipCap *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int) {
				amount = big.NewInt(-1)
				return amount, gasPrice, gasFeeCap, gasTipCap
			},
			wantSuccess: false,
		},
		{
			name: "negative gas price",
			modifier: func(amount, gasPrice, gasFeeCap, gasTipCap *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int) {
				gasPrice = big.NewInt(-1)

				gasFeeCap = nil
				gasTipCap = nil
				return amount, gasPrice, gasFeeCap, gasTipCap
			},
			wantSuccess: false,
		},
		{
			name: "negative gas fee cap",
			modifier: func(amount, gasPrice, gasFeeCap, gasTipCap *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int) {
				gasFeeCap = new(big.Int).Neg(gasFeeCap)
				return amount, gasPrice, gasFeeCap, gasTipCap
			},
			wantSuccess: false,
		},
		{
			name: "negative gas tip cap",
			modifier: func(amount, gasPrice, gasFeeCap, gasTipCap *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int) {
				gasTipCap = new(big.Int).Neg(gasTipCap)
				return amount, gasPrice, gasFeeCap, gasTipCap
			},
			wantSuccess: false,
		},
		{
			name: "negative gas fee cap & gas tip cap",
			modifier: func(amount, gasPrice, gasFeeCap, gasTipCap *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int) {
				gasFeeCap = new(big.Int).Neg(gasFeeCap)
				gasTipCap = new(big.Int).Neg(gasTipCap)
				return amount, gasPrice, gasFeeCap, gasTipCap
			},
			wantSuccess: false,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			ctx := suite.Ctx()

			from := sender.GetEthAddress()
			to := receiver.GetEthAddress()

			senderBalanceBefore := balance(from)
			receiverBalanceBefore := balance(to)

			baseFee := suite.CITS.ChainApp.FeeMarketKeeper().GetBaseFee(suite.Ctx())

			amount, gasPrice, gasFeeCap, gasTipCap := tt.modifier(common.Big0, nil, baseFee, baseFee)

			newTx := func() *evmtypes.MsgEthereumTx {
				return evmtypes.NewTx(
					suite.CITS.ChainApp.EvmKeeper().ChainID(),
					suite.CITS.ChainApp.EvmKeeper().GetNonce(ctx, from),
					&to,
					amount,       // amount
					params.TxGas, // gas limit
					gasPrice,     // gas price
					gasFeeCap,    // gas fee cap
					gasTipCap,    // gas tip cap
					nil,          // data
					&ethtypes.AccessList{},
				)
			}

			msgEthereumTx := newTx()
			msgEthereumTx.From = from.String()

			resp, err := suite.CITS.DeliverEthTx(sender, msgEthereumTx)

			suite.Commit()

			senderBalanceAfter := balance(from)
			receiverBalanceAfter := balance(to)

			if tt.wantSuccess {
				suite.Require().NoError(err)
				suite.Require().NotNil(resp)

				receipt := suite.GetTxReceipt(common.HexToHash(resp.EthTxHash))
				suite.Require().NotNil(receipt)

				suite.Equal(ethtypes.ReceiptStatusSuccessful, receipt.Status, "tx must be successful")
				suite.Empty(resp.EvmError)

				if amount == nil {
					amount = common.Big0
				}

				effectivePrice := msgEthereumTx.GetEffectiveFee(baseFee)
				txFee := effectivePrice

				suite.Equalf(
					new(big.Int).Sub(
						senderBalanceBefore,
						new(big.Int).Add(
							amount,
							txFee,
						),
					),
					senderBalanceAfter,
					"sender balance must decrease by (%s + %s) from %s", amount, txFee, senderBalanceBefore,
				)
				suite.Equalf(
					new(big.Int).Add(
						receiverBalanceBefore,
						amount,
					),
					receiverBalanceAfter,
					"receiver balance must increase by %s from %s", amount, receiverBalanceBefore,
				)
			} else {
				suite.Require().Error(err)

				mapReceipt, err := suite.CITS.RpcBackend.GetTransactionReceipt(common.HexToHash(resp.EthTxHash))
				suite.Require().NoError(err)

				if mapReceipt != nil {
					bzMapReceipt, err := json.Marshal(mapReceipt)
					suite.Require().NoError(err)

					var receipt ethtypes.Receipt
					err = json.Unmarshal(bzMapReceipt, &receipt)
					suite.Require().NoError(err)

					suite.Equal(ethtypes.ReceiptStatusFailed, receipt.Status, "tx must be failed")

				}

				suite.Equal(senderBalanceBefore, senderBalanceAfter, "sender balance must not change")
				suite.Equal(receiverBalanceBefore, receiverBalanceAfter, "receiver balance must not change")
			}
		})
	}
}
