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

func TestIsChain(t *testing.T) {
	tests := []struct {
		name                       string
		chainId                    string
		wantIsEthermintDevChain    bool
		wantIsOneOfDymensionChains bool
	}{
		{
			name:                       "Dymension Mainnet",
			chainId:                    "dymension_1100-1",
			wantIsEthermintDevChain:    false,
			wantIsOneOfDymensionChains: true,
		},
		{
			name:                       "Dymension Mainnet",
			chainId:                    "dymension_1100-2",
			wantIsEthermintDevChain:    false,
			wantIsOneOfDymensionChains: true,
		},
		{
			name:                       "Dymension Mainnet",
			chainId:                    "dymension_1-2",
			wantIsEthermintDevChain:    false,
			wantIsOneOfDymensionChains: true,
		},
		{
			name:                       "Dymension Testnet Blumbus",
			chainId:                    "blumbus_111-1",
			wantIsEthermintDevChain:    false,
			wantIsOneOfDymensionChains: true,
		},
		{
			name:                       "Dymension Devnet Froopyland",
			chainId:                    "froopyland_100-1",
			wantIsEthermintDevChain:    false,
			wantIsOneOfDymensionChains: true,
		},
		{
			name:                       "Ethermint Devnet",
			chainId:                    "ethermint_9000-1",
			wantIsEthermintDevChain:    true,
			wantIsOneOfDymensionChains: false,
		},
		{
			name:                       "Cosmos",
			chainId:                    "cosmoshub-4",
			wantIsEthermintDevChain:    false,
			wantIsOneOfDymensionChains: false,
		},
		{
			name:                       "Evmos Mainnet",
			chainId:                    "evmos_9001-2",
			wantIsEthermintDevChain:    false,
			wantIsOneOfDymensionChains: false,
		},
		{
			name:                       "Evmos Testnet",
			chainId:                    "evmos_9000-4",
			wantIsEthermintDevChain:    false,
			wantIsOneOfDymensionChains: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sdkCtx := sdk.Context{}
			sdkCtx = sdkCtx.WithChainID(tt.chainId)
			require.Equal(t, tt.wantIsEthermintDevChain, IsEthermintDevChain(sdkCtx))
			require.Equal(t, tt.wantIsOneOfDymensionChains, IsOneOfDymensionChains(sdkCtx))
		})
	}
}
