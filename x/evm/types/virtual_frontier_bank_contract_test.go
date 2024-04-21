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
				MinDenom: "wei",
			},
			wantErr:         false,
			wantErrContains: "",
		},
		{
			name: "min denom cannot be empty",
			meta: VFBankContractMetadata{
				MinDenom: "",
			},
			wantErr:         true,
			wantErrContains: "empty",
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
			name:       "approve",
			meta:       defaultMetadata,
			input:      []byte{0x09, 0x5e, 0xa7, 0xb3},
			wantMethod: VFBCmApprove_NotSupported,
			wantFound:  true,
		},
		{
			name:       "transfer from",
			meta:       defaultMetadata,
			input:      []byte{0x23, 0xb8, 0x72, 0xdd},
			wantMethod: VFBCmTransferFrom_NotSupported,
			wantFound:  true,
		},
		{
			name:       "allowance",
			meta:       defaultMetadata,
			input:      []byte{0xdd, 0x62, 0xed, 0x3e},
			wantMethod: VFBCmAllowance_NotSupported,
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
			meta := VFBankContractMetadata{}
			gotMethod, gotFound := meta.GetMethodFromSignature(tt.input)
			require.Equal(t, tt.wantFound, gotFound)
			require.Equal(t, tt.wantMethod, gotMethod)
		})
	}
}
