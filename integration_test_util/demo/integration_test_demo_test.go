package demo

//goland:noinspection SpellCheckingInspection
import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/evmos/ethermint/integration_test_util"
	itutiltypes "github.com/evmos/ethermint/integration_test_util/types"
	"github.com/stretchr/testify/suite"
	"testing"
)

//goland:noinspection GoSnakeCaseUsage,SpellCheckingInspection
type DemoTestSuite struct {
	suite.Suite
	CITS   *integration_test_util.ChainIntegrationTestSuite
	IBCITS *integration_test_util.ChainsIbcIntegrationTestSuite
}

func (suite *DemoTestSuite) App() itutiltypes.ChainApp {
	return suite.CITS.ChainApp
}

func (suite *DemoTestSuite) Ctx() sdk.Context {
	return suite.CITS.CurrentContext
}

func (suite *DemoTestSuite) Commit() {
	suite.CITS.Commit()
}

func TestDemoTestSuite(t *testing.T) {
	suite.Run(t, new(DemoTestSuite))
}

func (suite *DemoTestSuite) SetupSuite() {
}

func (suite *DemoTestSuite) SetupTest() {
	suite.CITS = integration_test_util.CreateChainIntegrationTestSuite(suite.T(), suite.Require())
}

func (suite *DemoTestSuite) SetupIbcTest() {
	testChain2 := integration_test_util.IntegrationTestChain2

	// There is issue that IBC dual chains not work with Tendermint client so temporary disable it
	suite.CITS.Cleanup() // don't use Tendermint enabled chain
	testChain1 := integration_test_util.IntegrationTestChain1
	testChain1.DisableTendermint = true
	testChain2.DisableTendermint = true
	suite.CITS = integration_test_util.CreateChainIntegrationTestSuiteFromChainConfig(
		suite.T(), suite.Require(),
		testChain1,
	)
	// end of temporary disable Tendermint

	chain2 := integration_test_util.CreateChainIntegrationTestSuiteFromChainConfig(
		suite.T(), suite.Require(),
		testChain2,
	)

	suite.IBCITS = integration_test_util.CreateChainsIbcIntegrationTestSuite(suite.CITS, chain2, nil, nil)
}

func (suite *DemoTestSuite) TearDownTest() {
	if suite.IBCITS != nil {
		suite.IBCITS.Cleanup()
	} else {
		suite.CITS.Cleanup()
	}
}

func (suite *DemoTestSuite) TearDownSuite() {
}

func (suite *DemoTestSuite) SkipIfDisabledContractCreation() {
	if suite.CITS.TestConfig.DisabledContractCreation {
		suite.T().Skip("Contract creation is disabled, skip")
	}
}

func (suite *DemoTestSuite) SkipIfDisabledTendermint() {
	if !suite.CITS.HasTendermint() {
		suite.T().Skip("Tendermint is disabled, some methods can not be used, skip")
	}
}

func (suite *DemoTestSuite) TestEnsureStateResetEachTest1() {
	suite.testEnsureStateResetEachTest()
}

func (suite *DemoTestSuite) TestEnsureStateResetEachTest2() {
	suite.testEnsureStateResetEachTest()
}

func (suite *DemoTestSuite) testEnsureStateResetEachTest() {
	wallet1 := suite.CITS.WalletAccounts.Number(1)

	balanceBefore := suite.CITS.QueryBalance(0, wallet1.GetCosmosAddress().String())
	suite.Require().Equalf(
		suite.CITS.TestConfig.InitBalanceAmount, balanceBefore.Amount,
		"balance must be reset to default each test",
	)
	suite.True(balanceBefore.Amount.GT(sdk.ZeroInt()), "balance must be reset to default each test")

	// change balance
	err := suite.CITS.TxSend(wallet1, suite.CITS.WalletAccounts.Number(2), 0.1)
	suite.Commit()
	suite.Require().NoError(err)

	balanceAfter := suite.CITS.QueryBalance(0, wallet1.GetCosmosAddress().String())
	suite.Require().Truef(balanceAfter.Amount.LT(balanceBefore.Amount), "balance must be reduced to be evident for next test")
}
