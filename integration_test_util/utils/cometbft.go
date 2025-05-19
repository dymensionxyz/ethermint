package utils

//goland:noinspection SpellCheckingInspection
import (
	"context"
	"cosmossdk.io/log"
	"fmt"
	cdb "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtcfg "github.com/cometbft/cometbft/config"
	tmcrypto "github.com/cometbft/cometbft/crypto"
	cmtnode "github.com/cometbft/cometbft/node"
	cmtp2p "github.com/cometbft/cometbft/p2p"
	cmtprivval "github.com/cometbft/cometbft/privval"
	cmtproxy "github.com/cometbft/cometbft/proxy"
	cmtrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmtgrpc "github.com/cometbft/cometbft/rpc/grpc"
	cmtjrpcclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	cmtrpctest "github.com/cometbft/cometbft/rpc/test"
	cmttypes "github.com/cometbft/cometbft/types"
	servercmtlog "github.com/cosmos/cosmos-sdk/server/log"
	"github.com/google/uuid"
	"time"
)

// StartCometBFTNode starts a CometBFT node for the given ABCI Application, used for testing purposes.
func StartCometBFTNode(app abci.Application, genesis *cmttypes.GenesisDoc, db cdb.DB, validatorPrivKey tmcrypto.PrivKey, logger log.Logger) (cometNode *cmtnode.Node, rpcPort int, tempFiles []string) {
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
	config := cmtrpctest.GetConfig(false)

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

	randomStateFilePath := fmt.Sprintf("/tmp/%s-cometbft-state-file-%s.tmp.json", "dymd", uuid.New().String())
	tempFiles = append(tempFiles, randomStateFilePath)
	pv := cmtprivval.NewFilePV(validatorPrivKey, "", randomStateFilePath)
	pApp := cmtproxy.NewLocalClientCreator(app)
	nodeKey := &cmtp2p.NodeKey{
		PrivKey: pv.Key.PrivKey,
	}

	var genesisProvider cmtnode.GenesisDocProvider = func() (*cmttypes.GenesisDoc, error) {
		return genesis, nil
	}

	node, err := cmtnode.NewNode(
		config,          // config
		pv,              // private validator
		nodeKey,         // node key
		pApp,            // client creator
		genesisProvider, // genesis doc provider
		func(_ *cmtcfg.DBContext) (cdb.DB, error) { // db provider
			return db, nil
		},
		cmtnode.DefaultMetricsProvider(config.Instrumentation),                 // metrics provider
		servercmtlog.CometLoggerWrapper{Logger: logger.With("server", "node")}, // logger
	)
	if err != nil {
		panic(err)
	}
	err = node.Start()
	if err != nil {
		panic(err)
	}

	waitForRPC := func() {
		client, err := cmtjrpcclient.New(config.RPC.ListenAddress)
		if err != nil {
			panic(err)
		}
		result := new(cmtrpctypes.ResultStatus)
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
		client := cmtgrpc.StartGRPCClient(config.RPC.GRPCListenAddress)
		for {
			_, err := client.Ping(context.Background(), &cmtgrpc.RequestPing{})
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
