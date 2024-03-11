package types

import (
	transfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
)

const (
	// TestnetChainID defines the Evmos EIP155 chain ID for testnet
	TestnetChainID = "ethermint_9000"
	// BaseDenom defines the Evmos mainnet denomination
	BaseDenom = "aphoton"
)

var (
	UosmoDenomtrace = transfertypes.DenomTrace{
		Path:      "transfer/channel-0",
		BaseDenom: "uosmo",
	}
	UosmoIbcdenom = UosmoDenomtrace.IBCDenom()

	UatomDenomtrace = transfertypes.DenomTrace{
		Path:      "transfer/channel-1",
		BaseDenom: "uatom",
	}
	UatomIbcdenom = UatomDenomtrace.IBCDenom()

	UevmosDenomtrace = transfertypes.DenomTrace{
		Path:      "transfer/channel-0",
		BaseDenom: "aevmos",
	}
	UevmosIbcdenom = UevmosDenomtrace.IBCDenom()

	UatomOsmoDenomtrace = transfertypes.DenomTrace{
		Path:      "transfer/channel-0/transfer/channel-1",
		BaseDenom: "uatom",
	}
	UatomOsmoIbcdenom = UatomOsmoDenomtrace.IBCDenom()

	AevmosDenomtrace = transfertypes.DenomTrace{
		Path:      "transfer/channel-0",
		BaseDenom: "aevmos",
	}
	AevmosIbcdenom = AevmosDenomtrace.IBCDenom()
)
