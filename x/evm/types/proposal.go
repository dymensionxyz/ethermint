package types

import (
	"fmt"
	v1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/ethereum/go-ethereum/common"
	"regexp"
	"strings"
)

// constants
const (
	ProposalTypeUpdateVirtualFrontierBankContractsProposal string = "UpdateVirtualFrontierBankContractsProposal"
)

// Implements Proposal Interface
var (
	_ v1beta1.Content = &UpdateVirtualFrontierBankContractsProposal{}
)

func init() {
	v1beta1.RegisterProposalType(ProposalTypeUpdateVirtualFrontierBankContractsProposal)
	v1beta1.ModuleCdc.Amino.RegisterConcrete(&UpdateVirtualFrontierBankContractsProposal{}, "evm/UpdateVirtualFrontierBankContractsProposal", nil)
}

// NewUpdateVirtualFrontierBankContractsProposal returns new instance of UpdateVirtualFrontierBankContractsProposal
func NewUpdateVirtualFrontierBankContractsProposal(title, description string, contracts ...VirtualFrontierBankContractProposalContent) v1beta1.Content {
	return &UpdateVirtualFrontierBankContractsProposal{
		Title:       title,
		Description: description,
		Contracts:   contracts,
	}
}

// ProposalRoute returns router key for this proposal
func (*UpdateVirtualFrontierBankContractsProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*UpdateVirtualFrontierBankContractsProposal) ProposalType() string {
	return ProposalTypeUpdateVirtualFrontierBankContractsProposal
}

// ValidateBasic performs a stateless check of the proposal fields
func (m *UpdateVirtualFrontierBankContractsProposal) ValidateBasic() error {
	if len(m.Contracts) == 0 {
		return fmt.Errorf("missing contract list")
	}

	var uniqueContracts = make(map[common.Address]bool)

	for _, contract := range m.Contracts {
		if err := contract.ValidateBasic(); err != nil {
			return err
		}

		contractAddress := common.HexToAddress(contract.ContractAddress)
		if _, found := uniqueContracts[contractAddress]; found {
			return fmt.Errorf("duplicate update for contract address: %s", contract.ContractAddress)
		}

		uniqueContracts[contractAddress] = true
	}

	return v1beta1.ValidateAbstract(m)
}

// ValidateBasic performs a stateless check of the proposal fields
func (m *VirtualFrontierBankContractProposalContent) ValidateBasic() error {
	if m.ContractAddress == "" {
		return fmt.Errorf("missing contract address")
	}
	if !common.IsHexAddress(m.ContractAddress) {
		return fmt.Errorf("invalid contract address")
	}
	if strings.ToLower(m.ContractAddress) != m.ContractAddress {
		return fmt.Errorf("contract address must be in lowercase")
	}
	if !strings.HasPrefix(m.ContractAddress, "0x") {
		return fmt.Errorf("contract address must start with 0x")
	}

	if m.DisplayName == "" {
		return fmt.Errorf("missing display name")
	}
	if !regexp.MustCompile(`^[a-zA-Z\d\-_/]+$`).MatchString(m.DisplayName) {
		return fmt.Errorf("accepted characters in display name are: a-z, A-Z, 0-9, -, _ and /")
	}

	if m.Exponent > 18 {
		return fmt.Errorf("exponent must be less than or equal to 18")
	}

	return nil
}