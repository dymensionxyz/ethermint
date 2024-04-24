package ante_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/evmos/ethermint/app/ante"

	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

func (suite *AnteTestSuite) TestRejectMsgDecorator() {
	_, testAddresses, err := generatePrivKeyAddressPairs(5)
	suite.Require().NoError(err)

	decorator := ante.NewRejectMessagesDecorator(
		[]string{
			sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
			sdk.MsgTypeURL(&stakingtypes.MsgUndelegate{}),
		},
	)

	testMsgSend := createMsgSend(testAddresses)
	testMsgEthereumTx := &evmtypes.MsgEthereumTx{}

	testCases := []struct {
		name        string
		msgs        []sdk.Msg
		expectedErr error
	}{
		{
			"enabled msg - non blocked msg",
			[]sdk.Msg{
				testMsgSend,
			},
			nil,
		},
		{
			"blocked msg MsgEthereumTx",
			[]sdk.Msg{
				testMsgEthereumTx,
			},
			sdkerrors.ErrUnauthorized,
		},
		{
			"blocked msg",
			[]sdk.Msg{
				&stakingtypes.MsgUndelegate{},
			},
			sdkerrors.ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest()
			tx, err := suite.createTx(suite.priv, tc.msgs...)
			suite.Require().NoError(err)

			_, err = decorator.AnteHandle(suite.ctx, tx, false, NextFn)
			if tc.expectedErr != nil {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expectedErr)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
