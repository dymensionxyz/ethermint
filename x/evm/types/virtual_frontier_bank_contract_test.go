package types

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestVFBankContractMetadata_ValidateBasic(t *testing.T) {
	tests := []struct {
		name            string
		meta            VFBankContractMetadata
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "normal",
			meta: VFBankContractMetadata{
				MinDenom:    "wei",
				Exponent:    18,
				DisplayName: "ETH",
			},
			wantErr:         false,
			wantErrContains: "",
		},
		{
			name: "normal, decimals=6",
			meta: VFBankContractMetadata{
				MinDenom:    "wei",
				Exponent:    6,
				DisplayName: "ETH",
			},
			wantErr:         false,
			wantErrContains: "",
		},
		{
			name: "min denom cannot be empty",
			meta: VFBankContractMetadata{
				MinDenom:    "",
				Exponent:    18,
				DisplayName: "ETH",
			},
			wantErr:         true,
			wantErrContains: "empty",
		},
		{
			name: "display name cannot be empty",
			meta: VFBankContractMetadata{
				MinDenom:    "wei",
				Exponent:    18,
				DisplayName: "",
			},
			wantErr:         true,
			wantErrContains: "empty",
		},
		{
			name: "bad decimals",
			meta: VFBankContractMetadata{
				MinDenom:    "wei",
				Exponent:    19,
				DisplayName: "ETH",
			},
			wantErr:         true,
			wantErrContains: "greater than 18",
		},
		{
			name: "min denom and display denom can not be the same, case-insensitive",
			meta: VFBankContractMetadata{
				MinDenom:    "wei",
				Exponent:    18,
				DisplayName: "WEI",
			},
			wantErr:         true,
			wantErrContains: "cannot be the same",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.meta.ValidateBasic()
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			require.ErrorContains(t, err, tt.wantErrContains)
		})
	}
}

func TestVFBankContractMetadata_GetMethodFromSignature(t *testing.T) {
	defaultMetadata := VFBankContractMetadata{}

	tests := []struct {
		name       string
		meta       VFBankContractMetadata
		input      []byte
		wantMethod VFBankContractMethod
		wantFound  bool
	}{
		{
			name:       "name",
			meta:       defaultMetadata,
			input:      []byte{0x06, 0xfd, 0xde, 0x03},
			wantMethod: VFBCmName,
			wantFound:  true,
		},
		{
			name:       "only check first 4 bytes, ignore the rest",
			meta:       defaultMetadata,
			input:      []byte{0x06, 0xfd, 0xde, 0x03, 0xff, 0xff}, /*invalid input still accepted*/
			wantMethod: VFBCmName,
			wantFound:  true,
		},
		{
			name:       "symbol",
			meta:       defaultMetadata,
			input:      []byte{0x95, 0xd8, 0x9b, 0x41},
			wantMethod: VFBCmSymbol,
			wantFound:  true,
		},
		{
			name:       "decimals",
			meta:       defaultMetadata,
			input:      []byte{0x31, 0x3c, 0xe5, 0x67},
			wantMethod: VFBCmDecimals,
			wantFound:  true,
		},
		{
			name:       "total supply",
			meta:       defaultMetadata,
			input:      []byte{0x18, 0x16, 0x0d, 0xdd},
			wantMethod: VFBCmTotalSupply,
			wantFound:  true,
		},
		{
			name:       "balance of",
			meta:       defaultMetadata,
			input:      []byte{0x70, 0xa0, 0x82, 0x31},
			wantMethod: VFBCmBalanceOf,
			wantFound:  true,
		},
		{
			name:       "transfer",
			meta:       defaultMetadata,
			input:      []byte{0xa9, 0x05, 0x9c, 0xbb},
			wantMethod: VFBCmTransfer,
			wantFound:  true,
		},
		{
			name:       "empty returns unknown",
			meta:       defaultMetadata,
			input:      []byte{},
			wantMethod: VFBCmUnknown,
			wantFound:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMethod, gotFound := tt.meta.GetMethodFromSignature(tt.input)
			require.Equal(t, tt.wantFound, gotFound)
			require.Equal(t, tt.wantMethod, gotMethod)
		})
	}
}
