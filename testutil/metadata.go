package testutil

import (
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"strings"
)

func NewBankDenomMetadata(denom string, decimals uint8) banktypes.Metadata {
	name := strings.ToUpper(denom[1:])
	return banktypes.Metadata{
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    denom,
				Exponent: 0,
			},
			{
				Denom:    name,
				Exponent: uint32(decimals),
			},
		},
		Base:    denom,
		Display: name,
		Name:    name,
		Symbol:  name,
	}
}
