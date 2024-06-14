package types

//goland:noinspection SpellCheckingInspection
import (
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	ibctransferkeeper "github.com/cosmos/ibc-go/v7/modules/apps/transfer/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"
	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"
	feemarketkeeper "github.com/evmos/ethermint/x/feemarket/keeper"
)

func (c chainAppImp) AccountKeeper() *authkeeper.AccountKeeper {
	return &c.app.AccountKeeper
}

func (c chainAppImp) BankKeeper() bankkeeper.Keeper {
	return c.app.BankKeeper
}

func (c chainAppImp) DistributionKeeper() distributionkeeper.Keeper {
	return c.app.DistrKeeper
}

func (c chainAppImp) EvmKeeper() *evmkeeper.Keeper {
	return c.app.EvmKeeper
}

func (c chainAppImp) FeeMarketKeeper() *feemarketkeeper.Keeper {
	return &c.app.FeeMarketKeeper
}

func (c chainAppImp) GovKeeper() *govkeeper.Keeper {
	return &c.app.GovKeeper
}

func (c chainAppImp) IbcTransferKeeper() *ibctransferkeeper.Keeper {
	return &c.app.TransferKeeper
}

func (c chainAppImp) IbcKeeper() *ibckeeper.Keeper {
	return c.app.IBCKeeper
}

func (c chainAppImp) SlashingKeeper() *slashingkeeper.Keeper {
	return &c.app.SlashingKeeper
}

func (c chainAppImp) StakingKeeper() *stakingkeeper.Keeper {
	return c.app.StakingKeeper
}
