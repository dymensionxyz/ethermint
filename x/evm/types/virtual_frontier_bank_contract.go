package types

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type VFBankContractMethod uint8

const (
	VFBCmUnknown VFBankContractMethod = iota
	VFBCmName
	VFBCmSymbol
	VFBCmDecimals
	VFBCmTotalSupply
	VFBCmBalanceOf
	VFBCmTransfer
)

// ValidateBasic performs basic validation of the VFBankContractMetadata fields
func (m *VFBankContractMetadata) ValidateBasic() error {
	if len(m.MinDenom) == 0 {
		return fmt.Errorf("min denom cannot be empty")
	}
	if m.Exponent > 18 {
		return fmt.Errorf("exponent cannot be greater than 18")
	}
	if len(m.DisplayName) == 0 {
		return fmt.Errorf("display name cannot be empty")
	}
	if strings.EqualFold(m.MinDenom, m.DisplayName) {
		return fmt.Errorf("min denom and display name cannot be the same")
	}
	return nil
}

// GetMethodFromSignature returns the contract method delivers from the first 4 bytes of the input.
func (m *VFBankContractMetadata) GetMethodFromSignature(input []byte) (method VFBankContractMethod, found bool) {
	if len(input) < 4 {
		return VFBCmUnknown, false
	}

	switch strings.ToLower(hex.EncodeToString(input[:4])) {
	case "06fdde03":
		return VFBCmName, true
	case "95d89b41":
		return VFBCmSymbol, true
	case "313ce567":
		return VFBCmDecimals, true
	case "18160ddd":
		return VFBCmTotalSupply, true
	case "70a08231":
		return VFBCmBalanceOf, true
	case "a9059cbb":
		return VFBCmTransfer, true
	default:
		return VFBCmUnknown, false
	}
}
