package utils

import (
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUseZeroGasConfig(t *testing.T) {
	ctx := sdk.Context{}.
		WithKVGasConfig(storetypes.KVGasConfig()).
		WithTransientKVGasConfig(storetypes.TransientGasConfig())

	require.NotZero(t, ctx.KVGasConfig().ReadCostFlat) // ensure test data

	newCtx := UseZeroGasConfig(ctx)

	newKvGasConfig := newCtx.KVGasConfig()
	require.Zero(t, newKvGasConfig.ReadCostFlat)

	newTransientKvGasConfig := newCtx.TransientKVGasConfig()
	require.Zero(t, newTransientKvGasConfig.ReadCostFlat)

	require.NotZero(t, ctx.KVGasConfig().ReadCostFlat, "the change should not effect the original context")
}
