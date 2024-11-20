package keeper_test

import (
	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/evmos/ethermint/testutil"
	ethermint "github.com/evmos/ethermint/types"
	"github.com/evmos/ethermint/utils"
	"github.com/evmos/ethermint/x/evm/keeper"
	"github.com/evmos/ethermint/x/evm/types"
	"math"
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
		Type:     types.VFC_TYPE_BANK,
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

func (suite *KeeperTestSuite) TestGetSetHasMappingVirtualFrontierBankContractAddressByDenom() {
	const denom1 = "uosmo"
	keccak1 := crypto.Keccak256Hash([]byte(denom1))
	contractAddress1 := crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, 1)

	const denom2 = "ibc/ABCDEFG"
	keccak2 := crypto.Keccak256Hash([]byte(denom2))
	contractAddress2 := crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, 2)

	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, denom1))
	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, denom2))

	suite.Equal(append(types.KeyPrefixVirtualFrontierBankContractAddressByDenom, keccak1.Bytes()...), types.VirtualFrontierBankContractAddressByDenomKey(denom1))
	suite.Equal(append(types.KeyPrefixVirtualFrontierBankContractAddressByDenom, keccak2.Bytes()...), types.VirtualFrontierBankContractAddressByDenomKey(denom2))

	err := suite.app.EvmKeeper.SetMappingVirtualFrontierBankContractAddressByDenom(suite.ctx, denom1, contractAddress1)
	suite.Require().NoError(err)
	suite.True(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, denom1))
	addr, found := suite.app.EvmKeeper.GetVirtualFrontierBankContractAddressByDenom(suite.ctx, denom1)
	suite.Require().True(found)
	suite.Equal(contractAddress1, addr)

	err = suite.app.EvmKeeper.SetMappingVirtualFrontierBankContractAddressByDenom(suite.ctx, denom2, contractAddress2)
	suite.Require().NoError(err)
	suite.True(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, denom2))
	addr, found = suite.app.EvmKeeper.GetVirtualFrontierBankContractAddressByDenom(suite.ctx, denom2)
	suite.Require().True(found)
	suite.Equal(contractAddress2, addr)
}

func (suite *KeeperTestSuite) TestDeployVirtualFrontierBankContractForAllBankDenomMetadataRecords() {
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

	metaOfValidButNotIbc := testutil.NewBankDenomMetadata("gamm/pool-1", 18)
	suite.app.BankKeeper.SetDenomMetaData(suite.ctx, metaOfValidButNotIbc)

	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfValid1.Base))
	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfInvalid.Base))
	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfOverflowDecimals.Base))
	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfValid2.Base))

	err := suite.app.EvmKeeper.DeployVirtualFrontierBankContractForAllBankDenomMetadataRecords(suite.ctx, func(metadata banktypes.Metadata) bool {
		return strings.HasPrefix(metadata.Base, "ibc/")
	})
	suite.Require().NoError(err)

	suite.True(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfValid1.Base), "virtual frontier bank contract for valid metadata should be created")
	suite.False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfInvalid.Base), "should skip virtual frontier bank contract creation for invalid metadata")
	suite.False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfOverflowDecimals.Base), "should skip virtual frontier bank contract creation for metadata which exponent overflow of uint8")
	suite.True(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfValid2.Base), "virtual frontier bank contract for valid metadata should be created")
	suite.False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfValidButNotIbc.Base), "should skip non-IBC tokens")
}

func (suite *KeeperTestSuite) TestDeployVirtualFrontierBankContractForBankDenomMetadataRecord() {
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

	metaOfValidButNotIbc := testutil.NewBankDenomMetadata("gamm/pool-1", 18)
	suite.app.BankKeeper.SetDenomMetaData(suite.ctx, metaOfValidButNotIbc)

	metaOfValid3ButNotBankMetadata := testutil.NewBankDenomMetadata("ibc/usk", 6)

	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfValid1.Base))
	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfInvalid.Base))
	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfOverflowDecimals.Base))
	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfValid2.Base))
	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfValidButNotIbc.Base))
	suite.Require().False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, metaOfValid3ButNotBankMetadata.Base))

	tests := []struct {
		name                string
		base                string
		preRun              func()
		wantDeployedSuccess bool
		wantFound           []string
		wantNotFound        []string
	}{
		{
			name:                "success",
			base:                metaOfValid1.Base,
			wantDeployedSuccess: true,
			wantFound:           []string{metaOfValid1.Base},
			wantNotFound:        []string{metaOfValid2.Base, metaOfInvalid.Base, metaOfOverflowDecimals.Base, metaOfValidButNotIbc.Base, metaOfValid3ButNotBankMetadata.Base},
		},
		{
			name:                "success",
			base:                metaOfValid2.Base,
			wantDeployedSuccess: true,
			wantFound:           []string{metaOfValid1.Base, metaOfValid2.Base},
			wantNotFound:        []string{metaOfInvalid.Base, metaOfOverflowDecimals.Base, metaOfValidButNotIbc.Base, metaOfValid3ButNotBankMetadata.Base},
		},
		{
			name:                "do not deploy invalid metadata",
			base:                metaOfInvalid.Base,
			wantDeployedSuccess: false,
			wantFound:           []string{metaOfValid1.Base, metaOfValid2.Base},
			wantNotFound:        []string{metaOfInvalid.Base, metaOfOverflowDecimals.Base, metaOfValidButNotIbc.Base, metaOfValid3ButNotBankMetadata.Base},
		},
		{
			name:                "do not deploy for metadata with exponent > uint8",
			base:                metaOfOverflowDecimals.Base,
			wantDeployedSuccess: false,
			wantFound:           []string{metaOfValid1.Base, metaOfValid2.Base},
			wantNotFound:        []string{metaOfInvalid.Base, metaOfOverflowDecimals.Base, metaOfValidButNotIbc.Base, metaOfValid3ButNotBankMetadata.Base},
		},
		{
			name:                "any base passed, will be deployed as long as it valid",
			base:                metaOfValidButNotIbc.Base,
			wantDeployedSuccess: true,
			wantFound:           []string{metaOfValid1.Base, metaOfValid2.Base, metaOfValidButNotIbc.Base},
			wantNotFound:        []string{metaOfInvalid.Base, metaOfOverflowDecimals.Base, metaOfValid3ButNotBankMetadata.Base},
		},
		{
			name:                "ignore metadata not found",
			base:                metaOfValid3ButNotBankMetadata.Base,
			wantDeployedSuccess: false,
			wantFound:           []string{metaOfValid1.Base, metaOfValid2.Base, metaOfValidButNotIbc.Base},
			wantNotFound:        []string{metaOfInvalid.Base, metaOfOverflowDecimals.Base, metaOfValid3ButNotBankMetadata.Base},
		},
		{
			name: "re-deploy for added missing metadata",
			base: metaOfValid3ButNotBankMetadata.Base,
			preRun: func() {
				suite.app.BankKeeper.SetDenomMetaData(suite.ctx, metaOfValid3ButNotBankMetadata)
			},
			wantDeployedSuccess: true,
			wantFound:           []string{metaOfValid1.Base, metaOfValid2.Base, metaOfValidButNotIbc.Base, metaOfValid3ButNotBankMetadata.Base},
			wantNotFound:        []string{metaOfInvalid.Base, metaOfOverflowDecimals.Base},
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			if tt.preRun != nil {
				tt.preRun()
			}

			err := suite.app.EvmKeeper.DeployVirtualFrontierBankContractForBankDenomMetadataRecord(suite.ctx, tt.base)
			if tt.wantDeployedSuccess {
				suite.Require().NoError(err)
				suite.True(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, tt.base), "want deployment for %s success", tt.base)
			} else {
				suite.Require().Error(err)
				suite.False(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, tt.base), "want deployment for %s failed", tt.base)
			}
			for base, found := range tt.wantFound {
				suite.Truef(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, found), "want smart contract for %s exists", base)
			}
			for base, notFound := range tt.wantNotFound {
				suite.Falsef(suite.app.EvmKeeper.HasVirtualFrontierBankContractByDenom(suite.ctx, notFound), "do not want smart contract for %s exists", base)
			}
		})
	}
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

	suite.Run("do not accept denom that exponent overflow of uint8", func() {
		meta3 := testutil.NewBankDenomMetadata("ibc/prohibited", 1)
		meta3.DenomUnits[1].Exponent = math.MaxUint8 + 1

		suite.app.BankKeeper.SetDenomMetaData(suite.ctx, meta3)

		vfbcMeta3, _ := types.CollectMetadataForVirtualFrontierBankContract(meta3)
		_, err = suite.app.EvmKeeper.DeployNewVirtualFrontierBankContract(suite.ctx, &types.VirtualFrontierContract{
			Active: true,
		}, &types.VFBankContractMetadata{
			MinDenom: vfbcMeta3.MinDenom,
		}, &vfbcMeta3)

		if suite.NotNil(err) {
			suite.Contains(err.Error(), "decimals does not fit uint8")
		}
	})
}

func (suite *KeeperTestSuite) TestDeployedVirtualFrontierBankContracts() {
	meta1 := testutil.NewBankDenomMetadata("ibc/uatomAABBCC", 6)
	meta2 := testutil.NewBankDenomMetadata("ibc/uosmoXXYYZZ", 6)

	suite.app.BankKeeper.SetDenomMetaData(suite.ctx, meta1)
	suite.app.BankKeeper.SetDenomMetaData(suite.ctx, meta2)

	vfbcMeta1, _ := types.CollectMetadataForVirtualFrontierBankContract(meta1)
	vfbcMeta2, _ := types.CollectMetadataForVirtualFrontierBankContract(meta2)

	addrAtom, err := suite.app.EvmKeeper.DeployNewVirtualFrontierBankContract(suite.ctx, &types.VirtualFrontierContract{
		Active: false,
	}, &types.VFBankContractMetadata{
		MinDenom: meta1.Base,
	}, &vfbcMeta1)
	suite.Require().NoError(err)

	addrOsmo, err := suite.app.EvmKeeper.DeployNewVirtualFrontierBankContract(suite.ctx, &types.VirtualFrontierContract{
		Active: true,
	}, &types.VFBankContractMetadata{
		MinDenom: meta2.Base,
	}, &vfbcMeta2)
	suite.Require().NoError(err)

	suite.Run("deployment code must equals to the hard-coded one", func() {
		for _, contractAddress := range []common.Address{addrAtom, addrOsmo} {
			suite.Equal(types.VFBCCodeHash, suite.app.EvmKeeper.GetAccount(suite.ctx, contractAddress).CodeHash)
		}
	})

	suite.Run("deployed bytecode must correctly mapped", func() {
		suite.Equal(types.VFBCCode, suite.app.EvmKeeper.GetCode(suite.ctx, common.BytesToHash(types.VFBCCodeHash)))
	})

	suite.Run("code hash for VFC account but proto is not EthAccount", func() {
		acc := suite.app.AccountKeeper.GetAccount(suite.ctx, addrAtom.Bytes())

		if _, isEthAccount := acc.(*ethermint.EthAccount); isEthAccount {
			// change account type
			baseAcc := authtypes.BaseAccount{}
			baseAcc.SetAddress(acc.GetAddress())
			baseAcc.SetPubKey(acc.GetPubKey())
			baseAcc.SetAccountNumber(acc.GetAccountNumber())
			baseAcc.SetSequence(acc.GetSequence())
			suite.app.AccountKeeper.SetAccount(suite.ctx, &baseAcc)

			// ensure account overridden
			acc2 := suite.app.AccountKeeper.GetAccount(suite.ctx, addrAtom.Bytes())
			_, isBaseAccount := acc2.(*authtypes.BaseAccount)
			suite.Require().True(isBaseAccount, "account must be overridden to a BaseAccount")
		}

		suite.Equal(
			types.VFBCCodeHash,
			suite.app.EvmKeeper.GetAccount(suite.ctx, addrAtom).CodeHash,
			"code hash must be mapped correctly",
		)
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
		suite.False(suite.app.EvmKeeper.GetParams(suite.ctx).EnableCreate, "contract creation should still be disabled at this point")

		suite.Commit()
		suite.False(suite.app.EvmKeeper.GetParams(suite.ctx).EnableCreate, "contract creation should still be disabled at this point")
	})
}

func (suite *KeeperTestSuite) TestGenesisImportVirtualFrontierContracts() {
	getDeployerSeq := func() uint64 {
		deployerModuleAccount := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, types.ModuleVirtualFrontierContractDeployerName)
		suite.Require().NotNil(deployerModuleAccount)
		return deployerModuleAccount.GetSequence()
	}

	// prepare
	initialDeployerSeq := getDeployerSeq()

	contractAddress1 := crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, initialDeployerSeq+0)
	contractAddress2 := crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, initialDeployerSeq+1)

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

	testAllContracts := func() {
		contract1 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress1)
		suite.Require().NotNil(contract1)

		contract2 := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, contractAddress2)
		suite.Require().NotNil(contract2)

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
	}

	suite.Run("after setup, contract should be created correctly", func() {
		testAllContracts()
	})

	suite.Run("genesis import", func() {
		deployerSeq := getDeployerSeq()

		var genesisVFCs []types.VirtualFrontierContract
		{ // export existing VFCs
			suite.app.EvmKeeper.IterateVirtualFrontierContracts(suite.ctx, func(contract types.VirtualFrontierContract) bool {
				genesisVFCs = append(genesisVFCs, contract)
				return false
			})
			suite.Require().NotEmpty(genesisVFCs)
		}

		// reset state
		suite.SetupTest()

		{ // disable create, ensure can be deployed regardless enable state
			currentParams := suite.app.EvmKeeper.GetParams(suite.ctx)
			currentParams.EnableCreate = false
			err := suite.app.EvmKeeper.SetParams(suite.ctx, currentParams)
			suite.Require().NoError(err)
			suite.Commit()
		}

		{ // restore deployer seq
			deployerModuleAccount := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, types.ModuleVirtualFrontierContractDeployerName)
			suite.Require().NotNil(deployerModuleAccount)
			err := deployerModuleAccount.SetSequence(deployerSeq)
			suite.Require().NoError(err)
			suite.app.AccountKeeper.SetModuleAccount(suite.ctx, deployerModuleAccount)
		}

		var toDeploy []types.VirtualFrontierContract
		if utils.IsEthermintDevChain(suite.ctx) { // exclude the native denom contract because it deployed by genesis
			for _, c := range genesisVFCs {
				if c.Type == types.VFC_TYPE_BANK {
					var meta types.VFBankContractMetadata
					suite.appCodec.MustUnmarshal(c.Metadata, &meta)

					if meta.MinDenom == suite.app.EvmKeeper.GetParams(suite.ctx).EvmDenom {
						continue
					}
				}

				toDeploy = append(toDeploy, c)
			}
		} else {
			toDeploy = genesisVFCs
		}

		err := suite.app.EvmKeeper.GenesisImportVirtualFrontierContracts(suite.ctx, toDeploy)
		suite.Require().NoError(err)

		testAllContracts()

		var currentVFCs []types.VirtualFrontierContract
		suite.app.EvmKeeper.IterateVirtualFrontierContracts(suite.ctx, func(contract types.VirtualFrontierContract) bool {
			currentVFCs = append(currentVFCs, contract)
			return false
		})

		suite.Equal(genesisVFCs, currentVFCs)

		for _, c := range currentVFCs {
			contractAddr := common.HexToAddress(c.Address)

			suite.Require().True(suite.app.EvmKeeper.IsVirtualFrontierContract(suite.ctx, contractAddr))

			resCode, err := suite.app.EvmKeeper.Code(suite.ctx, &types.QueryCodeRequest{
				Address: c.Address,
			})
			suite.Require().NoError(err)
			suite.Equal(types.VFBCCode, resCode.Code)

			acc := suite.app.EvmKeeper.GetAccount(suite.ctx, contractAddr)
			suite.Require().NotNil(acc)
			suite.Equal(uint64(1), acc.Nonce)
			suite.Equal(types.VFBCCodeHash, acc.CodeHash)

			if c.Type == types.VFC_TYPE_BANK {
				var meta types.VFBankContractMetadata
				suite.appCodec.MustUnmarshal(c.Metadata, &meta)

				ca, found := suite.app.EvmKeeper.GetVirtualFrontierBankContractAddressByDenom(suite.ctx, meta.MinDenom)
				suite.Require().True(found)
				suite.Equal(contractAddr, ca)
			} else {
				suite.Failf("unexpected type", "type: %s", c.Type)
			}
		}
	})
}
