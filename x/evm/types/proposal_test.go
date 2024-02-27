package types

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestUpdateVirtualFrontierBankContractsProposal_ValidateBasic(t *testing.T) {
	contractAddr1 := "0x0000000000000000000000000000000000002001"
	contractAddr2 := "0x0000000000000000000000000000000000002002"

	tests := []struct {
		name     string
		proposal UpdateVirtualFrontierBankContractsProposal
		wantErr  bool
	}{
		{
			name: "normal",
			proposal: UpdateVirtualFrontierBankContractsProposal{
				Contracts: []VirtualFrontierBankContractProposalContent{
					{
						ContractAddress: contractAddr1,
						Active:          true,
						DisplayName:     "OSMO",
						Exponent:        6,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "normal, multiple",
			proposal: UpdateVirtualFrontierBankContractsProposal{
				Contracts: []VirtualFrontierBankContractProposalContent{
					{
						ContractAddress: contractAddr1,
						Active:          true,
						DisplayName:     "OSMO",
						Exponent:        6,
					},
					{
						ContractAddress: contractAddr2,
						Active:          false,
						DisplayName:     "ATOM",
						Exponent:        6,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicated record for the same contract",
			proposal: UpdateVirtualFrontierBankContractsProposal{
				Contracts: []VirtualFrontierBankContractProposalContent{
					{
						ContractAddress: contractAddr1,
						Active:          true,
						DisplayName:     "OSMO1",
						Exponent:        6,
					},
					{
						ContractAddress: contractAddr1,
						Active:          false,
						DisplayName:     "OSMO2",
						Exponent:        6,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "inner content is not valid",
			proposal: UpdateVirtualFrontierBankContractsProposal{
				Contracts: []VirtualFrontierBankContractProposalContent{
					{
						ContractAddress: contractAddr1,
						Active:          true,
						DisplayName:     "OSMO",
						Exponent:        20, // wrong
					},
				},
			},
			wantErr: true,
		},
		{
			name: "reject empty",
			proposal: UpdateVirtualFrontierBankContractsProposal{
				Contracts: []VirtualFrontierBankContractProposalContent{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal := tt.proposal
			proposal.Title = "Title"
			proposal.Description = "Description"
			err := proposal.ValidateBasic()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestVirtualFrontierBankContractProposalContent_ValidateBasic(t *testing.T) {
	contractAddr1 := "0x0000000000000000000000000000000000002001"

	tests := []struct {
		name    string
		content VirtualFrontierBankContractProposalContent
		wantErr bool
	}{
		{
			name: "normal",
			content: VirtualFrontierBankContractProposalContent{
				ContractAddress: contractAddr1,
				Active:          true,
				DisplayName:     "OSMO",
				Exponent:        6,
			},
			wantErr: false,
		},
		{
			name: "bad address",
			content: VirtualFrontierBankContractProposalContent{
				ContractAddress: "0xzzzzzzz",
				Active:          true,
				DisplayName:     "OSMO",
				Exponent:        6,
			},
			wantErr: true,
		},
		{
			name: "bad address",
			content: VirtualFrontierBankContractProposalContent{
				ContractAddress: contractAddr1[:20],
				Active:          true,
				DisplayName:     "OSMO",
				Exponent:        6,
			},
			wantErr: true,
		},
		{
			name: "bad address",
			content: VirtualFrontierBankContractProposalContent{
				ContractAddress: contractAddr1 + "00",
				Active:          true,
				DisplayName:     "OSMO",
				Exponent:        6,
			},
			wantErr: true,
		},
		{
			name: "contract address must starts with 0x",
			content: VirtualFrontierBankContractProposalContent{
				ContractAddress: contractAddr1[2:],
				Active:          true,
				DisplayName:     "OSMO",
				Exponent:        6,
			},
			wantErr: true,
		},
		{
			name: "contract address must be lower case",
			content: VirtualFrontierBankContractProposalContent{
				ContractAddress: strings.ToUpper(contractAddr1),
				Active:          true,
				DisplayName:     "OSMO",
				Exponent:        6,
			},
			wantErr: true,
		},
		{
			name: "display name is required",
			content: VirtualFrontierBankContractProposalContent{
				ContractAddress: contractAddr1,
				Active:          true,
				DisplayName:     "",
				Exponent:        6,
			},
			wantErr: true,
		},
		{
			name: "exponent must <= 18",
			content: VirtualFrontierBankContractProposalContent{
				ContractAddress: contractAddr1,
				Active:          true,
				DisplayName:     "OSMO",
				Exponent:        18,
			},
			wantErr: false,
		},
		{
			name: "reject exponent > 18",
			content: VirtualFrontierBankContractProposalContent{
				ContractAddress: contractAddr1,
				Active:          true,
				DisplayName:     "OSMO",
				Exponent:        19,
			},
			wantErr: true,
		},
		{
			name: "reject bad display name",
			content: VirtualFrontierBankContractProposalContent{
				ContractAddress: contractAddr1,
				Active:          true,
				DisplayName:     "<script>alert('xss')</script>",
				Exponent:        6,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.content.ValidateBasic()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
