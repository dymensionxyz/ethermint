package types

//goland:noinspection SpellCheckingInspection
import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
	chainapp "github.com/evmos/ethermint/app"
	abci "github.com/tendermint/tendermint/abci/types"
)

var _ ChainApp = &chainAppImp{}

type chainAppImp struct {
	app *chainapp.EthermintApp
}

func (c chainAppImp) App() abci.Application {
	return c.app
}

func (c chainAppImp) BaseApp() *baseapp.BaseApp {
	return c.app.BaseApp
}

func (c chainAppImp) IbcTestingApp() ibctesting.TestingApp {
	return c
}

func (c chainAppImp) InterfaceRegistry() codectypes.InterfaceRegistry {
	return c.app.InterfaceRegistry()
}

func (c chainAppImp) FundAccount(ctx sdk.Context, account *TestAccount, amounts sdk.Coins) error {
	if err := c.BankKeeper().MintCoins(ctx, minttypes.ModuleName, amounts); err != nil {
		return err
	}

	return c.BankKeeper().SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, account.GetCosmosAddress(), amounts)
}
