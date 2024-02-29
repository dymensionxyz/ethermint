package keeper_test

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/evmos/ethermint/testutil"
	ethermint "github.com/evmos/ethermint/types"
	"github.com/evmos/ethermint/x/evm/keeper"
	"github.com/evmos/ethermint/x/evm/types"
	"strings"
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
		MinDenom: m.MinDenom,
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

func (suite *KeeperTestSuite) TestGetSetIsVirtualFrontierContract() {
	deployerModuleAccount := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, types.ModuleVirtualFrontierContractDeployerName)
	suite.Require().NotNil(deployerModuleAccount)

	contractAddress1 := crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, deployerModuleAccount.GetSequence()+0)
	contractAddress2 := crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, deployerModuleAccount.GetSequence()+1)
	contractAddress3 := crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, deployerModuleAccount.GetSequence()+2)

	var err error

	err = suite.app.EvmKeeper.SetVirtualFrontierContract(suite.ctx, contractAddress1, virtualFrontierBankContract{
		Address:     strings.ToLower(contractAddress1.String()),
		Active:      false,
		MinDenom:    "ibc/uatomAABBCC",
		Exponent:    6,
		DisplayName: "ATOM",
	}.convert(suite.appCodec))
	suite.Require().NoError(err)

	err = suite.app.EvmKeeper.SetVirtualFrontierContract(suite.ctx, contractAddress2, virtualFrontierBankContract{
		Address:     strings.ToLower(contractAddress2.String()),
		Active:      true,
		MinDenom:    "ibc/uosmoXXYYZZ",
		Exponent:    6,
		DisplayName: "OSMO",
	}.convert(suite.appCodec))
	suite.Require().NoError(err)

	suite.True(suite.app.EvmKeeper.IsVirtualFrontierContract(suite.ctx, contractAddress1))
	contract1 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress1)
	suite.Require().NotNil(contract1)

	suite.True(suite.app.EvmKeeper.IsVirtualFrontierContract(suite.ctx, contractAddress2))
	contract2 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress2)
	suite.Require().NotNil(contract2)

	suite.False(suite.app.EvmKeeper.IsVirtualFrontierContract(suite.ctx, contractAddress3))
	contract3 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress3)
	suite.Require().Nil(contract3)

	suite.Equal(virtualFrontierBankContract{
		Address:     strings.ToLower(contractAddress1.String()),
		Active:      false,
		MinDenom:    "ibc/uatomAABBCC",
		Exponent:    6,
		DisplayName: "ATOM",
	}.convert(suite.appCodec), contract1)

	suite.Equal(virtualFrontierBankContract{
		Address:     strings.ToLower(contractAddress2.String()),
		Active:      true,
		MinDenom:    "ibc/uosmoXXYYZZ",
		Exponent:    6,
		DisplayName: "OSMO",
	}.convert(suite.appCodec), contract2)

	err = suite.app.EvmKeeper.SetVirtualFrontierContract(suite.ctx, contractAddress3, virtualFrontierBankContract{
		Address:     strings.ToLower(contractAddress3.String()),
		Active:      true,
		MinDenom:    "", // <= missing
		Exponent:    6,
		DisplayName: "OSMO",
	}.convert(suite.appCodec))
	suite.Require().Error(err, "should reject contracts those not pass basic validation")
	suite.Nil(suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress3))
	suite.False(suite.app.EvmKeeper.IsVirtualFrontierContract(suite.ctx, contractAddress3))

	err = suite.app.EvmKeeper.SetVirtualFrontierContract(suite.ctx, contractAddress3, virtualFrontierBankContract{
		Address:     strings.ToLower(contractAddress2.String()), // miss-match
		Active:      true,
		MinDenom:    "ibc/uAABBCC",
		Exponent:    6,
		DisplayName: "AABBCCDD",
	}.convert(suite.appCodec))
	suite.Require().Error(err, "should reject contracts those miss-match address")
	suite.False(suite.app.EvmKeeper.IsVirtualFrontierContract(suite.ctx, contractAddress3))
}

func (suite *KeeperTestSuite) TestGetSetMappingVirtualFrontierBankContractAddressByDenom() {
	const denom1 = "uosmo"
	keccak1 := crypto.Keccak256Hash([]byte(denom1))
	contractAddress1 := crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, 1)

	const denom2 = "ibc/ABCDEFG"
	keccak2 := crypto.Keccak256Hash([]byte(denom2))
	contractAddress2 := crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, 2)

	suite.Equal(append(types.KeyPrefixVirtualFrontierBankContractAddressByDenom, keccak1.Bytes()...), types.VirtualFrontierBankContractAddressByDenomKey(denom1))
	suite.Equal(append(types.KeyPrefixVirtualFrontierBankContractAddressByDenom, keccak2.Bytes()...), types.VirtualFrontierBankContractAddressByDenomKey(denom2))

	err := suite.app.EvmKeeper.SetMappingVirtualFrontierBankContractAddressByDenom(suite.ctx, denom1, contractAddress1)
	suite.Require().NoError(err)
	addr, found := suite.app.EvmKeeper.GetVirtualFrontierBankContractAddressByDenom(suite.ctx, denom1)
	suite.Require().True(found)
	suite.Equal(contractAddress1, addr)

	err = suite.app.EvmKeeper.SetMappingVirtualFrontierBankContractAddressByDenom(suite.ctx, denom2, contractAddress2)
	suite.Require().NoError(err)
	addr, found = suite.app.EvmKeeper.GetVirtualFrontierBankContractAddressByDenom(suite.ctx, denom2)
	suite.Require().True(found)
	suite.Equal(contractAddress2, addr)
}

func (suite *KeeperTestSuite) TestDeployNewVirtualFrontierBankContract() {
	deployerModuleAccount := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, types.ModuleVirtualFrontierContractDeployerName)
	suite.Require().NotNil(deployerModuleAccount)

	contractAddress1 := crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, deployerModuleAccount.GetSequence()+0)
	contractAddress2 := crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, deployerModuleAccount.GetSequence()+1)
	contractAddress3 := crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, deployerModuleAccount.GetSequence()+2)

	meta1 := testutil.NewBankDenomMetadata("ibc/uatomAABBCC", 6)
	meta2 := testutil.NewBankDenomMetadata("ibc/uosmoXXYYZZ", 6)

	suite.app.BankKeeper.SetDenomMetaData(suite.ctx, meta1)
	suite.app.BankKeeper.SetDenomMetaData(suite.ctx, meta2)

	vfbcMeta1, _ := types.CollectMetadataForVirtualFrontierBankContract(meta1)
	vfbcMeta2, _ := types.CollectMetadataForVirtualFrontierBankContract(meta2)

	bytecode, err := keeper.PrepareBytecodeForVirtualFrontierBankContractDeployment("TEST", 1)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(bytecode)

	addr, err := suite.app.EvmKeeper.DeployNewVirtualFrontierBankContract(suite.ctx, &types.VirtualFrontierContract{
		Active: false,
	}, &types.VFBankContractMetadata{
		MinDenom: "ibc/uatomAABBCC",
	}, &vfbcMeta1)
	suite.Require().NoError(err)
	suite.Equal(contractAddress1, addr)

	contractAccount1 := suite.app.EvmKeeper.GetAccount(suite.ctx, addr)
	suite.Require().NotNil(contractAccount1, "contract account should be created")
	suite.NotEmpty(contractAccount1.CodeHash, "contract account should have code hash")
	suite.NotEmpty(suite.app.EvmKeeper.GetCode(suite.ctx, common.BytesToHash(contractAccount1.CodeHash)), "contract account should have code")
	suite.Equal(uint64(1), contractAccount1.Nonce, "contract account nonce should be set to 1 as per EVM behavior")
	_, isEthAccount := suite.app.AccountKeeper.GetAccount(suite.ctx, addr.Bytes()).(*ethermint.EthAccount)
	suite.True(isEthAccount, "contract account should be an EthAccount")

	addr, err = suite.app.EvmKeeper.DeployNewVirtualFrontierBankContract(suite.ctx, &types.VirtualFrontierContract{
		Active: true,
	}, &types.VFBankContractMetadata{
		MinDenom: "ibc/uosmoXXYYZZ",
	}, &vfbcMeta2)
	suite.Require().NoError(err)
	suite.Equal(contractAddress2, addr)

	contractAccount2 := suite.app.EvmKeeper.GetAccount(suite.ctx, addr)
	suite.Require().NotNil(contractAccount2, "contract account should be created")
	suite.NotEmpty(contractAccount2.CodeHash, "contract account should have code hash")
	suite.NotEmpty(suite.app.EvmKeeper.GetCode(suite.ctx, common.BytesToHash(contractAccount2.CodeHash)), "contract account should have code")
	suite.Equal(uint64(1), contractAccount2.Nonce, "contract account nonce should be set to 1 as per EVM behavior")
	_, isEthAccount = suite.app.AccountKeeper.GetAccount(suite.ctx, addr.Bytes()).(*ethermint.EthAccount)
	suite.True(isEthAccount, "contract account should be an EthAccount")

	contract1 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress1)
	suite.Require().NotNil(contract1)

	contract2 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress2)
	suite.Require().NotNil(contract2)

	contract3 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress3)
	suite.Require().Nil(contract3)

	suite.Equal(virtualFrontierBankContract{
		Address:     strings.ToLower(contractAddress1.String()),
		Active:      false,
		MinDenom:    "ibc/uatomAABBCC",
		Exponent:    6,
		DisplayName: "ATOM",
	}.convert(suite.appCodec), contract1)

	suite.Equal(virtualFrontierBankContract{
		Address:     strings.ToLower(contractAddress2.String()),
		Active:      true,
		MinDenom:    "ibc/uosmoXXYYZZ",
		Exponent:    6,
		DisplayName: "OSMO",
	}.convert(suite.appCodec), contract2)

	addr, err = suite.app.EvmKeeper.DeployNewVirtualFrontierContract(suite.ctx, virtualFrontierBankContract{
		Active:      true,
		MinDenom:    "", // <= missing
		Exponent:    6,
		DisplayName: "OSMO",
	}.convert(suite.appCodec), bytecode)
	suite.Require().Error(err, "should reject contracts those not pass basic validation")
	suite.Equal(common.Address{}, addr, "when error, address should be empty")
	suite.Nil(suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress3))

	suite.Run("create when disabled contract creation", func() {
		suite.Commit()
		currentParams := suite.app.EvmKeeper.GetParams(suite.ctx)
		currentParams.EnableCreate = false
		suite.app.EvmKeeper.SetParams(suite.ctx, currentParams)
		suite.Commit()

		suite.Require().False(suite.app.EvmKeeper.GetParams(suite.ctx).EnableCreate, "contract creation should be disabled at this point")

		meta3 := testutil.NewBankDenomMetadata("ibc/aphotonMMNNOO", 18)
		suite.app.BankKeeper.SetDenomMetaData(suite.ctx, meta3)
		vfbcMeta3, _ := types.CollectMetadataForVirtualFrontierBankContract(meta3)

		addr, err = suite.app.EvmKeeper.DeployNewVirtualFrontierBankContract(suite.ctx, &types.VirtualFrontierContract{
			Active: true,
		}, &types.VFBankContractMetadata{
			MinDenom: "ibc/aphotonMMNNOO",
		}, &vfbcMeta3)
		suite.Require().NoError(err)
		suite.NotEqual(common.Address{}, addr)
		suite.NotNil(suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, addr), "contract should be created")
		contractAccount := suite.app.EvmKeeper.GetAccount(suite.ctx, addr)
		suite.NotNil(contractAccount, "contract account should be created")
		suite.NotEmpty(contractAccount.CodeHash, "contract account should have code hash")
		suite.NotEmpty(suite.app.EvmKeeper.GetCode(suite.ctx, common.BytesToHash(contractAccount.CodeHash)), "contract account should have code")
	})
}

func (suite *KeeperTestSuite) TestDeployNewVirtualFrontierContract() {
	deployerModuleAccount := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, types.ModuleVirtualFrontierContractDeployerName)
	suite.Require().NotNil(deployerModuleAccount)

	contractAddress1 := crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, deployerModuleAccount.GetSequence()+0)
	contractAddress2 := crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, deployerModuleAccount.GetSequence()+1)
	contractAddress3 := crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, deployerModuleAccount.GetSequence()+2)

	bytecode, err := keeper.PrepareBytecodeForVirtualFrontierBankContractDeployment("TEST", 1)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(bytecode)

	addr, err := suite.app.EvmKeeper.DeployNewVirtualFrontierContract(suite.ctx, virtualFrontierBankContract{
		Active:      false,
		MinDenom:    "ibc/uatomAABBCC",
		Exponent:    6,
		DisplayName: "ATOM",
	}.convert(suite.appCodec), bytecode)
	suite.Require().NoError(err)
	suite.Equal(contractAddress1, addr)

	contractAccount1 := suite.app.EvmKeeper.GetAccount(suite.ctx, addr)
	suite.Require().NotNil(contractAccount1, "contract account should be created")
	suite.NotEmpty(contractAccount1.CodeHash, "contract account should have code hash")
	suite.NotEmpty(suite.app.EvmKeeper.GetCode(suite.ctx, common.BytesToHash(contractAccount1.CodeHash)), "contract account should have code")
	suite.Equal(uint64(1), contractAccount1.Nonce, "contract account nonce should be set to 1 as per EVM behavior")
	_, isEthAccount := suite.app.AccountKeeper.GetAccount(suite.ctx, addr.Bytes()).(*ethermint.EthAccount)
	suite.True(isEthAccount, "contract account should be an EthAccount")

	addr, err = suite.app.EvmKeeper.DeployNewVirtualFrontierContract(suite.ctx, virtualFrontierBankContract{
		Active:      true,
		MinDenom:    "ibc/uosmoXXYYZZ",
		Exponent:    6,
		DisplayName: "OSMO",
	}.convert(suite.appCodec), bytecode)
	suite.Require().NoError(err)
	suite.Equal(contractAddress2, addr)

	contractAccount2 := suite.app.EvmKeeper.GetAccount(suite.ctx, addr)
	suite.Require().NotNil(contractAccount2, "contract account should be created")
	suite.NotEmpty(contractAccount2.CodeHash, "contract account should have code hash")
	suite.NotEmpty(suite.app.EvmKeeper.GetCode(suite.ctx, common.BytesToHash(contractAccount2.CodeHash)), "contract account should have code")
	suite.Equal(uint64(1), contractAccount2.Nonce, "contract account nonce should be set to 1 as per EVM behavior")
	_, isEthAccount = suite.app.AccountKeeper.GetAccount(suite.ctx, addr.Bytes()).(*ethermint.EthAccount)
	suite.True(isEthAccount, "contract account should be an EthAccount")

	contract1 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress1)
	suite.Require().NotNil(contract1)

	contract2 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress2)
	suite.Require().NotNil(contract2)

	contract3 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress3)
	suite.Require().Nil(contract3)

	suite.Equal(virtualFrontierBankContract{
		Address:     strings.ToLower(contractAddress1.String()),
		Active:      false,
		MinDenom:    "ibc/uatomAABBCC",
		Exponent:    6,
		DisplayName: "ATOM",
	}.convert(suite.appCodec), contract1)

	suite.Equal(virtualFrontierBankContract{
		Address:     strings.ToLower(contractAddress2.String()),
		Active:      true,
		MinDenom:    "ibc/uosmoXXYYZZ",
		Exponent:    6,
		DisplayName: "OSMO",
	}.convert(suite.appCodec), contract2)

	addr, err = suite.app.EvmKeeper.DeployNewVirtualFrontierContract(suite.ctx, virtualFrontierBankContract{
		Active:      true,
		MinDenom:    "", // <= missing
		Exponent:    6,
		DisplayName: "OSMO",
	}.convert(suite.appCodec), bytecode)
	suite.Require().Error(err, "should reject contracts those not pass basic validation")
	suite.Equal(common.Address{}, addr, "when error, address should be empty")
	suite.Nil(suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress3))

	suite.Run("create when disabled contract creation", func() {
		suite.Commit()
		currentParams := suite.app.EvmKeeper.GetParams(suite.ctx)
		currentParams.EnableCreate = false
		suite.app.EvmKeeper.SetParams(suite.ctx, currentParams)
		suite.Commit()

		suite.Require().False(suite.app.EvmKeeper.GetParams(suite.ctx).EnableCreate, "contract creation should be disabled at this point")

		addr, err = suite.app.EvmKeeper.DeployNewVirtualFrontierContract(suite.ctx, virtualFrontierBankContract{
			Active:      true,
			MinDenom:    "ibc/aphotonMMNNOO",
			Exponent:    18,
			DisplayName: "PHOTON",
		}.convert(suite.appCodec), bytecode)
		suite.Require().NoError(err)
		suite.NotEqual(common.Address{}, addr)
		suite.NotNil(suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, addr), "contract should be created")
		contractAccount := suite.app.EvmKeeper.GetAccount(suite.ctx, addr)
		suite.NotNil(contractAccount, "contract account should be created")
		suite.NotEmpty(contractAccount.CodeHash, "contract account should have code hash")
		suite.NotEmpty(suite.app.EvmKeeper.GetCode(suite.ctx, common.BytesToHash(contractAccount.CodeHash)), "contract account should have code")
	})
}
