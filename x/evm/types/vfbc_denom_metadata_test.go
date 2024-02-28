package types

import (
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCollectMetadataForVirtualFrontierBankContract(t *testing.T) {
	tests := []struct {
		name                      string
		bankMeta                  types.Metadata
		wantVfbcDenomMeta         VirtualFrontierBankContractDenomMetadata
		wantIsInputPassValidation bool
	}{
		{
			name: "normal, valid input",
			bankMeta: types.Metadata{
				DenomUnits: []*types.DenomUnit{
					{
						Denom:    "adym",
						Exponent: 0,
					},
					{
						Denom:    "DYM_D",
						Exponent: 18,
					},
				},
				Base:    "adym",
				Name:    "DYM_N",
				Symbol:  "DYM_S",
				Display: "DYM_D",
			},
			wantVfbcDenomMeta: VirtualFrontierBankContractDenomMetadata{
				MinDenom: "adym",
				Decimals: 18,
				Name:     "DYM_D",
				Symbol:   "DYM_S",
			},
			wantIsInputPassValidation: true,
		},
		{
			name: "input bank metadata is invalid, still process and return the result",
			bankMeta: types.Metadata{
				Base:    "adym",
				Name:    "DYM_N",
				Symbol:  "DYM_S",
				Display: "DYM_D",
			},
			wantVfbcDenomMeta: VirtualFrontierBankContractDenomMetadata{
				// still can be processed and output result
				MinDenom: "adym",
				Name:     "DYM_D",
				Symbol:   "DYM_S",
			},
			wantIsInputPassValidation: false,
		},
		{
			name: "min denom is base, regardless units",
			bankMeta: types.Metadata{
				Base: "adym",
				DenomUnits: []*types.DenomUnit{
					{
						Denom:    "anotherdym",
						Exponent: 0,
					},
				},
				Name:    "DYM_N",
				Symbol:  "DYM_S",
				Display: "DYM_D",
			},
			wantVfbcDenomMeta: VirtualFrontierBankContractDenomMetadata{
				MinDenom: "adym",
				Name:     "DYM_D",
				Symbol:   "DYM_S",
			},
			wantIsInputPassValidation: false,
		},
		{
			name: "name() priority bank metadata `display`",
			bankMeta: types.Metadata{
				Base:    "adym",
				Name:    "DYM_N",
				Symbol:  "DYM_S",
				Display: "DYM_D",
			},
			wantVfbcDenomMeta: VirtualFrontierBankContractDenomMetadata{
				MinDenom: "adym",
				Name:     "DYM_D",
				Symbol:   "DYM_S",
			},
		},
		{
			name: "name(), if missing `display`, use `name`",
			bankMeta: types.Metadata{
				Base:    "adym",
				Name:    "DYM_N",
				Symbol:  "DYM_S",
				Display: "", // missing
			},
			wantVfbcDenomMeta: VirtualFrontierBankContractDenomMetadata{
				MinDenom: "adym",
				Name:     "DYM_N",
				Symbol:   "DYM_S",
			},
		},
		{
			name: "name(), if both missing `display` and `name`, use empty",
			bankMeta: types.Metadata{
				Base:    "adym",
				Name:    "", // missing
				Symbol:  "DYM_S",
				Display: "", // missing
			},
			wantVfbcDenomMeta: VirtualFrontierBankContractDenomMetadata{
				MinDenom: "adym",
				Name:     "",
				Symbol:   "DYM_S",
			},
		},
		{
			name: "symbol(), if missing `symbol`, use empty",
			bankMeta: types.Metadata{
				Base:    "adym",
				Name:    "DYM_N",
				Symbol:  "",
				Display: "DYM_D",
			},
			wantVfbcDenomMeta: VirtualFrontierBankContractDenomMetadata{
				MinDenom: "adym",
				Name:     "DYM_D",
				Symbol:   "",
			},
		},
		{
			name: "decimals() take biggest among multiple denom of the same exponent",
			bankMeta: types.Metadata{
				DenomUnits: []*types.DenomUnit{
					{
						Denom:    "adym",
						Exponent: 0,
					},
					{
						Denom:    "DYM_1",
						Exponent: 18,
					},
					{
						Denom:    "DYM_2",
						Exponent: 18,
					},
				},
				Base:    "adym",
				Name:    "DYM_N",
				Symbol:  "DYM_S",
				Display: "DYM_D",
			},
			wantVfbcDenomMeta: VirtualFrontierBankContractDenomMetadata{
				MinDenom: "adym",
				Decimals: 18,
				Name:     "DYM_D",
				Symbol:   "DYM_S",
			},
		},
		{
			name: "decimals() take biggest among un-ordered denom units",
			bankMeta: types.Metadata{
				DenomUnits: []*types.DenomUnit{
					{
						Denom:    "DYM_1",
						Exponent: 9,
					},
					{
						Denom:    "DYM_2",
						Exponent: 18,
					},
					{
						Denom:    "adym",
						Exponent: 0,
					},
				},
				Base:    "adym",
				Name:    "DYM_N",
				Symbol:  "DYM_S",
				Display: "DYM_D",
			},
			wantVfbcDenomMeta: VirtualFrontierBankContractDenomMetadata{
				MinDenom: "adym",
				Decimals: 18,
				Name:     "DYM_D",
				Symbol:   "DYM_S",
			},
		},
		{
			name: "decimals() if missing denom units, use 0",
			bankMeta: types.Metadata{
				DenomUnits: nil,
				Base:       "adym",
				Name:       "DYM_N",
				Symbol:     "DYM_S",
				Display:    "DYM_D",
			},
			wantVfbcDenomMeta: VirtualFrontierBankContractDenomMetadata{
				MinDenom: "adym",
				Name:     "DYM_D",
				Symbol:   "DYM_S",
				Decimals: 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVfbcDenomMeta, gotIsInputPassValidation := CollectMetadataForVirtualFrontierBankContract(tt.bankMeta)
			require.Equal(t, tt.wantVfbcDenomMeta, gotVfbcDenomMeta)
			require.Equal(t, tt.wantIsInputPassValidation, gotIsInputPassValidation)
		})
	}
}
