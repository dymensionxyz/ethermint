package vfc_it_suite_test

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"math/big"
)

func (suite *EvmITSuite) TestTransfer() {
	sender := suite.CITS.WalletAccounts.Number(1)
	receiver := suite.CITS.WalletAccounts.Number(2)

	suite.CITS.MintCoin(sender, suite.CITS.NewBaseCoin(30))

	balance := func(address common.Address) *big.Int {
		return suite.App().EvmKeeper().GetBalance(suite.Ctx(), address)
	}

	tests := []struct {
		name        string
		amount      *big.Int
		wantSuccess bool
	}{
		{
			name:        "send small amount",
			amount:      big.NewInt(10),
			wantSuccess: true,
		},
		{
			name:        "send all",
			amount:      balance(sender.GetEthAddress()),
			wantSuccess: false,
		},
		{
			name:        "send more than have",
			amount:      new(big.Int).Add(common.Big1, balance(sender.GetEthAddress())),
			wantSuccess: false,
		},
		{
			name:        "send zero",
			amount:      common.Big0,
			wantSuccess: true,
		},
		{
			name:        "send negative",
			amount:      big.NewInt(-1),
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

			msgEthereumTx := evmtypes.NewTx(
				suite.CITS.ChainApp.EvmKeeper().ChainID(),
				suite.CITS.ChainApp.EvmKeeper().GetNonce(ctx, from),
				&to,
				tt.amount,    // amount
				params.TxGas, // gas limit
				nil,          // gas price
				baseFee,      // gas fee cap
				baseFee,      // gas tip cap
				nil,          // data
				&ethtypes.AccessList{},
			)
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

				amount := tt.amount
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

					suite.Equal(senderBalanceBefore, senderBalanceAfter, "sender balance must not change")
					suite.Equal(receiverBalanceBefore, receiverBalanceAfter, "receiver balance must not change")
				}
			}
		})
	}
}
