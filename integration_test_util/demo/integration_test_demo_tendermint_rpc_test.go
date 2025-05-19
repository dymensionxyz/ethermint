package demo

import (
	"context"
)

//goland:noinspection SpellCheckingInspection

func (suite *DemoTestSuite) Test_QC_TmRpc_Block() {
	suite.SkipIfDisabledCometBFT()

	backupContextHeight := suite.CITS.BaseApp().LastBlockHeight()
	suite.Commit()

	resultBlock, err := suite.CITS.QueryClients.CometBFTRpcHttpClient.Block(context.Background(), &backupContextHeight)
	suite.Require().NoError(err)
	suite.Require().NotNil(resultBlock)
	suite.Equal(backupContextHeight, resultBlock.Block.Height)
}
