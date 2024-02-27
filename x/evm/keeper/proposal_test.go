package keeper_test

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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
		addr, err := suite.app.EvmKeeper.DeployNewVirtualFrontierBankContract(
			suite.ctx,
			&types.VirtualFrontierContract{
				Active:           true,
				Type:             uint32(types.VirtualFrontierContractTypeBankContract),
				Metadata:         nil,
				LastUpdateHeight: 0,
			},
			&types.VFBankContractMetadata{
				MinDenom:    "uosmo",
				Exponent:    6,
				DisplayName: "OSMO",
			},
		)
		suite.Require().NoError(err)
		suite.Equal(contractAddr1, strings.ToLower(addr.String()))
		addr, err = suite.app.EvmKeeper.DeployNewVirtualFrontierBankContract(
			suite.ctx,
			&types.VirtualFrontierContract{
				Active:           true,
				Type:             uint32(types.VirtualFrontierContractTypeBankContract),
				Metadata:         nil,
				LastUpdateHeight: 0,
			},
			&types.VFBankContractMetadata{
				MinDenom:    "uatom",
				Exponent:    6,
				DisplayName: "ATOM",
			},
		)
		suite.Require().NoError(err)
		suite.Equal(contractAddr2, strings.ToLower(addr.String()))
	}

	tests := []struct {
		name      string
		contracts []types.VirtualFrontierBankContractProposalContent
		wantErr   bool
	}{
		{
			name: "normal",
			contracts: []types.VirtualFrontierBankContractProposalContent{
				{
					ContractAddress: contractAddr1,
					Active:          false,
					DisplayName:     "CHANGED",
					Exponent:        16,
				},
			},
			wantErr: false,
		},
		{
			name: "normal, multiple",
			contracts: []types.VirtualFrontierBankContractProposalContent{
				{
					ContractAddress: contractAddr1,
					Active:          false,
					DisplayName:     "CHANGED",
					Exponent:        18,
				},
				{
					ContractAddress: contractAddr2,
					Active:          true,
					DisplayName:     "CHANGED",
					Exponent:        12,
				},
			},
			wantErr: false,
		},
		{
			name:      "not allow empty list",
			contracts: nil,
			wantErr:   true,
		},
		{
			name: "invalid contract content",
			contracts: []types.VirtualFrontierBankContractProposalContent{
				{
					ContractAddress: contractAddr1,
					Active:          false,
					DisplayName:     "^^",
					Exponent:        16,
				},
			},
			wantErr: true,
		},
		{
			name: "can not be the same as min denom",
			contracts: []types.VirtualFrontierBankContractProposalContent{
				{
					ContractAddress: contractAddr1,
					Active:          false,
					DisplayName:     "uosmo",
					Exponent:        16,
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
					DisplayName:     "OSMO",
					Exponent:        6,
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

			blockNumber := suite.ctx.BlockHeight()
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
				suite.Equal(uint32(types.VirtualFrontierContractTypeBankContract), vfContract.Type)
				if suite.NotEmpty(vfContract.Metadata) {
					var bankContractMeta types.VFBankContractMetadata
					suite.NoError(suite.appCodec.Unmarshal(vfContract.Metadata, &bankContractMeta))
					suite.Equal(updateContent.DisplayName, bankContractMeta.DisplayName)
					suite.Equal(updateContent.Exponent, bankContractMeta.Exponent)
				}
				suite.Equal(uint64(blockNumber), vfContract.LastUpdateHeight)
			}
		})
	}
}
