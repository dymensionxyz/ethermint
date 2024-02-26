package types_test

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/ethermint/app"
	"github.com/evmos/ethermint/encoding"
	"github.com/evmos/ethermint/x/evm/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestVirtualFrontierContract_ValidateBasic(t *testing.T) {
	encodingConfig := encoding.MakeConfig(app.ModuleBasics)

	validVFBankContractMetadata := types.VFBankContractMetadata{
		MinDenom:    "wei",
		Exponent:    18,
		DisplayName: "ETH",
	}
	validVFBankContractMetadataBz := encodingConfig.Codec.MustMarshal(&validVFBankContractMetadata)

	invalidVFBankContractMetadata := types.VFBankContractMetadata{
		MinDenom:    "",
		Exponent:    18,
		DisplayName: "ETH",
	}
	invalidVFBankContractMetadataBz := encodingConfig.Codec.MustMarshal(&invalidVFBankContractMetadata)

	tests := []struct {
		name            string
		contract        types.VirtualFrontierContract
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "normal",
			contract: types.VirtualFrontierContract{
				Address:  "0x405b96e2538ac85ee862e332fa634b158d013ae1",
				Active:   true,
				Type:     uint32(types.VirtualFrontierContractTypeBankContract),
				Metadata: validVFBankContractMetadataBz,
			},
			wantErr:         false,
			wantErrContains: "",
		},
		{
			name: "normal, decimals=6",
			contract: types.VirtualFrontierContract{
				Address:  "0x405b96e2538ac85ee862e332fa634b158d013ae1",
				Active:   true,
				Type:     uint32(types.VirtualFrontierContractTypeBankContract),
				Metadata: validVFBankContractMetadataBz,
			},
			wantErr:         false,
			wantErrContains: "",
		},
		{
			name: "address can not be the nil one",
			contract: types.VirtualFrontierContract{
				Address:  "0x0000000000000000000000000000000000000000",
				Active:   true,
				Type:     uint32(types.VirtualFrontierContractTypeBankContract),
				Metadata: validVFBankContractMetadataBz,
			},
			wantErr:         true,
			wantErrContains: "nil address",
		},
		{
			name: "bad format address",
			contract: types.VirtualFrontierContract{
				Address:  "0x405b96e2538ac85ee862e332fa634b158d013ae100", // 21 bytes
				Active:   true,
				Type:     uint32(types.VirtualFrontierContractTypeBankContract),
				Metadata: validVFBankContractMetadataBz,
			},
			wantErr:         true,
			wantErrContains: "malformed address",
		},
		{
			name: "address must start with 0x",
			contract: types.VirtualFrontierContract{
				Address:  "405b96e2538ac85ee862e332fa634b158d013ae1",
				Active:   true,
				Type:     uint32(types.VirtualFrontierContractTypeBankContract),
				Metadata: validVFBankContractMetadataBz,
			},
			wantErr:         true,
			wantErrContains: "start with 0x",
		},
		{
			name: "address must be lowercase",
			contract: types.VirtualFrontierContract{
				Address:  "0xAA5b96e2538ac85ee862e332fa634b158d013aBB",
				Active:   true,
				Type:     uint32(types.VirtualFrontierContractTypeBankContract),
				Metadata: validVFBankContractMetadataBz,
			},
			wantErr:         true,
			wantErrContains: "lowercase",
		},
		{
			name: "missing address",
			contract: types.VirtualFrontierContract{
				Address:  "",
				Active:   true,
				Type:     uint32(types.VirtualFrontierContractTypeBankContract),
				Metadata: validVFBankContractMetadataBz,
			},
			wantErr:         true,
			wantErrContains: "malformed address",
		},
		{
			name: "type must be specified (not set)",
			contract: types.VirtualFrontierContract{
				Address:  "0x405b96e2538ac85ee862e332fa634b158d013ae1",
				Active:   true,
				Metadata: validVFBankContractMetadataBz,
			},
			wantErr:         true,
			wantErrContains: "type must be specified",
		},
		{
			name: "type must be specified (unknown type)",
			contract: types.VirtualFrontierContract{
				Address:  "0x405b96e2538ac85ee862e332fa634b158d013ae1",
				Active:   true,
				Type:     uint32(types.VirtualFrontierContractTypeUnknown),
				Metadata: validVFBankContractMetadataBz,
			},
			wantErr:         true,
			wantErrContains: "type must be specified",
		},
		{
			name: "invalid VF bank contract metadata",
			contract: types.VirtualFrontierContract{
				Address:  "0x405b96e2538ac85ee862e332fa634b158d013ae1",
				Active:   true,
				Type:     uint32(types.VirtualFrontierContractTypeBankContract),
				Metadata: invalidVFBankContractMetadataBz,
			},
			wantErr:         true,
			wantErrContains: "metadata does not pass validation",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.contract.ValidateBasic(encodingConfig.Codec)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			require.ErrorContains(t, err, tt.wantErrContains)
		})
	}
}

func TestVirtualFrontierContract_ContractAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    common.Address
	}{
		{
			name:    "normal",
			address: "0x405b96e2538ac85ee862e332fa634b158d013ae1",
			want:    common.HexToAddress("0x405b96e2538ac85ee862e332fa634b158d013ae1"),
		},
		{
			name:    "normal, without 0x prefix",
			address: "405b96e2538ac85ee862e332fa634b158d013ae1",
			want:    common.HexToAddress("0x405b96e2538ac85ee862e332fa634b158d013ae1"),
		},
		{
			name:    "normal, empty address",
			address: "",
			want:    common.Address{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &types.VirtualFrontierContract{
				Address: tt.address,
			}
			require.Equal(t, tt.want, m.ContractAddress())
		})
	}
}
