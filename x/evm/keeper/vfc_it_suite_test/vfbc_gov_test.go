package vfc_it_suite_test

import (
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"strings"
)

func (suite *VfcITSuite) TestGovVirtualFrontierBankContract() {
	vfbcContractAddress, found := suite.App().EvmKeeper().GetVirtualFrontierBankContractAddressByDenom(suite.Ctx(), suite.CITS.ChainConstantsConfig.GetMinDenom())
	suite.Require().True(found)

	vfbcOfNative := suite.App().EvmKeeper().GetVirtualFrontierContract(suite.Ctx(), vfbcContractAddress)
	suite.Require().NotNil(vfbcOfNative)
	if !vfbcOfNative.Active {
		// enable the contract
		vfbcOfNative.Active = true
		suite.App().EvmKeeper().SetVirtualFrontierContract(suite.Ctx(), vfbcContractAddress, vfbcOfNative)
		suite.Commit()
		vfbcOfNative = suite.App().EvmKeeper().GetVirtualFrontierContract(suite.Ctx(), vfbcContractAddress)
	}
	suite.Require().True(vfbcOfNative.Active, "must be enabled")

	originalVfbcOfNative := func() *evmtypes.VirtualFrontierContract {
		ret := *vfbcOfNative

		ret.Metadata = vfbcOfNative.Metadata[:] // clone the metadata

		return &ret
	}()

	ensureOriginalImmutableFieldsUnchanged := func(vfbcOfNative *evmtypes.VirtualFrontierContract, oppositeActive bool) {
		suite.Require().Equal(originalVfbcOfNative.Address, vfbcOfNative.Address)
		suite.Require().Equal(originalVfbcOfNative.Type, vfbcOfNative.Type)
		if oppositeActive {
			suite.Require().Equal(!originalVfbcOfNative.Active, vfbcOfNative.Active)
		} else {
			suite.Require().Equal(originalVfbcOfNative.Active, vfbcOfNative.Active)
		}
		suite.Require().Equal(originalVfbcOfNative.Metadata, vfbcOfNative.Metadata)
	}

	suite.Commit()

	// create a proposal to disable the virtual frontier bank contract

	content := evmtypes.NewUpdateVirtualFrontierBankContractsProposal(
		"Disable",
		"Disable",
		evmtypes.VirtualFrontierBankContractProposalContent{
			ContractAddress: strings.ToLower(vfbcContractAddress.String()),
			Active:          false,
		},
	)

	proposalId := suite.CITS.TxFullGov(suite.CITS.WalletAccounts.Number(1), content)
	suite.Commit()

	proposal := suite.CITS.QueryGovProposalById(proposalId)
	suite.Require().Equal(govv1types.StatusPassed, proposal.Status, "proposal must be passed")

	vfbcOfNative = suite.App().EvmKeeper().GetVirtualFrontierContract(suite.Ctx(), vfbcContractAddress)
	suite.Require().NotNil(vfbcOfNative)
	suite.Require().False(vfbcOfNative.Active, "must be disabled via GOV")
	ensureOriginalImmutableFieldsUnchanged(vfbcOfNative, true)

	suite.Commit()

	// create a proposal to enable the virtual frontier bank contract again

	content = evmtypes.NewUpdateVirtualFrontierBankContractsProposal(
		"Enable",
		"Enable",
		evmtypes.VirtualFrontierBankContractProposalContent{
			ContractAddress: strings.ToLower(vfbcContractAddress.String()),
			Active:          true,
		},
	)

	proposalId = suite.CITS.TxFullGov(suite.CITS.WalletAccounts.Number(1), content)
	suite.Commit()

	proposal = suite.CITS.QueryGovProposalById(proposalId)
	suite.Require().Equal(govv1types.StatusPassed, proposal.Status, "proposal must be passed")

	vfbcOfNative = suite.App().EvmKeeper().GetVirtualFrontierContract(suite.Ctx(), vfbcContractAddress)
	suite.Require().NotNil(vfbcOfNative)
	suite.Require().True(vfbcOfNative.Active, "must be enabled via GOV")
	ensureOriginalImmutableFieldsUnchanged(vfbcOfNative, false)
}
