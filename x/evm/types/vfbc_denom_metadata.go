package types

import (
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"math"
)

// VirtualFrontierBankContractDenomMetadata is a struct that contains the metadata of a contract denom,
// collected from the bank module's denom metadata.
type VirtualFrontierBankContractDenomMetadata struct {
	MinDenom string `json:"min_denom"` // corresponds to the "base" in the bank denom metadata
	Decimals uint32 `json:"decimals"`  // decimals is the biggest "exponent" among units of bank denom metadata
	Name     string `json:"name"`      // corresponds to the "display" (fallback to "name") in the bank denom metadata
	Symbol   string `json:"symbol"`    // corresponds to the "symbol" in the bank denom metadata
}

// CollectMetadataForVirtualFrontierBankContract collects the metadata of a contract denom from the bank module's denom metadata,
// to be used to return result for the Virtual Frontier Bank Contract execution.
func CollectMetadataForVirtualFrontierBankContract(
	metadata banktypes.Metadata,
) (
	vfbcDenomMeta VirtualFrontierBankContractDenomMetadata,
	isInputPassValidation bool,
) {
	isInputPassValidation = metadata.Validate() == nil

	var biggestExponent uint32
	for _, unit := range metadata.DenomUnits {
		if unit.Exponent > biggestExponent {
			biggestExponent = unit.Exponent
		}
	}

	// Consider the name()
	// Priority: display > name, follow spec of the bank metadata, `display` is for the client display
	name := metadata.Display
	if name == "" {
		name = metadata.Name
	}

	vfbcDenomMeta = VirtualFrontierBankContractDenomMetadata{
		MinDenom: metadata.Base,
		Decimals: biggestExponent,
		Name:     name,
		Symbol:   metadata.Symbol,
	}

	return
}

// CanDecimalsUint8 is a function that checks if the decimals fits uint8
func (m VirtualFrontierBankContractDenomMetadata) CanDecimalsUint8() bool {
	return m.Decimals <= math.MaxUint8
}
