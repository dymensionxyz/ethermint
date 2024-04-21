package types

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type VFBankContractMethod uint8

//goland:noinspection GoSnakeCaseUsage
const (
	VFBCmUnknown VFBankContractMethod = iota
	VFBCmName
	VFBCmSymbol
	VFBCmDecimals
	VFBCmTotalSupply
	VFBCmBalanceOf
	VFBCmTransfer
	VFBCmApprove_NotSupported
	VFBCmTransferFrom_NotSupported
	VFBCmAllowance_NotSupported
)

//goland:noinspection GoSnakeCaseUsage,SpellCheckingInspection
const (
	VFBCopgName               uint64 = 3400
	VFBCopgName_Revert               = VFBCopgName / 4
	VFBCopgSymbol             uint64 = 3400
	VFBCopgSymbol_Revert             = VFBCopgSymbol / 4
	VFBCopgDecimals           uint64 = 340
	VFBCopgDecimals_Revert           = VFBCopgDecimals / 4
	VFBCopgTotalSupply        uint64 = 2400
	VFBCopgTotalSupply_Revert        = VFBCopgTotalSupply / 4
	VFBCopgBalanceOf          uint64 = 2800
	VFBCopgBalanceOf_Revert          = VFBCopgBalanceOf / 4
	VFBCopgTransfer           uint64 = 13700
	VFBCopgTransfer_Revert           = VFBCopgTransfer / 4
	VFBCopgOtherNotSupported  uint64 = 100
)

// ValidateBasic performs basic validation of the VFBankContractMetadata fields
func (m *VFBankContractMetadata) ValidateBasic() error {
	if len(m.MinDenom) == 0 {
		return fmt.Errorf("min denom cannot be empty")
	}
	return nil
}

// GetMethodFromSignature returns the contract method delivers from the first 4 bytes of the input.
func (m *VFBankContractMetadata) GetMethodFromSignature(input []byte) (method VFBankContractMethod, found bool) {
	if len(input) < 4 {
		return VFBCmUnknown, false
	}

	switch strings.ToLower(hex.EncodeToString(input[:4])) {
	case "06fdde03": // first 4 bytes of the keccak256 hash of "name()"
		return VFBCmName, true
	case "95d89b41": // first 4 bytes of the keccak256 hash of "symbol()"
		return VFBCmSymbol, true
	case "313ce567": // first 4 bytes of the keccak256 hash of "decimals()"
		return VFBCmDecimals, true
	case "18160ddd": // first 4 bytes of the keccak256 hash of "totalSupply()"
		return VFBCmTotalSupply, true
	case "70a08231": // first 4 bytes of the keccak256 hash of "balanceOf(address)"
		return VFBCmBalanceOf, true
	case "a9059cbb": // first 4 bytes of the keccak256 hash of "transfer(address,uint256)"
		return VFBCmTransfer, true
	case "095ea7b3": // first 4 bytes of the keccak256 hash of "approve(address,uint256)"
		return VFBCmApprove_NotSupported, true
	case "23b872dd": // first 4 bytes of the keccak256 hash of "transferFrom(address,address,uint256)"
		return VFBCmTransferFrom_NotSupported, true
	case "dd62ed3e": // first 4 bytes of the keccak256 hash of "allowance(address,address)"
		return VFBCmAllowance_NotSupported, true
	default:
		return VFBCmUnknown, false
	}
}
