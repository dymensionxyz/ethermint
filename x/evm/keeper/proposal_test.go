package keeper_test

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/evmos/ethermint/testutil"
	"github.com/evmos/ethermint/x/evm/types"
	"strings"
)

func (suite KeeperTestSuite) TestUpdateVirtualFrontierBankContracts() {
	deployerModuleAccount := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, types.ModuleVirtualFrontierContractDeployerName)
	suite.Require().NotNil(deployerModuleAccount)

	contractAddr1 := strings.ToLower(crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, deployerModuleAccount.GetSequence()+0).String())
	contractAddr2 := strings.ToLower(crypto.CreateAddress(types.VirtualFrontierContractDeployerAddress, deployerModuleAccount.GetSequence()+1).String())
	contractAddrNonExists := "0x0000000000000000000000000000000000009999"

	registerLegacyVFCs := func() {
		const denom1 = "uosmo"
		const denom2 = "uatom"

		meta1 := testutil.NewBankDenomMetadata(denom1, 6)
		meta2 := testutil.NewBankDenomMetadata(denom2, 6)

		suite.app.BankKeeper.SetDenomMetaData(suite.ctx, meta1)
		suite.app.BankKeeper.SetDenomMetaData(suite.ctx, meta2)

		vfbcMeta1, _ := types.CollectMetadataForVirtualFrontierBankContract(meta1)
		vfbcMeta2, _ := types.CollectMetadataForVirtualFrontierBankContract(meta2)

		addr, err := suite.app.EvmKeeper.DeployNewVirtualFrontierBankContract(
			suite.ctx,
			&types.VirtualFrontierContract{
				Active:   true,
				Type:     types.VFC_TYPE_BANK,
				Metadata: nil,
			},
			&types.VFBankContractMetadata{
				MinDenom: denom1,
			},
			&vfbcMeta1,
		)
		suite.Require().NoError(err)
		suite.Equal(contractAddr1, strings.ToLower(addr.String()))
		gotAddr, found := suite.app.EvmKeeper.GetVirtualFrontierBankContractAddressByDenom(suite.ctx, denom1)
		suite.Require().True(found)
		suite.Equal(contractAddr1, strings.ToLower(gotAddr.String()))

		addr, err = suite.app.EvmKeeper.DeployNewVirtualFrontierBankContract(
			suite.ctx,
			&types.VirtualFrontierContract{
				Active:   true,
				Type:     types.VFC_TYPE_BANK,
				Metadata: nil,
			},
			&types.VFBankContractMetadata{
				MinDenom: denom2,
			},
			&vfbcMeta2,
		)
		suite.Require().NoError(err)
		suite.Equal(contractAddr2, strings.ToLower(addr.String()))
		gotAddr, found = suite.app.EvmKeeper.GetVirtualFrontierBankContractAddressByDenom(suite.ctx, denom2)
		suite.Require().True(found)
		suite.Equal(contractAddr2, strings.ToLower(gotAddr.String()))
	}

	tests := []struct {
		name      string
		contracts []types.VirtualFrontierBankContractProposalContent
		wantErr   bool
	}{
		{
			name: "disable contract",
			contracts: []types.VirtualFrontierBankContractProposalContent{
				{
					ContractAddress: contractAddr1,
					Active:          false,
				},
			},
			wantErr: false,
		},
		{
			name: "active contract",
			contracts: []types.VirtualFrontierBankContractProposalContent{
				{
					ContractAddress: contractAddr1,
					Active:          true,
				},
			},
			wantErr: false,
		},
		{
			name: "multiple contracts",
			contracts: []types.VirtualFrontierBankContractProposalContent{
				{
					ContractAddress: contractAddr1,
					Active:          false,
				},
				{
					ContractAddress: contractAddr2,
					Active:          true,
				},
			},
			wantErr: false,
		},
		{
			name: "multiple contracts but duplicated",
			contracts: []types.VirtualFrontierBankContractProposalContent{
				{
					ContractAddress: contractAddr1,
					Active:          false,
				},
				{
					ContractAddress: contractAddr2,
					Active:          true,
				},
				{
					ContractAddress: contractAddr1,
					Active:          true,
				},
			},
			wantErr: true,
		},
		{
			name:      "not allow empty list",
			contracts: nil,
			wantErr:   true,
		},
		{
			name: "missing contract address",
			contracts: []types.VirtualFrontierBankContractProposalContent{
				{
					ContractAddress: "",
					Active:          false,
				},
			},
			wantErr: true,
		},
		{
			name: "reject non-exists contract",
			contracts: []types.VirtualFrontierBankContractProposalContent{
				{
					ContractAddress: contractAddrNonExists,
					Active:          true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()

			registerLegacyVFCs()
			suite.Commit()

			contractsAddr, err := suite.app.EvmKeeper.UpdateVirtualFrontierBankContracts(suite.ctx, tt.contracts...)
			if tt.wantErr {
				suite.Require().Error(err)
				suite.Empty(contractsAddr)
				return
			}

			suite.Require().NoError(err)

			suite.Require().Len(contractsAddr, len(tt.contracts))

			for _, updateContent := range tt.contracts {
				vfContract := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, common.HexToAddress(updateContent.ContractAddress))
				suite.Require().NotNil(vfContract)

				suite.Equal(strings.ToLower(updateContent.ContractAddress), vfContract.Address)
				suite.Equal(updateContent.Active, vfContract.Active)
				suite.Equal(types.VFC_TYPE_BANK, vfContract.Type)
				if suite.NotEmpty(vfContract.Metadata) {
					var bankContractMeta types.VFBankContractMetadata
					suite.NoError(suite.appCodec.Unmarshal(vfContract.Metadata, &bankContractMeta))
				}
			}
		})
	}
}
