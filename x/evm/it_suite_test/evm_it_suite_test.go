package vfc_it_suite_test

//goland:noinspection SpellCheckingInspection
import (
	"encoding/json"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/ethermint/integration_test_util"
	itutiltypes "github.com/evmos/ethermint/integration_test_util/types"
	"github.com/stretchr/testify/suite"
	"testing"
)

//goland:noinspection GoSnakeCaseUsage,SpellCheckingInspection
type EvmITSuite struct {
	suite.Suite
	CITS *integration_test_util.ChainIntegrationTestSuite
}

func (suite *EvmITSuite) App() itutiltypes.ChainApp {
	return suite.CITS.ChainApp
}

func (suite *EvmITSuite) Ctx() sdk.Context {
	return suite.CITS.CurrentContext
}

func (suite *EvmITSuite) Commit() {
	suite.CITS.Commit()
}

func TestVfcITSuite(t *testing.T) {
	suite.Run(t, new(EvmITSuite))
}

func (suite *EvmITSuite) SetupSuite() {
}

func (suite *EvmITSuite) SetupTest() {
	suite.CITS = integration_test_util.CreateChainIntegrationTestSuite(suite.T(), suite.Require())
}

func (suite *EvmITSuite) TearDownTest() {
	suite.CITS.Cleanup()
}

func (suite *EvmITSuite) TearDownSuite() {
}

func (suite *EvmITSuite) GetTxReceipt(txHash common.Hash) *ethtypes.Receipt {
	mapReceipt, err := suite.CITS.RpcBackend.GetTransactionReceipt(txHash)
	suite.Require().NoError(err)
	suite.Require().NotNil(mapReceipt)

	bzMapReceipt, err := json.Marshal(mapReceipt)
	suite.Require().NoError(err)

	var receipt ethtypes.Receipt
	err = json.Unmarshal(bzMapReceipt, &receipt)
	suite.Require().NoError(err)

	return &receipt
}
