package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/evmos/ethermint/x/evm/client/cli"
)

var (
	UpdateVirtualFrontierBankContractProposalHandler = govclient.NewProposalHandler(cli.NewUpdateVirtualFrontierBankContractCmd)
)
