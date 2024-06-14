package types

//goland:noinspection SpellCheckingInspection
import (
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	chainapp "github.com/evmos/ethermint/app"
)

var _ ChainApp = &chainAppImp{}
var _ ibctesting.TestingApp = &chainAppImp{}

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

func (c chainAppImp) PrepareProposal(proposal abci.RequestPrepareProposal) abci.ResponsePrepareProposal {
	return c.app.PrepareProposal(proposal)
}

func (c chainAppImp) ProcessProposal(proposal abci.RequestProcessProposal) abci.ResponseProcessProposal {
	return c.app.ProcessProposal(proposal)
}
