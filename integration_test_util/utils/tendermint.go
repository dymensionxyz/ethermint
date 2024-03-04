package utils

//goland:noinspection SpellCheckingInspection
import (
	"context"
	"fmt"
	cdb "github.com/cometbft/cometbft-db"
	"github.com/google/uuid"
	abci "github.com/tendermint/tendermint/abci/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/log"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	tmcorestypes "github.com/tendermint/tendermint/rpc/core/types"
	tmgrpc "github.com/tendermint/tendermint/rpc/grpc"
	tmrpcclient "github.com/tendermint/tendermint/rpc/jsonrpc/client"
	rpctest "github.com/tendermint/tendermint/rpc/test"
	tmtypes "github.com/tendermint/tendermint/types"
	"time"
)

// StartTendermintNode starts a Tendermint node for the given ABCI Application, used for testing purposes.
func StartTendermintNode(app abci.Application, genesis *tmtypes.GenesisDoc, db cdb.DB, validatorPrivKey tmcrypto.PrivKey, logger log.Logger) (tendermintNode *nm.Node, rpcPort int, tempFiles []string) {
	if app == nil {
		panic("missing app")
	}
	if genesis == nil {
		panic("missing genesis")
	}
	if db == nil {
		panic("missing db")
	}
	if validatorPrivKey == nil {
		panic("missing validator private key")
	}

	useRpc := true
	useGrpc := false

	// Create & start node
	config := rpctest.GetConfig(false)

	// timeout commit is not a big, not a small number, but enough to broadcast amount of txs
	config.Consensus.TimeoutCommit = 500 * time.Millisecond
	config.Consensus.SkipTimeoutCommit = false // don't use default (which is true), because the block procedures too fast

	var portRpc, portGrpc int

	config.ProxyApp = fmt.Sprintf("tcp://localhost:%d", GetNextPortAvailable())

	if useRpc {
		portRpc = GetNextPortAvailable()
		config.RPC.ListenAddress = fmt.Sprintf("tcp://localhost:%d", portRpc)
	} else {
		config.RPC.ListenAddress = ""
	}

	if useGrpc {
		portGrpc = GetNextPortAvailable()
		config.RPC.GRPCListenAddress = fmt.Sprintf("tcp://localhost:%d", portGrpc)
	} else {
		config.RPC.GRPCListenAddress = ""
	}

	config.RPC.PprofListenAddress = "" // fmt.Sprintf("tcp://localhost:%d", GetNextPortAvailable())

	config.P2P.ListenAddress = fmt.Sprintf("tcp://localhost:%d", GetNextPortAvailable())

	randomStateFilePath := fmt.Sprintf("/tmp/%s-tendermint-state-file-%s.tmp.json", "dymd", uuid.New().String())
	tempFiles = append(tempFiles, randomStateFilePath)
	pv := privval.NewFilePV(validatorPrivKey, "", randomStateFilePath)
	pApp := proxy.NewLocalClientCreator(app)
	nodeKey := &p2p.NodeKey{
		PrivKey: pv.Key.PrivKey,
	}

	var genesisProvider nm.GenesisDocProvider = func() (*tmtypes.GenesisDoc, error) {
		return genesis, nil
	}

	node, err := nm.NewNode(
		config,          // config
		pv,              // private validator
		nodeKey,         // node key
		pApp,            // client creator
		genesisProvider, // genesis doc provider
		func(_ *nm.DBContext) (cdb.DB, error) { // db provider
			return db, nil
		},
		nm.DefaultMetricsProvider(config.Instrumentation), // metrics provider
		logger, // logger
	)
	if err != nil {
		panic(err)
	}
	err = node.Start()
	if err != nil {
		panic(err)
	}

	waitForRPC := func() {
		client, err := tmrpcclient.New(config.RPC.ListenAddress)
		if err != nil {
			panic(err)
		}
		result := new(tmcorestypes.ResultStatus)
		for {
			_, err := client.Call(context.Background(), "status", map[string]interface{}{}, result)
			if err == nil {
				return
			}

			fmt.Println("error", err)
			time.Sleep(time.Millisecond)
		}
	}
	waitForGRPC := func() {
		client := tmgrpc.StartGRPCClient(config.RPC.GRPCListenAddress)
		for {
			_, err := client.Ping(context.Background(), &tmgrpc.RequestPing{})
			if err == nil {
				return
			}
		}
	}
	if useRpc {
		waitForRPC()
	}
	if useGrpc {
		waitForGRPC()
	}

	return node, portRpc, tempFiles
}
