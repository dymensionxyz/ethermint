package types_test

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/evmos/ethermint/app"
	"github.com/evmos/ethermint/encoding"
	"github.com/evmos/ethermint/utils"
	"github.com/evmos/ethermint/x/evm/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestVirtualFrontierContract_ValidateBasic(t *testing.T) {
	encodingConfig := encoding.MakeConfig(app.ModuleBasics)

	validVFBankContractMetadata := types.VFBankContractMetadata{
		MinDenom: "wei",
	}
	validVFBankContractMetadataBz := encodingConfig.Codec.MustMarshal(&validVFBankContractMetadata)

	invalidVFBankContractMetadata := types.VFBankContractMetadata{
		MinDenom: "",
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
				Type:     types.VFC_TYPE_BANK,
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
				Type:     types.VFC_TYPE_BANK,
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
				Type:     types.VFC_TYPE_BANK,
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
				Type:     types.VFC_TYPE_BANK,
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
				Type:     types.VFC_TYPE_BANK,
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
				Type:     types.VFC_TYPE_BANK,
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
			name: "type must be specified",
			contract: types.VirtualFrontierContract{
				Address:  "0x405b96e2538ac85ee862e332fa634b158d013ae1",
				Active:   true,
				Type:     types.VFC_TYPE_UNSPECIFIED,
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
				Type:     types.VFC_TYPE_BANK,
				Metadata: invalidVFBankContractMetadataBz,
			},
			wantErr:         true,
			wantErrContains: "metadata cannot be empty",
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

func TestVFCExecutionResult(t *testing.T) {
	tests := []struct {
		name                    string
		inputFunc               func() *types.VFCExecutionResult
		inputStartGas           uint64
		wantPanicCreateInstance bool
		wantPanic               bool
		wantSuccess             bool
		wantRetErrorString      bool
		wantRetWhenSuccess      []byte
		wantIsErrExecReverted   bool
		wantLeftOverGas         uint64
		wantRetErrorContains    string
		wantEvmErrorContains    string
	}{
		{
			name: "when out of gas",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCOutOfGas()
			},
			inputStartGas:           100_000,
			wantPanicCreateInstance: false,
			wantPanic:               false,
			wantSuccess:             false,
			wantRetErrorString:      true,
			wantRetWhenSuccess:      nil,
			wantIsErrExecReverted:   false,
			wantLeftOverGas:         0,
			wantRetErrorContains:    vm.ErrOutOfGas.Error(),
			wantEvmErrorContains:    vm.ErrOutOfGas.Error(),
		},
		{
			name: "when revert with op gas consume and detailed error",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCRevert(10, fmt.Errorf("pseudo 1"))
			},
			inputStartGas:           100_000,
			wantPanicCreateInstance: false,
			wantPanic:               false,
			wantSuccess:             false,
			wantRetErrorString:      true,
			wantRetWhenSuccess:      nil,
			wantIsErrExecReverted:   true,
			wantLeftOverGas:         99_990,
			wantRetErrorContains:    "pseudo 1",
			wantEvmErrorContains:    vm.ErrExecutionReverted.Error(),
		},
		{
			name: "when revert with op gas consume but without provide error",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCRevert(10, nil)
			},
			inputStartGas:           100_000,
			wantPanicCreateInstance: false,
			wantPanic:               false,
			wantSuccess:             false,
			wantRetErrorString:      true,
			wantRetWhenSuccess:      nil,
			wantIsErrExecReverted:   true,
			wantLeftOverGas:         99_990,
			wantRetErrorContains:    vm.ErrExecutionReverted.Error(), // copied
			wantEvmErrorContains:    vm.ErrExecutionReverted.Error(),
		},
		{
			name: "when revert with detailed error but no op gas",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCRevert(0, fmt.Errorf("pseudo 2"))
			},
			inputStartGas:           100_000,
			wantPanicCreateInstance: false,
			wantPanic:               false,
			wantSuccess:             false,
			wantRetErrorString:      true,
			wantRetWhenSuccess:      nil,
			wantIsErrExecReverted:   true,
			wantLeftOverGas:         100_000, // consume zero gas
			wantRetErrorContains:    "pseudo 2",
			wantEvmErrorContains:    vm.ErrExecutionReverted.Error(),
		},
		{
			name: "when revert without op gas and reason",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCRevert(0, nil)
			},
			inputStartGas:           100_000,
			wantPanicCreateInstance: false,
			wantPanic:               false,
			wantSuccess:             false,
			wantRetErrorString:      true,
			wantRetWhenSuccess:      nil,
			wantIsErrExecReverted:   true,
			wantLeftOverGas:         100_000,                         // consume zero gas
			wantRetErrorContains:    vm.ErrExecutionReverted.Error(), // copied
			wantEvmErrorContains:    vm.ErrExecutionReverted.Error(),
		},
		{
			name: "when revert with op gas consume larger than start gas",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCRevert(999_999, fmt.Errorf("pseudo 3"))
			},
			inputStartGas: 100_000,
			wantPanic:     true,
		},
		{
			name: "when error",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCError(fmt.Errorf("pseudo 4"))
			},
			inputStartGas:           100_000,
			wantPanicCreateInstance: false,
			wantPanic:               false,
			wantSuccess:             false,
			wantRetErrorString:      true,
			wantRetWhenSuccess:      nil,
			wantIsErrExecReverted:   false,
			wantLeftOverGas:         0, // should consume all gas
			wantRetErrorContains:    "pseudo 4",
			wantEvmErrorContains:    "pseudo 4",
		},
		{
			name: "when error",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCError(fmt.Errorf("pseudo 4"))
			},
			inputStartGas:           100_000,
			wantPanicCreateInstance: false,
			wantPanic:               false,
			wantSuccess:             false,
			wantRetErrorString:      true,
			wantRetWhenSuccess:      nil,
			wantIsErrExecReverted:   false,
			wantLeftOverGas:         0, // should consume all gas
			wantRetErrorContains:    "pseudo 4",
			wantEvmErrorContains:    "pseudo 4",
		},
		{
			name: "when error, error details must be provided",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCError(nil)
			},
			wantPanicCreateInstance: true,
		},
		{
			name: "when error, not accept details error is 'execution reverted'",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCError(vm.ErrExecutionReverted)
			},
			wantPanicCreateInstance: true,
		},
		{
			name: "when exec success",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCSuccess([]byte{0x19, 0x92}, 30_000)
			},
			inputStartGas:           100_000,
			wantPanicCreateInstance: false,
			wantPanic:               false,
			wantSuccess:             true,
			wantRetErrorString:      false,
			wantRetWhenSuccess:      []byte{0x19, 0x92},
			wantIsErrExecReverted:   false,
			wantLeftOverGas:         70_000,
			wantRetErrorContains:    "",
			wantEvmErrorContains:    "",
		},
		{
			name: "when exec success without ret",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCSuccess(nil, 30_000)
			},
			inputStartGas:           100_000,
			wantPanicCreateInstance: false,
			wantPanic:               false,
			wantSuccess:             true,
			wantRetErrorString:      false,
			wantRetWhenSuccess:      nil, // keep as is
			wantIsErrExecReverted:   false,
			wantLeftOverGas:         70_000,
			wantRetErrorContains:    "",
			wantEvmErrorContains:    "",
		},
		{
			name: "when exec success without gas consumption",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCSuccess([]byte{0x19, 0x92}, 0)
			},
			inputStartGas:           100_000,
			wantPanicCreateInstance: false,
			wantPanic:               false,
			wantSuccess:             true,
			wantRetErrorString:      false,
			wantRetWhenSuccess:      []byte{0x19, 0x92},
			wantIsErrExecReverted:   false,
			wantLeftOverGas:         100_000,
			wantRetErrorContains:    "",
			wantEvmErrorContains:    "",
		},
		{
			name: "when exec success but gas consumption greater than start gas",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCSuccess([]byte{0x19, 0x92}, 999_999)
			},
			inputStartGas:           100_000,
			wantPanicCreateInstance: false,
			wantPanic:               true,
		},
		{
			name: "when exec success, ret bool true",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCSuccessWithRetBool(true, 30_000)
			},
			inputStartGas:           100_000,
			wantPanicCreateInstance: false,
			wantPanic:               false,
			wantSuccess:             true,
			wantRetErrorString:      false,
			wantRetWhenSuccess: func() []byte {
				var ret []byte

				ret = make([]byte, 32)
				ret[31] = 0x1

				return ret
			}(),
			wantIsErrExecReverted: false,
			wantLeftOverGas:       70_000,
			wantRetErrorContains:  "",
			wantEvmErrorContains:  "",
		},
		{
			name: "when exec success, ret bool false",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCSuccessWithRetBool(false, 30_000)
			},
			inputStartGas:           100_000,
			wantPanicCreateInstance: false,
			wantPanic:               false,
			wantSuccess:             true,
			wantRetErrorString:      false,
			wantRetWhenSuccess:      make([]byte, 32),
			wantIsErrExecReverted:   false,
			wantLeftOverGas:         70_000,
			wantRetErrorContains:    "",
			wantEvmErrorContains:    "",
		},
		{
			name: "when exec success, ret bool, but gas consumption greater than start gas",
			inputFunc: func() *types.VFCExecutionResult {
				return types.NewExecVFCSuccessWithRetBool(true, 999_999)
			},
			inputStartGas:           100_000,
			wantPanicCreateInstance: false,
			wantPanic:               true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanicCreateInstance {
				require.Panics(t, func() {
					_ = tt.inputFunc()
				})
				return
			}

			input := tt.inputFunc()

			if tt.wantPanic {
				require.Panics(t, func() {
					_, _, _, _, _ = input.GetDetailedResult(tt.inputStartGas)
				})
				return
			}

			gotSuccess, gotRet, gotIsErrExecReverted, gotLeftOverGas, gotVmErr := input.GetDetailedResult(tt.inputStartGas)

			if tt.wantSuccess {
				require.True(t, gotSuccess)
				if len(tt.wantRetWhenSuccess) == 0 && len(gotRet) == 0 {
					// ok
				} else {
					require.Equal(t, tt.wantRetWhenSuccess, gotRet)
				}
				require.False(t, gotIsErrExecReverted)
				require.Equal(t, tt.wantLeftOverGas, gotLeftOverGas)
				require.Nil(t, gotVmErr)
			} else {
				require.False(t, gotSuccess)
				require.NotEmpty(t, gotRet, "ret must be ABI-encoded ABI(string)")
				require.Contains(t, "0x"+hex.EncodeToString(gotRet), "0x08c379a0", "ret must be ABI-encoded ABI(string)")
				require.Equal(t, tt.wantIsErrExecReverted, gotIsErrExecReverted)
				require.Equal(t, tt.wantLeftOverGas, gotLeftOverGas)
				require.NotNil(t, gotVmErr)
				require.NotEmpty(t, tt.wantRetErrorContains, "bad setup test-case")
				require.NotEmpty(t, tt.wantEvmErrorContains, "bad setup test-case")
				require.Contains(t, gotVmErr.Error(), tt.wantEvmErrorContains)

				errMsg := utils.MustAbiDecodeString(gotRet[4:] /*skip 4bytes sig*/)
				require.Contains(t, errMsg, tt.wantRetErrorContains)
			}
		})
	}
}
