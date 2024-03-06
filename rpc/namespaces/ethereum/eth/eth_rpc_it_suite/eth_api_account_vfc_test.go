package demo

import (
	"github.com/evmos/ethermint/integration_test_util"
	rpctypes "github.com/evmos/ethermint/rpc/types"
	"github.com/evmos/ethermint/testutil"
)

func (suite *EthRpcTestSuite) Test_GetCode_VFC() {
	deployer := suite.CITS.WalletAccounts.Number(1)
	normalErc20ContractAddress, _, _, err := suite.CITS.TxDeploy2WDymContract(deployer, deployer)
	suite.Require().NoError(err)

	const ibcAtom = "ibc/uatom"
	metaOfValid1 := testutil.NewBankDenomMetadata(ibcAtom, 6)
	suite.App().BankKeeper().SetDenomMetaData(suite.Ctx(), metaOfValid1)

	suite.Commit() // trigger deploy contract for IBC Atom

	latest := rpctypes.EthLatestBlockNumber
	latestBlock := rpctypes.BlockNumberOrHash{
		BlockNumber: &latest,
		BlockHash:   nil,
	}
	ethPublicApi := suite.GetEthPublicAPI()

	testAccount := integration_test_util.NewTestAccount(suite.T(), nil)
	code, err := ethPublicApi.GetCode(testAccount.GetEthAddress(), latestBlock)
	suite.Require().NoError(err, "failed to get code")
	suite.Require().Empty(code, "code must be empty")

	code, err = ethPublicApi.GetCode(normalErc20ContractAddress, latestBlock)
	suite.Require().NoError(err, "failed to get code")
	suite.Require().NotEmpty(code, "code must not be empty")

	vfbcContractAddressOfIbcAtom, found := suite.App().EvmKeeper().GetVirtualFrontierBankContractAddressByDenom(suite.Ctx(), ibcAtom)
	suite.Require().True(found, "contract must exists")

	code, err = ethPublicApi.GetCode(vfbcContractAddressOfIbcAtom, latestBlock)
	suite.Require().NoError(err, "failed to get code")
	suite.Require().NotEmpty(code, "code must not be empty")

	codeOfVfbcContractAddressOfIbcAtom := code

	suite.Equal(
		suite.App().EvmKeeper().GetNonce(suite.Ctx(), normalErc20ContractAddress),
		suite.App().EvmKeeper().GetNonce(suite.Ctx(), vfbcContractAddressOfIbcAtom),
	)

	vfbcContractAddressOfNative, found := suite.App().EvmKeeper().GetVirtualFrontierBankContractAddressByDenom(suite.Ctx(), suite.CITS.ChainConstantsConfig.GetMinDenom())
	suite.Require().True(found, "contract must exists")

	suite.Equal(
		suite.App().EvmKeeper().GetNonce(suite.Ctx(), normalErc20ContractAddress),
		suite.App().EvmKeeper().GetNonce(suite.Ctx(), vfbcContractAddressOfNative),
	)

	code, err = ethPublicApi.GetCode(vfbcContractAddressOfNative, latestBlock)
	suite.Require().NoError(err, "failed to get code")
	suite.Require().NotEmpty(code, "code must not be empty")
	codeOfVfbcContractAddressOfNative := code

	suite.Equal(codeOfVfbcContractAddressOfIbcAtom, codeOfVfbcContractAddressOfNative, "deployed bytecode of the genesis deployed VFBC must be equals to consensus-deployed ones")
}
