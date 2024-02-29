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
					},
					{
						ContractAddress: contractAddr2,
						Active:          false,
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
					},
					{
						ContractAddress: contractAddr1,
						Active:          false,
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
			},
			wantErr: false,
		},
		{
			name: "bad address",
			content: VirtualFrontierBankContractProposalContent{
				ContractAddress: "0xzzzzzzz",
				Active:          true,
			},
			wantErr: true,
		},
		{
			name: "bad address",
			content: VirtualFrontierBankContractProposalContent{
				ContractAddress: contractAddr1[:20],
				Active:          true,
			},
			wantErr: true,
		},
		{
			name: "bad address",
			content: VirtualFrontierBankContractProposalContent{
				ContractAddress: contractAddr1 + "00",
				Active:          true,
			},
			wantErr: true,
		},
		{
			name: "contract address must starts with 0x",
			content: VirtualFrontierBankContractProposalContent{
				ContractAddress: contractAddr1[2:],
				Active:          true,
			},
			wantErr: true,
		},
		{
			name: "contract address must be lower case",
			content: VirtualFrontierBankContractProposalContent{
				ContractAddress: strings.ToUpper(contractAddr1),
				Active:          true,
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
