package types

import (
	"cosmossdk.io/errors"
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/evmos/ethermint/utils"
	"strings"
)

// ValidateBasic performs basic validation of the VirtualFrontierContract fields
func (m *VirtualFrontierContract) ValidateBasic(cdc codec.BinaryCodec) error {
	emptyAddress := common.Address{}
	if m.Address == emptyAddress.String() {
		return fmt.Errorf("address cannot be nil address")
	}
	if !common.IsHexAddress(m.Address) {
		return fmt.Errorf("malformed address format: %s", m.Address)
	}
	if !strings.HasPrefix(m.Address, "0x") {
		return fmt.Errorf("address must start with 0x")
	}
	if m.Address != strings.ToLower(m.Address) {
		return fmt.Errorf("address must be in lowercase")
	}

	if len(m.Metadata) == 0 {
		return fmt.Errorf("metadata cannot be empty")
	}

	switch m.Type {
	case VFC_TYPE_BANK:
		var bankContractMetadata VFBankContractMetadata
		var err error

		err = cdc.Unmarshal(m.Metadata, &bankContractMetadata)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal bank contract metadata")
		}

		if err = bankContractMetadata.ValidateBasic(); err != nil {
			return errors.Wrap(err, "the inner bank contract metadata does not pass validation")
		}

		break
	default:
		return fmt.Errorf("type must be specified")
	}

	return nil
}

// ContractAddress returns the contract address of the VirtualFrontierContract
func (m *VirtualFrontierContract) ContractAddress() common.Address {
	return common.HexToAddress(m.Address)
}

// GetTypeName returns the human-readable type name of the type of the VirtualFrontierContract
func (m *VirtualFrontierContract) GetTypeName() string {
	switch m.Type {
	case VFC_TYPE_BANK:
		return "bank"
	default:
		return ""
	}
}

type VFCExecutionResult struct {
	ret             []byte // return, only be set in success. If not, it is ABI-encoded Error(string) value
	opConsumeGas    uint64 // is amount of gas consume before execution revert
	optionalDescErr error
	vmErr           error
}

func NewExecVFCOutOfGas() *VFCExecutionResult {
	return &VFCExecutionResult{
		vmErr: vm.ErrOutOfGas,
	}
}

func NewExecVFCRevert(opConsumeGas uint64, optionalDescErr error) *VFCExecutionResult {
	if optionalDescErr == nil {
		optionalDescErr = vm.ErrExecutionReverted
	}
	return &VFCExecutionResult{
		opConsumeGas:    opConsumeGas,
		optionalDescErr: optionalDescErr,
		vmErr:           vm.ErrExecutionReverted,
	}
}

// NewExecVFCError creates a new VFCExecutionResult with a result error.
// The operation will consume all gas.
func NewExecVFCError(descErr error) *VFCExecutionResult {
	if descErr == nil {
		panic("descErr cannot be nil")
	}
	if descErr == vm.ErrExecutionReverted {
		panic("descErr cannot be 'execution reverted' error, use NewExecVFCRevert instead'")
	}
	return &VFCExecutionResult{
		optionalDescErr: descErr,
		vmErr:           descErr,
	}
}

// NewExecVFCSuccess creates a new VFCExecutionResult, represent for a success execution, with a result return value.
// The operation will consume exactly the amount of gas provided.
func NewExecVFCSuccess(ret []byte, opConsumeGas uint64) *VFCExecutionResult {
	return &VFCExecutionResult{
		ret:          ret,
		opConsumeGas: opConsumeGas,
	}
}

// NewExecVFCSuccessWithRetBool is the same as NewExecVFCSuccess.
// The operation will consume exactly the amount of gas provided.
func NewExecVFCSuccessWithRetBool(val bool, opConsumeGas uint64) *VFCExecutionResult {
	ret := make([]byte, 32)
	if val {
		ret[31] = 1
	}
	return NewExecVFCSuccess(ret, opConsumeGas)
}

func (m VFCExecutionResult) GetDetailedResult(startGas uint64) (success bool, ret []byte, isExecutionRevertedOnFalse bool, leftOverGas uint64, vmErr error) {
	success = m.vmErr == nil
	isExecutionRevertedOnFalse = m.vmErr == vm.ErrExecutionReverted
	vmErr = m.vmErr

	var gasToConsume uint64

	if success {
		// logic
		ret = m.ret

		gasToConsume = m.opConsumeGas // consume exactly as provided
	} else {
		if m.vmErr == vm.ErrOutOfGas {
			gasToConsume = startGas // consume all gas
		} else if m.vmErr == vm.ErrExecutionReverted {
			gasToConsume = m.opConsumeGas
		} else {
			gasToConsume = startGas // consume all gas on non-execution reverted error
		}

		// building ret, based on the error
		var retErrorContent string
		if m.optionalDescErr != nil {
			retErrorContent = m.optionalDescErr.Error()
		} else {
			retErrorContent = m.vmErr.Error()
		}

		ret = append(
			[]byte{0x08, 0xc3, 0x79, 0xa0}, // signature
			utils.MustAbiEncodeString(retErrorContent)...,
		)
	}

	if gasToConsume > startGas {
		panic(fmt.Sprintf("gas overflow: consume %d, limit %d", gasToConsume, startGas))
	}

	leftOverGas = startGas - gasToConsume // overflow had been checked just above

	return
}
