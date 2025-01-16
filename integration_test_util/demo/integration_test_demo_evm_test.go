package demo

import (
	math "cosmossdk.io/math"
	rpctypes "github.com/evmos/ethermint/rpc/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

//goland:noinspection SpellCheckingInspection

func (suite *DemoTestSuite) Test_QC_Evm_Balance() {
	balance := suite.queryEvmBalance(0, suite.CITS.WalletAccounts.Number(1).GetEthAddress().String())
	suite.Require().True(balance.GT(math.ZeroInt()))
}

func (suite *DemoTestSuite) Test_QC_Evm_Balance_At_Different_Blocks() {
	sender := suite.CITS.WalletAccounts.Number(1)
	receiver := suite.CITS.WalletAccounts.Number(2)

	senderBalanceBefore := suite.queryEvmBalance(0, sender.GetEthAddress().String())
	receiverBalanceBefore := suite.queryEvmBalance(0, receiver.GetEthAddress().String())

	suite.Require().Truef(senderBalanceBefore.GT(math.ZeroInt()), "sender must have balance")

	contextHeightBeforeSend := suite.CITS.CurrentContext.BlockHeight()
	suite.Commit()

	_, err := suite.CITS.TxSendViaEVM(sender, receiver, 0.1)
	suite.Commit()
	suite.Require().NoError(err)

	senderBalanceAfter := suite.queryEvmBalance(0, sender.GetEthAddress().String())
	receiverBalanceAfter := suite.queryEvmBalance(0, receiver.GetEthAddress().String())

	suite.NotEqualf(senderBalanceBefore.String(), senderBalanceAfter.String(), "sender balance must be reduced")
	suite.Truef(senderBalanceAfter.LT(senderBalanceBefore), "sender balance must be reduced")

	suite.NotEqualf(receiverBalanceBefore.String(), receiverBalanceAfter.String(), "receiver balance must be increased")
	suite.Truef(receiverBalanceBefore.LT(receiverBalanceAfter), "receiver balance must be increased")

	// Historical block height
	historicalSenderBalance := suite.queryEvmBalance(contextHeightBeforeSend, sender.GetEthAddress().String())
	historicalReceiverBalanceAfter := suite.queryEvmBalance(contextHeightBeforeSend, receiver.GetEthAddress().String())
	suite.Equal(senderBalanceBefore.String(), historicalSenderBalance.String(), "mis-match sender balance at historical height")
	suite.Equal(receiverBalanceBefore.String(), historicalReceiverBalanceAfter.String(), "mis-match sender balance at historical height")
}

func (suite *DemoTestSuite) queryEvmBalance(height int64, evmAddress string) math.Int {
	res, err := suite.CITS.QueryClientsAt(height).EVM.Balance(
		rpctypes.ContextWithHeight(height),
		&evmtypes.QueryBalanceRequest{
			Address: evmAddress,
		},
	)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	if res.Balance == "0" {
		return math.ZeroInt()
	}
	bal, ok := math.NewIntFromString(res.Balance)
	suite.Require().True(ok)
	return bal
}
