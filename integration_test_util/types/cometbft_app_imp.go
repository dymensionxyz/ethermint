package types

import (
	"fmt"
	cmtnode "github.com/cometbft/cometbft/node"
	"strings"
)

var _ CometBftApp = &cometBftAppImp{}

type cometBftAppImp struct {
	cometNode *cmtnode.Node
	rpcAddr   string
	grpcAddr  string
}

func NewCometBftApp(cometNode *cmtnode.Node, rpcPort int) CometBftApp {
	app := &cometBftAppImp{
		cometNode: cometNode,
	}
	if rpcPort > 0 {
		app.rpcAddr = fmt.Sprintf("tcp://localhost:%d", rpcPort)
	}
	return app
}

func (a *cometBftAppImp) CometBftNode() *cmtnode.Node {
	return a.cometNode
}

func (a *cometBftAppImp) GetRpcAddr() (addr string, supported bool) {
	return a.rpcAddr, a.rpcAddr != ""
}

func (a *cometBftAppImp) Shutdown() {
	if a == nil || a.cometNode == nil || !a.cometNode.IsRunning() {
		return
	}
	err := a.cometNode.Stop()
	if err != nil {
		if strings.Contains(err.Error(), "already stopped") {
			// ignore
		} else {
			fmt.Println("Failed to stop CometBFT node")
			fmt.Println(err)
		}
	}
	a.cometNode.Wait()
}
