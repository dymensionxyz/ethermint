package keeper_test

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/ethermint/x/evm/types"
)

type virtualFrontierBankContract struct {
	Address     string
	Active      bool
	MinDenom    string
	Exponent    uint32
	DisplayName string
}

func (m virtualFrontierBankContract) convert(cdc codec.Codec) *types.VirtualFrontierContract {
	meta := types.VFBankContractMetadata{
		MinDenom:    m.MinDenom,
		Exponent:    m.Exponent,
		DisplayName: m.DisplayName,
	}

	bz, err := cdc.Marshal(&meta)
	if err != nil {
		panic(err)
	}

	return &types.VirtualFrontierContract{
		Address:  m.Address,
		Active:   m.Active,
		Type:     uint32(types.VirtualFrontierContractTypeBankContract),
		Metadata: bz,
	}
}

func (suite *KeeperTestSuite) TestGetSetVirtualFrontierContract() {
	contractAddress1 := common.HexToAddress("0x0000000000000000000000000000000000001001")
	contractAddress2 := common.HexToAddress("0x0000000000000000000000000000000000001002")
	contractAddress3 := common.HexToAddress("0x0000000000000000000000000000000000001003")

	var err error

	err = suite.app.EvmKeeper.SetVirtualFrontierContract(suite.ctx, contractAddress1, virtualFrontierBankContract{
		Address:     contractAddress1.String(),
		Active:      false,
		MinDenom:    "ibc/uatomAABBCC",
		Exponent:    6,
		DisplayName: "ATOM",
	}.convert(suite.appCodec))
	suite.Require().NoError(err)

	err = suite.app.EvmKeeper.SetVirtualFrontierContract(suite.ctx, contractAddress2, virtualFrontierBankContract{
		Address:     contractAddress2.String(),
		Active:      true,
		MinDenom:    "ibc/uosmoXXYYZZ",
		Exponent:    6,
		DisplayName: "OSMO",
	}.convert(suite.appCodec))
	suite.Require().NoError(err)

	contract1 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress1)
	suite.Require().NotNil(contract1)

	contract2 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress2)
	suite.Require().NotNil(contract2)

	contract3 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress3)
	suite.Require().Nil(contract3)

	suite.Equal(virtualFrontierBankContract{
		Address:     contractAddress1.String(),
		Active:      false,
		MinDenom:    "ibc/uatomAABBCC",
		Exponent:    6,
		DisplayName: "ATOM",
	}.convert(suite.appCodec), contract1)

	suite.Equal(virtualFrontierBankContract{
		Address:     contractAddress2.String(),
		Active:      true,
		MinDenom:    "ibc/uosmoXXYYZZ",
		Exponent:    6,
		DisplayName: "OSMO",
	}.convert(suite.appCodec), contract2)

	err = suite.app.EvmKeeper.SetVirtualFrontierContract(suite.ctx, contractAddress3, virtualFrontierBankContract{
		Address:     contractAddress3.String(),
		Active:      true,
		MinDenom:    "", // <= missing
		Exponent:    6,
		DisplayName: "OSMO",
	}.convert(suite.appCodec))
	suite.Require().Error(err, "should reject contracts those not pass basic validation")
	suite.Nil(suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress3))

	err = suite.app.EvmKeeper.SetVirtualFrontierContract(suite.ctx, contractAddress3, virtualFrontierBankContract{
		Address:     contractAddress2.String(), // miss-match
		Active:      true,
		MinDenom:    "ibc/uAABBCC",
		Exponent:    6,
		DisplayName: "AABBCCDD",
	}.convert(suite.appCodec))
	suite.Require().Error(err, "should reject contracts those miss-match address")
}

func (suite *KeeperTestSuite) TestDeployNewVirtualFrontierContract() {
	contractAddress1 := common.HexToAddress("0x0000000000000000000000000000000000002001")
	contractAddress2 := common.HexToAddress("0x0000000000000000000000000000000000002002")
	contractAddress3 := common.HexToAddress("0x0000000000000000000000000000000000002003")

	var err error

	err = suite.app.EvmKeeper.DeployNewVirtualFrontierContract(suite.ctx, contractAddress1, virtualFrontierBankContract{
		Address:     contractAddress1.String(),
		Active:      false,
		MinDenom:    "ibc/uatomAABBCC",
		Exponent:    6,
		DisplayName: "ATOM",
	}.convert(suite.appCodec))
	suite.Require().NoError(err)

	err = suite.app.EvmKeeper.DeployNewVirtualFrontierContract(suite.ctx, contractAddress2, virtualFrontierBankContract{
		Address:     contractAddress2.String(),
		Active:      true,
		MinDenom:    "ibc/uosmoXXYYZZ",
		Exponent:    6,
		DisplayName: "OSMO",
	}.convert(suite.appCodec))
	suite.Require().NoError(err)

	err = suite.app.EvmKeeper.DeployNewVirtualFrontierContract(suite.ctx, contractAddress2, virtualFrontierBankContract{
		Address:     contractAddress2.String(),
		Active:      true,
		MinDenom:    "ibc/uosmoXXYYZZ",
		Exponent:    6,
		DisplayName: "OSMO",
	}.convert(suite.appCodec))
	suite.Require().Error(err, "should not allow override contract")
	suite.Contains(err.Error(), "already registered in params")

	contract1 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress1)
	suite.Require().NotNil(contract1)

	contract2 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress2)
	suite.Require().NotNil(contract2)

	contract3 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress3)
	suite.Require().Nil(contract3)

	suite.Equal(virtualFrontierBankContract{
		Address:     contractAddress1.String(),
		Active:      false,
		MinDenom:    "ibc/uatomAABBCC",
		Exponent:    6,
		DisplayName: "ATOM",
	}.convert(suite.appCodec), contract1)

	suite.Equal(virtualFrontierBankContract{
		Address:     contractAddress2.String(),
		Active:      true,
		MinDenom:    "ibc/uosmoXXYYZZ",
		Exponent:    6,
		DisplayName: "OSMO",
	}.convert(suite.appCodec), contract2)

	err = suite.app.EvmKeeper.DeployNewVirtualFrontierContract(suite.ctx, contractAddress3, virtualFrontierBankContract{
		Address:     contractAddress3.String(),
		Active:      true,
		MinDenom:    "", // <= missing
		Exponent:    6,
		DisplayName: "OSMO",
	}.convert(suite.appCodec))
	suite.Require().Error(err, "should reject contracts those not pass basic validation")
	suite.Nil(suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress3))

	err = suite.app.EvmKeeper.DeployNewVirtualFrontierContract(suite.ctx, contractAddress3, virtualFrontierBankContract{
		Address:     contractAddress2.String(), // miss-match
		Active:      true,
		MinDenom:    "ibc/uAABBCC",
		Exponent:    6,
		DisplayName: "AABBCCDD",
	}.convert(suite.appCodec))
	suite.Require().Error(err, "should reject contracts those miss-match address")
	suite.Nil(suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress3))
}
