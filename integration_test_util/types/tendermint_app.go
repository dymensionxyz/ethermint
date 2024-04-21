package types

import nm "github.com/tendermint/tendermint/node"

type TendermintApp interface {
	TendermintNode() *nm.Node
	GetRpcAddr() (addr string, supported bool)
	Shutdown()
}
