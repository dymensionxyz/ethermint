package utils

import (
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// UseZeroGasConfig set the gas config to zero for both KV and TransientKV store.
// Must be called before EVM execution to ignore gas consumption on Cosmos side.
// Gas consumption should be decided by Ethereum side.
func UseZeroGasConfig(ctx sdk.Context) sdk.Context {
	return ctx.WithKVGasConfig(storetypes.GasConfig{}).WithTransientKVGasConfig(storetypes.GasConfig{})
}
