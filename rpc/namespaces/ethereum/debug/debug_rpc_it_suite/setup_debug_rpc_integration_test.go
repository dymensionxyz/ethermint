package demo

//goland:noinspection SpellCheckingInspection
import (
	"encoding/json"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/ethermint/integration_test_util"
	itutiltypes "github.com/evmos/ethermint/integration_test_util/types"
	"github.com/evmos/ethermint/rpc/namespaces/ethereum/debug"
	"github.com/stretchr/testify/suite"
	"testing"
)

type DebugRpcTestSuite struct {
	suite.Suite
	CITS *integration_test_util.ChainIntegrationTestSuite
}

func (suite *DebugRpcTestSuite) App() itutiltypes.ChainApp {
	return suite.CITS.ChainApp
}

func (suite *DebugRpcTestSuite) Ctx() sdk.Context {
	return suite.CITS.CurrentContext
}

func (suite *DebugRpcTestSuite) Commit() {
	suite.CITS.Commit()
}

func TestEthRpcTestSuite(t *testing.T) {
	suite.Run(t, new(DebugRpcTestSuite))
}

func (suite *DebugRpcTestSuite) SetupSuite() {
}

func (suite *DebugRpcTestSuite) SetupTest() {
	suite.CITS = integration_test_util.CreateChainIntegrationTestSuite(suite.T(), suite.Require())
	suite.CITS.EnsureTendermint() // RPC requires Tendermint
}

func (suite *DebugRpcTestSuite) TearDownTest() {
	suite.CITS.Cleanup()
}

func (suite *DebugRpcTestSuite) TearDownSuite() {
}

func (suite *DebugRpcTestSuite) GetDebugAPI() *debug.API {
	return suite.GetDebugAPIAt(0)
}

func (suite *DebugRpcTestSuite) GetDebugAPIAt(height int64) *debug.API {
	return debug.NewAPI(server.NewDefaultContext(), suite.CITS.RpcBackendAt(height))
}

func (suite *DebugRpcTestSuite) GetTxReceipt(txHash common.Hash) *ethtypes.Receipt {
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
