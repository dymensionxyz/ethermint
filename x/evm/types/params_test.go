package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/params"

	"github.com/stretchr/testify/require"
)

func TestParamsValidate(t *testing.T) {
	newDefaultParamsWithVFContracts := func(vfContracts ...string) Params {
		params := DefaultParams()
		params.VirtualFrontierContracts = vfContracts
		return params
	}

	extraEips := []int64{2929, 1884, 1344}
	testCases := []struct {
		name     string
		params   Params
		expError bool
	}{
		{
			name:     "default",
			params:   DefaultParams(),
			expError: false,
		},
		{
			name:     "valid",
			params:   NewParams("ara", false, false, true, DefaultChainConfig(), extraEips),
			expError: false,
		},
		{
			name:     "empty",
			params:   Params{},
			expError: true,
		},
		{
			name: "invalid evm denom",
			params: Params{
				EvmDenom: "@!#!@$!@5^32",
			},
			expError: true,
		},
		{
			name: "invalid eip",
			params: Params{
				EvmDenom:  "stake",
				ExtraEIPs: []int64{1},
			},
			expError: true,
		},
		{
			name:     "creation not allowed",
			params:   NewParams("ara", false, true, true, DefaultChainConfig(), extraEips),
			expError: true,
		},
		{
			name:     "valid virtual frontier contracts",
			params:   newDefaultParamsWithVFContracts("0x405b96e2538ac85ee862e332fa634b158d013ae1", "0x9ede3180fae6322ea4fc946810152170e833ab1f"),
			expError: false,
		},
		{
			name:     "virtual frontier contracts must have prefix 0x",
			params:   newDefaultParamsWithVFContracts("405b96e2538ac85ee862e332fa634b158d013ae1", "9ede3180fae6322ea4fc946810152170e833ab1f"),
			expError: true,
		},
		{
			name:     "virtual frontier contracts must be valid addresses",
			params:   newDefaultParamsWithVFContracts("0x405b96e2538ac85ee862e332fa634b158d013ae100" /*21 bytes*/),
			expError: true,
		},
		{
			name:     "virtual frontier contracts must lowercase address",
			params:   newDefaultParamsWithVFContracts("0xAA5b96e2538ac85ee862e332fa634b158d013aBB"),
			expError: true,
		},
	}

	for _, tc := range testCases {
		err := tc.params.Validate()

		if tc.expError {
			require.Error(t, err, tc.name)
		} else {
			require.NoError(t, err, tc.name)
		}
	}
}

func TestParamsEIPs(t *testing.T) {
	extraEips := []int64{2929, 1884, 1344}
	params := NewParams("ara", false, true, true, DefaultChainConfig(), extraEips)
	actual := params.EIPs()

	require.Equal(t, []int([]int{2929, 1884, 1344}), actual)
}

func TestParamsValidatePriv(t *testing.T) {
	require.Error(t, validateEVMDenom(false))
	require.NoError(t, validateEVMDenom("inj"))
	require.Error(t, validateBool(""))
	require.NoError(t, validateBool(true))
	require.Error(t, validateEIPs(""))
	require.NoError(t, validateEIPs([]int64{1884}))
}

func TestValidateChainConfig(t *testing.T) {
	testCases := []struct {
		name     string
		i        interface{}
		expError bool
	}{
		{
			"invalid chain config type",
			"string",
			true,
		},
		{
			"valid chain config type",
			DefaultChainConfig(),
			false,
		},
	}
	for _, tc := range testCases {
		err := validateChainConfig(tc.i)

		if tc.expError {
			require.Error(t, err, tc.name)
		} else {
			require.NoError(t, err, tc.name)
		}
	}
}

func TestIsLondon(t *testing.T) {
	testCases := []struct {
		name   string
		height int64
		result bool
	}{
		{
			"Before london block",
			5,
			false,
		},
		{
			"After london block",
			12_965_001,
			true,
		},
		{
			"london block",
			12_965_000,
			true,
		},
	}

	for _, tc := range testCases {
		ethConfig := params.MainnetChainConfig
		require.Equal(t, IsLondon(ethConfig, tc.height), tc.result)
	}
}
