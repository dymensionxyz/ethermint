package keeper_test

import (
	"github.com/cometbft/cometbft/abci/types"
	"github.com/evmos/ethermint/testutil"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"math"
)

func (suite *KeeperTestSuite) TestBeginBlock() {
	metaOfValid1 := testutil.NewBankDenomMetadata("ibc/uatom", 6)
	suite.app.BankKeeper.SetDenomMetaData(suite.ctx, metaOfValid1)

	metaOfInvalid := testutil.NewBankDenomMetadata("ibc/uosmo", 6)
	metaOfInvalid.Display = ""
	suite.app.BankKeeper.SetDenomMetaData(suite.ctx, metaOfInvalid)
	suite.Require().True(suite.app.BankKeeper.HasDenomMetaData(suite.ctx, metaOfInvalid.Base)) // ensure invalid metadata is set

	metaOfOverflowDecimals := testutil.NewBankDenomMetadata("ibc/uosmo", 0)
	metaOfOverflowDecimals.DenomUnits[1].Exponent = math.MaxUint8 + 1 // overflow uint8
	suite.app.BankKeeper.SetDenomMetaData(suite.ctx, metaOfOverflowDecimals)
	suite.Require().True(suite.app.BankKeeper.HasDenomMetaData(suite.ctx, metaOfOverflowDecimals.Base)) // ensure invalid metadata is set

	metaOfValid2 := testutil.NewBankDenomMetadata("ibc/udym", 6)
	suite.app.BankKeeper.SetDenomMetaData(suite.ctx, metaOfValid2)

	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfValid1.Base))
	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfInvalid.Base))
	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfOverflowDecimals.Base))
	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfValid2.Base))

	suite.Require().NotPanics(func() {
		suite.app.EvmKeeper.BeginBlock(suite.ctx, types.RequestBeginBlock{})
	})

	suite.True(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfValid1.Base), "virtual frontier bank contract for valid metadata should be created")
	suite.False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfInvalid.Base), "should skip virtual frontier bank contract creation for invalid metadata")
	suite.False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfOverflowDecimals.Base), "should skip virtual frontier bank contract creation for metadata which exponent overflow of uint8")
	suite.True(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfValid2.Base), "virtual frontier bank contract for valid metadata should be created")
}

func (suite *KeeperTestSuite) TestEndBlock() {
	em := suite.ctx.EventManager()
	suite.Require().Equal(0, len(em.Events()))

	res := suite.app.EvmKeeper.EndBlock(suite.ctx, types.RequestEndBlock{})
	suite.Require().Equal([]types.ValidatorUpdate{}, res)

	// should emit 1 EventTypeBlockBloom event on EndBlock
	suite.Require().Equal(1, len(em.Events()))
	suite.Require().Equal(evmtypes.EventTypeBlockBloom, em.Events()[0].Type)
}
