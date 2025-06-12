package types

import nm "github.com/cometbft/cometbft/node"

type CometBftApp interface {
	CometBftNode() *nm.Node
	GetRpcAddr() (addr string, supported bool)
	Shutdown()
}
