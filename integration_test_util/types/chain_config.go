package types

//goland:noinspection SpellCheckingInspection
import (
	"math/big"

	math "cosmossdk.io/math"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type ChainConfig struct {
	CosmosChainId            string
	BaseDenom                string
	Bech32Prefix             string
	EvmChainId               int64
	EvmChainIdBigInt         *big.Int // dynamic: calculated from EvmChainId
	DisableTendermint        bool
	DisabledContractCreation bool
}

type TestConfig struct {
	SecondaryDenomUnits      []banktypes.DenomUnit
	InitBalanceAmount        math.Int
	DefaultFeeAmount         math.Int
	DisableCometBFT          bool
	DisabledContractCreation bool
}

type ChainConstantConfig struct {
	cosmosChainId string
	minDenom      string
}

func NewChainConstantConfig(cosmosChainId, minDenom string) ChainConstantConfig {
	return ChainConstantConfig{
		cosmosChainId: cosmosChainId,
		minDenom:      minDenom,
	}
}

func (c ChainConstantConfig) GetCosmosChainID() string {
	return c.cosmosChainId
}

func (c ChainConstantConfig) GetMinDenom() string {
	return c.minDenom
}
