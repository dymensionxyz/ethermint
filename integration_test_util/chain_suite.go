package integration_test_util

//goland:noinspection SpellCheckingInspection,GoSnakeCaseUsage
import (
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
	"fmt"
	sdkdb "github.com/cosmos/cosmos-db"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"math"
	"math/big"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"cosmossdk.io/log"
	"cosmossdk.io/simapp/params"
	cmtdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmversion "github.com/cometbft/cometbft/proto/tendermint/version"
	httpclient "github.com/cometbft/cometbft/rpc/client/http"
	cmtjrpcclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	tmstate "github.com/cometbft/cometbft/state"
	cmtstore "github.com/cometbft/cometbft/store"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cometbft/cometbft/version"
	"github.com/cosmos/cosmos-sdk/baseapp"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmostxtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govlegacytypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethermint_hd "github.com/evmos/ethermint/crypto/hd"
	"github.com/evmos/ethermint/encoding"
	kvindexer "github.com/evmos/ethermint/indexer"
	itutiltypes "github.com/evmos/ethermint/integration_test_util/types"
	rpcbackend "github.com/evmos/ethermint/rpc/backend"
	rpctypes "github.com/evmos/ethermint/rpc/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"
	"github.com/stretchr/testify/require"
)

// ChainIntegrationTestSuite is a helper for Chain integration test.
type ChainIntegrationTestSuite struct {
	t                    *testing.T
	require              *require.Assertions
	muTest               sync.RWMutex
	mu                   sync.RWMutex
	ibcSuite             *ChainsIbcIntegrationTestSuite
	historicalContext    map[int64]sdk.Context
	useKeyring           bool
	tempHolder           *itutiltypes.TemporaryHolder
	logger               log.Logger
	EncodingConfig       params.EncodingConfig
	ChainConstantsConfig itutiltypes.ChainConstantConfig
	DB                   *cmtdb.MemDB
	CometBFTApp          itutiltypes.CometBftApp
	ChainApp             itutiltypes.ChainApp
	ValidatorSet         *cmttypes.ValidatorSet
	CurrentContext       sdk.Context
	ValidatorAccounts    itutiltypes.TestAccounts
	WalletAccounts       itutiltypes.TestAccounts
	ModuleAccounts       map[string]authtypes.ModuleAccountI
	QueryClients         *itutiltypes.QueryClients
	EvmTxIndexer         *kvindexer.KVIndexer
	RpcBackend           *rpcbackend.Backend
	EthSigner            ethtypes.Signer
	TestConfig           itutiltypes.TestConfig
}

// CreateChainIntegrationTestSuite initialize an integration test suite using default configuration.
func CreateChainIntegrationTestSuite(t *testing.T, r *require.Assertions) *ChainIntegrationTestSuite {
	return CreateChainIntegrationTestSuiteFromChainConfig(t, r, IntegrationTestChain1, false)
}

//goland:noinspection SpellCheckingInspection
var IntegrationTestChain1 = itutiltypes.ChainConfig{
	CosmosChainId:            "ethermint_9000-1",
	BaseDenom:                "udym",
	Bech32Prefix:             "dym",
	EvmChainId:               9000,
	DisabledContractCreation: false,
}

//goland:noinspection SpellCheckingInspection
var IntegrationTestChain2 = itutiltypes.ChainConfig{
	CosmosChainId:            "ethermint_9001-1",
	BaseDenom:                "udym",
	Bech32Prefix:             "dym",
	EvmChainId:               9001,
	DisabledContractCreation: false,
}

// CreateChainIntegrationTestSuiteFromChainConfig initialize an integration test suite from a given chain config.
func CreateChainIntegrationTestSuiteFromChainConfig(t *testing.T, r *require.Assertions, chainCfg itutiltypes.ChainConfig, disableCometBFT bool) *ChainIntegrationTestSuite {
	//goland:noinspection SpellCheckingInspection
	const balancePerAccount = 2

	chainCfg.EvmChainIdBigInt = big.NewInt(chainCfg.EvmChainId)

	testEncodingCfg := encoding.MakeConfig()

	appEncodingCfg := params.EncodingConfig{
		InterfaceRegistry: testEncodingCfg.InterfaceRegistry,
		Codec:             testEncodingCfg.Codec,
		TxConfig:          testEncodingCfg.TxConfig,
		Amino:             testEncodingCfg.Amino,
	}

	//goland:noinspection SpellCheckingInspection
	testConfig := itutiltypes.TestConfig{
		SecondaryDenomUnits: []banktypes.DenomUnit{
			{
				Denom:    "utwo",
				Exponent: 6,
			},
			{
				Denom:    "uthree",
				Exponent: 8,
			},
		},
		InitBalanceAmount:        sdkmath.NewInt(int64(balancePerAccount * math.Pow10(18))),
		DefaultFeeAmount:         sdkmath.NewInt(int64(math.Pow10(16))),
		DisableCometBFT:          disableCometBFT,
		DisabledContractCreation: chainCfg.DisabledContractCreation,
	}

	tempHolder := itutiltypes.NewTemporaryHolder()

	// Setup assertions
	if r == nil {
		r = require.New(t)
	}

	// Setup Test accounts

	validatorAccounts := newValidatorAccounts(t)
	if disableCometBFT {
		// no-op
	} else {
		// test CometBFT use only one validator
		validatorAccounts = []*itutiltypes.TestAccount{validatorAccounts.Number(1)}
	}

	walletAccounts := newWalletsAccounts(t)

	// Init database
	cmtDB := cmtdb.NewMemDB()
	evmIndexerDb := sdkdb.NewMemDB() // use dedicated db for EVM Tx-Indexer to prevent data corruption

	// Setup chain app
	genesisAccountBalance := sdk.NewCoins(
		sdk.NewCoin(chainCfg.BaseDenom, testConfig.InitBalanceAmount),
	)
	for _, secondaryDenomUnit := range testConfig.SecondaryDenomUnits {
		genesisAccountBalance = genesisAccountBalance.Add(
			sdk.NewCoin(secondaryDenomUnit.Denom, testConfig.InitBalanceAmount),
		)
	}

	logger := log.NewNopLogger()
	app, tmApp, valSet := itutiltypes.NewChainApp(chainCfg, disableCometBFT, testConfig, appEncodingCfg, cmtDB, validatorAccounts, walletAccounts, genesisAccountBalance, tempHolder, logger)

	oga := app.EthermintApp()
	oga.BasicModuleManager.RegisterInterfaces(appEncodingCfg.InterfaceRegistry)
	oga.BasicModuleManager.RegisterLegacyAminoCodec(appEncodingCfg.Amino)

	baseApp := app.BaseApp()

	header := createFirstBlockHeader(
		chainCfg.CosmosChainId,
		validatorAccounts.Number(1).GetConsensusAddress(),
	)
	ctx := baseApp.NewContext(false).
		WithBlockHeader(header).
		WithChainID(chainCfg.CosmosChainId)

	evmParams := app.EvmKeeper().GetParams(ctx)
	evmParams.EvmDenom = chainCfg.BaseDenom
	err := app.EvmKeeper().SetParams(ctx, evmParams)
	require.NoError(t, err)

	// Setup validators
	for _, validatorAccount := range validatorAccounts {
		valAddrCodec := app.StakingKeeper().ValidatorAddressCodec()

		valAddr := validatorAccount.GetValidatorAddress()
		operator, err := valAddrCodec.BytesToString(valAddr)
		require.NoError(t, err)

		val, err := stakingtypes.NewValidator(
			operator,
			validatorAccount.GetSdkPubKey(),
			stakingtypes.Description{},
		)
		require.NoError(t, err)

		val = stakingkeeper.TestingUpdateValidator(app.StakingKeeper(), ctx, val, true)

		refKeeper := reflect.ValueOf(app.StakingKeeper())
		if refKeeper.Kind() == reflect.Ptr {
			refKeeper = reflect.Indirect(refKeeper.Elem())
		}

		err = app.DistributionKeeper().Hooks().AfterValidatorCreated(ctx, valAddr)

		err = app.StakingKeeper().SetValidatorByConsAddr(ctx, val)
		require.NoError(t, err)
	}

	clientCtx := cosmosclient.Context{}.
		WithChainID(chainCfg.CosmosChainId).
		WithCodec(appEncodingCfg.Codec).
		WithInterfaceRegistry(appEncodingCfg.InterfaceRegistry).
		WithTxConfig(appEncodingCfg.TxConfig).
		WithLegacyAmino(appEncodingCfg.Amino).
		WithKeyringOptions(ethermint_hd.EthSecp256k1Option())

	result := &ChainIntegrationTestSuite{
		t:                 t,
		require:           r,
		muTest:            sync.RWMutex{},
		mu:                sync.RWMutex{},
		historicalContext: make(map[int64]sdk.Context),
		tempHolder:        tempHolder,
		logger:            logger,
		EncodingConfig:    appEncodingCfg,
		ChainConstantsConfig: itutiltypes.NewChainConstantConfig(
			chainCfg.CosmosChainId,
			chainCfg.BaseDenom,
		),
		DB:                cmtDB,
		ChainApp:          app,
		CometBFTApp:       tmApp,
		ValidatorSet:      valSet,
		CurrentContext:    ctx,
		ValidatorAccounts: validatorAccounts,
		WalletAccounts:    walletAccounts,
		ModuleAccounts:    make(map[string]authtypes.ModuleAccountI),
		EvmTxIndexer:      kvindexer.NewKVIndexer(evmIndexerDb, log.NewNopLogger(), clientCtx),
		EthSigner:         ethtypes.LatestSignerForChainID(chainCfg.EvmChainIdBigInt),
		TestConfig:        testConfig,
	}

	if disableCometBFT {
		result.Commit() // Commit the initial block
	} else {
		time.Sleep(300 * time.Millisecond)
		result.Commit()
	}

	result.CreateAllQueryClientsAndRpcBackend()

	accounts, _ := result.QueryClients.Auth.ModuleAccounts(nil, &authtypes.QueryModuleAccountsRequest{})
	for _, acc := range accounts.Accounts {
		var account sdk.AccountI
		err = appEncodingCfg.InterfaceRegistry.UnpackAny(acc, &account)
		require.NoError(t, err)
		moduleAccount, ok := account.(sdk.ModuleAccountI)
		require.True(t, ok)
		result.ModuleAccounts[moduleAccount.GetName()] = moduleAccount
	}

	return result
}

func (suite *ChainIntegrationTestSuite) T() *testing.T {
	suite.muTest.RLock()
	defer suite.muTest.RUnlock()
	return suite.t
}

func (suite *ChainIntegrationTestSuite) Require() *require.Assertions {
	suite.muTest.RLock()
	defer suite.muTest.RUnlock()
	return suite.require
}

func (suite *ChainIntegrationTestSuite) UseKeyring() {
	suite.muTest.Lock()
	defer suite.muTest.Unlock()
	suite.useKeyring = true
}

// Cleanup cleans up the ChainIntegrationTestSuite.
// This method should be called after each test or suite, depends on the tactic you shut down the Integration chain.
func (suite *ChainIntegrationTestSuite) Cleanup() {
	if suite == nil {
		return
	}

	if suite.HasCometBFT() {
		suite.CometBFTApp.Shutdown()
	}

	if suite.tempHolder != nil {
		if tempFiles, anyTemp := suite.tempHolder.GetTempFiles(); anyTemp {
			for _, file := range tempFiles {
				err := os.RemoveAll(file)
				if err != nil {
					fmt.Println("Failed to remove temp file", file)
					fmt.Println(err)
				}
			}
		}
	}
}

// BaseApp returns the BaseApp instance of the Integrated chain.
func (suite *ChainIntegrationTestSuite) BaseApp() *baseapp.BaseApp {
	return suite.ChainApp.BaseApp()
}

// CreateAllQueryClientsAndRpcBackend creates all query clients and RPC backend instance at recent block height.
// This method should be called after each commit to refresh the query clients.
func (suite *ChainIntegrationTestSuite) CreateAllQueryClientsAndRpcBackend() {
	suite.QueryClients = suite.QueryClientsAt(0)
	suite.RpcBackend = suite.RpcBackendAt(0)
}

// ContextAt returns the context at a given context block height.
func (suite *ChainIntegrationTestSuite) ContextAt(height int64) sdk.Context {
	if height == 0 {
		height = suite.GetLatestBlockHeight()
	}

	if ctx, found := suite.historicalContext[height]; found {
		return ctx
	}

	qCtx, err := suite.createAppQueryContext(height, false)
	suite.Require().NoError(err)

	return qCtx
}

// createAppQueryContext returns the query context at a given context block height.
// Used as a helper method to create query context to adapt with older version of Cosmos-SDK BaseApp,
// which does not expose CreateQueryContext method.
func (suite *ChainIntegrationTestSuite) createAppQueryContext(height int64, prove bool) (sdk.Context, error) {
	return suite.BaseApp().CreateQueryContext(height, prove)
}

// QueryClientsAt returns the list of query client instance that connects to store data at a given context block height.
func (suite *ChainIntegrationTestSuite) QueryClientsAt(height int64) *itutiltypes.QueryClients {
	var sdkContext sdk.Context
	if suite.HasCometBFT() {
		if height == 0 {
			height = suite.GetLatestBlockHeight()
		}
		sdkContext = suite.CurrentContext
		if height > 0 {
			var err error
			sdkContext, err = suite.createAppQueryContext(height, false)
			suite.Require().NoError(err)
		}
	} else if height == 0 || height == suite.GetLatestBlockHeight() {
		// latest block
		sdkContext = suite.CurrentContext
	} else {
		var err error
		sdkContext, err = suite.createAppQueryContext(height, false)
		suite.Require().NoError(err)
	}

	queryHelper := NewQueryServerTestHelper(sdkContext, suite.ChainApp.InterfaceRegistry())

	authtypes.RegisterQueryServer(queryHelper, authkeeper.NewQueryServer(*suite.ChainApp.AccountKeeper()))
	authQueryClient := authtypes.NewQueryClient(queryHelper)

	banktypes.RegisterQueryServer(queryHelper, suite.ChainApp.BankKeeper())
	bankQueryClient := banktypes.NewQueryClient(queryHelper)

	distributiontypes.RegisterQueryServer(queryHelper, distributionkeeper.NewQuerier(suite.ChainApp.DistributionKeeper()))
	distributionQueryClient := distributiontypes.NewQueryClient(queryHelper)

	evmtypes.RegisterQueryServer(queryHelper, suite.ChainApp.EvmKeeper())
	evmQueryClient := evmtypes.NewQueryClient(queryHelper)

	feemarkettypes.RegisterQueryServer(queryHelper, suite.ChainApp.FeeMarketKeeper())
	feeMarketQueryClient := feemarkettypes.NewQueryClient(queryHelper)

	govv1types.RegisterQueryServer(queryHelper, govkeeper.NewQueryServer(suite.ChainApp.GovKeeper()))
	govV1QueryClient := govv1types.NewQueryClient(queryHelper)

	govlegacytypes.RegisterQueryServer(queryHelper, govkeeper.NewLegacyQueryServer(suite.ChainApp.GovKeeper()))
	govLegacyQueryClient := govlegacytypes.NewQueryClient(queryHelper)

	ibctransfertypes.RegisterQueryServer(queryHelper, suite.ChainApp.IbcTransferKeeper())
	ibcTransferQueryClient := ibctransfertypes.NewQueryClient(queryHelper)

	slashingtypes.RegisterQueryServer(queryHelper, suite.ChainApp.SlashingKeeper())
	slashingQueryClient := slashingtypes.NewQueryClient(queryHelper)

	stakingtypes.RegisterQueryServer(queryHelper, stakingkeeper.Querier{Keeper: suite.ChainApp.StakingKeeper()})
	stakingQueryClient := stakingtypes.NewQueryClient(queryHelper)

	serviceClient := cosmostxtypes.NewServiceClient(queryHelper)

	rpcQueryClient := rpctypes.QueryClient{
		ServiceClient: serviceClient,
		QueryClient:   evmQueryClient,
		FeeMarket:     feeMarketQueryClient,
	}

	var cometRpcHttpClient *httpclient.HTTP
	if suite.HasCometBFT() {
		rpcAddr26657, supported := suite.CometBFTApp.GetRpcAddr()
		suite.Require().True(supported)

		httpClient26657, err := cmtjrpcclient.DefaultHTTPClient(rpcAddr26657)
		suite.Require().NoError(err)

		cometRpcHttpClient, err = httpclient.NewWithClient(rpcAddr26657, "/websocket", httpClient26657)
		suite.Require().NoError(err)

		err = cometRpcHttpClient.Start()
		suite.Require().NoError(err)
	}

	clientQueryCtx := cosmosclient.Context{}.
		WithChainID(suite.ChainConstantsConfig.GetCosmosChainID()).
		WithCodec(suite.EncodingConfig.Codec).
		WithInterfaceRegistry(suite.EncodingConfig.InterfaceRegistry).
		WithTxConfig(suite.EncodingConfig.TxConfig).
		WithLegacyAmino(suite.EncodingConfig.Amino).
		WithKeyringOptions(ethermint_hd.EthSecp256k1Option()).
		WithAccountRetriever(authtypes.AccountRetriever{})

	if suite.useKeyring {
		clientQueryCtx = clientQueryCtx.WithKeyring(itutiltypes.NewIntegrationTestKeyring(suite.WalletAccounts))
	} else {
		clientQueryCtx = clientQueryCtx.WithKeyring(itutiltypes.NewIntegrationTestKeyring(nil))
	}

	if height > 0 {
		clientQueryCtx = clientQueryCtx.WithHeight(height)
	}

	if suite.HasCometBFT() {
		clientQueryCtx = clientQueryCtx.WithClient(cometRpcHttpClient)
	} else {
		// Due to RPC backend uses CometBFT HTTP client, but in IBC we don't use CometBFT,
		// so as a workaround we set a nil client to avoid crash.
		clientQueryCtx = clientQueryCtx.WithClient((*httpclient.HTTP)(nil))
	}

	cosmostxtypes.RegisterServiceServer(
		queryHelper,
		authtx.NewTxServer(clientQueryCtx, suite.BaseApp().Simulate, suite.ChainApp.InterfaceRegistry()),
	)

	return &itutiltypes.QueryClients{
		GrpcConnection:        queryHelper,
		ClientQueryCtx:        clientQueryCtx,
		CometBFTRpcHttpClient: cometRpcHttpClient,
		Auth:                  authQueryClient,
		Bank:                  bankQueryClient,
		Distribution:          distributionQueryClient,
		EVM:                   evmQueryClient,
		GovV1:                 govV1QueryClient,
		GovLegacy:             govLegacyQueryClient,
		IbcTransfer:           ibcTransferQueryClient,
		Slashing:              slashingQueryClient,
		Staking:               stakingQueryClient,
		ServiceClient:         serviceClient,
		Rpc:                   &rpcQueryClient,
	}
}

// RpcBackendAt returns the RPC-backend instance at a given context block height.
func (suite *ChainIntegrationTestSuite) RpcBackendAt(height int64) *rpcbackend.Backend {
	queryClients := suite.QueryClientsAt(height)
	rpcServerCtx := server.NewDefaultContext()
	rpcServerCtx.Viper.Set("json-rpc.gas-cap", 999_999_999)

	rpcBackend := rpcbackend.NewBackend(rpcServerCtx, rpcServerCtx.Logger, queryClients.ClientQueryCtx, false, suite.EvmTxIndexer)

	// override the query client with the mock query client, for changing query context
	getFieldQueryClient := func() reflect.Value {
		return reflect.Indirect(reflect.ValueOf(rpcBackend).Elem()).FieldByName("queryClient")
	}
	fieldQueryClient := getFieldQueryClient()
	reflect.NewAt(fieldQueryClient.Type(), unsafe.Pointer(fieldQueryClient.UnsafeAddr())).
		Elem().
		Set(reflect.ValueOf(queryClients.Rpc))

	return rpcBackend
}

// GetLatestBlockHeight returns the most recent block height.
func (suite *ChainIntegrationTestSuite) GetLatestBlockHeight() int64 {
	if suite.HasCometBFT() {
		// because CometBFT auto-commit blocks so the CurrentContext property might out-dated
		return suite.BaseApp().LastBlockHeight()
	}

	return suite.CurrentContext.BlockHeight()
}

// WaitNextBlockOrCommit returns the most recent block height beside the following logic:
//
// - When CometBFT is Enabled, it waits for the next block to be committed before returning result.
//
// - When CometBFT is Disabled, it triggers commit block and starts a new block with an updated context.
//
// USE-CASE for this: you want to submit one or multiple txs and have sometime to know the executed block,
// while CometBFT auto commit blocks.
func (suite *ChainIntegrationTestSuite) WaitNextBlockOrCommit() int64 {
	if !suite.HasCometBFT() {
		suite.Commit()
		return suite.GetLatestBlockHeight()
	}

	oldHeight := suite.GetLatestBlockHeight()
	var currentHeight int64
	for {
		currentHeight = suite.GetLatestBlockHeight()
		if currentHeight > oldHeight {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	return currentHeight
}

// Commit commits and starts a new block with an updated context.
func (suite *ChainIntegrationTestSuite) Commit() {
	if suite.ibcSuite != nil { // ibc-connected chains must be committed together
		suite.ibcSuite.CommitAllChains()
	} else {
		suite.commitAndBeginBlockAfter(1 * time.Hour)
	}
}

// ibcSuiteCommit is a helper function to commit with custom block time equals to IBC setup
func (suite *ChainIntegrationTestSuite) ibcSuiteCommit() {
	suite.commitAndBeginBlockAfter(5 * time.Second)
}

// commitAndBeginBlockAfter commits a block at a given time.
func (suite *ChainIntegrationTestSuite) commitAndBeginBlockAfter(t time.Duration) {
	suite.mu.Lock()
	defer suite.mu.Unlock()

	defer func() {
		suite.CreateAllQueryClientsAndRpcBackend()
	}()

	var newCtx sdk.Context
	var newValSet *cmttypes.ValidatorSet

	if suite.HasCometBFT() {
		// awaiting next block generated by CometBFT
		originalHeight := suite.GetLatestBlockHeight()
		var latestHeight int64
		for {
			time.Sleep(10 * time.Millisecond)
			latestHeight = suite.GetLatestBlockHeight()
			if latestHeight > originalHeight {
				break
			}
		}

		blockStore, stateStore := suite.GetBlockStoreAndStateStore()

		tmBlk := blockStore.LoadBlock(latestHeight)
		valSet, err := stateStore.LoadValidators(latestHeight)
		suite.Require().NoErrorf(err, "failed to load validator set for block %d", latestHeight)

		header := tmBlk.Header.ToProto()
		ctx := suite.createNewContext(suite.CurrentContext, *header)
		suite.triggerEvmIndexer(latestHeight, blockStore, stateStore) // trigger EVM Tx-Indexer indexing data to latest

		newCtx = ctx
		newValSet = valSet
	} else {
		// manually commit block and move to next
		backupContext := suite.CurrentContext

		nextCtx, nextValSet, err := suite.commitAndCreateNewCtx(suite.CurrentContext, t, suite.ValidatorSet)
		suite.Require().NoError(err)
		suite.Require().Equalf(suite.CurrentContext.BlockHeight()+1, nextCtx.BlockHeight(), "next block height must be increased by 1")

		suite.historicalContext[backupContext.BlockHeight()] = backupContext

		newCtx = nextCtx
		newValSet = nextValSet
	}

	suite.CurrentContext = newCtx
	suite.ValidatorSet = newValSet
}

// GetIbcTimeoutHeight returns a timeout height for IBC packet, based on recent block, plus the offset.
func (suite *ChainIntegrationTestSuite) GetIbcTimeoutHeight(offsetHeight int64) ibcclienttypes.Height {
	chainId := suite.ChainConstantsConfig.GetCosmosChainID()
	idx := strings.LastIndex(chainId, "-")
	rev := chainId[idx+1:]
	revInt, err := strconv.ParseUint(rev, 10, 64)
	suite.Require().NoError(err)
	return ibcclienttypes.NewHeight(revInt, uint64(suite.GetLatestBlockHeight()+offsetHeight))
}

// triggerEvmIndexer indexes EVM txs from blockStore and stateStore, upto latestHeight.
func (suite *ChainIntegrationTestSuite) triggerEvmIndexer(latestHeight int64, blockStore *cmtstore.BlockStore, stateStore tmstate.Store) {
	suite.Require().NotZero(latestHeight)
	suite.Require().NotNil(blockStore)
	suite.Require().NotNil(stateStore)

	lastIndexedHeight, err := suite.EvmTxIndexer.LastIndexedBlock()
	suite.Require().NoError(err)

	if lastIndexedHeight >= latestHeight {
		return
	}

	if lastIndexedHeight < 0 {
		lastIndexedHeight = 0
	}

	var ch int64
	for ch = lastIndexedHeight + 1; ch <= latestHeight; ch++ {
		tmBlk := blockStore.LoadBlock(ch)
		tmAbciResponse, err := stateStore.LoadFinalizeBlockResponse(ch)
		suite.Require().NoErrorf(err, "failed to load abci response for block %d", ch)
		err = suite.EvmTxIndexer.IndexBlock(tmBlk, tmAbciResponse.TxResults)
		suite.Require().NoErrorf(err, "failed to index block %d", ch)
	}
}

// GetBlockStoreAndStateStore returns blockStore and stateStore if CometBFT is Enabled.
//
// WARN: if CometBFT is Disabled, the call will panic.
func (suite *ChainIntegrationTestSuite) GetBlockStoreAndStateStore() (*cmtstore.BlockStore, tmstate.Store) {
	suite.EnsureCometBFT()
	blockStore := cmtstore.NewBlockStore(suite.DB)
	stateStore := tmstate.NewStore(suite.DB, tmstate.StoreOptions{
		DiscardABCIResponses: false,
	})
	return blockStore, stateStore
}

// createFirstBlockHeader creates a new CometBFT header, with context 1, for testing purposes.
func createFirstBlockHeader(
	chainID string,
	proposer sdk.ConsAddress,
) tmproto.Header {
	//goland:noinspection SpellCheckingInspection
	return tmproto.Header{
		ChainID:         chainID,
		Height:          1,
		Time:            time.Now().UTC(),
		ValidatorsHash:  nil,
		AppHash:         nil,
		ProposerAddress: proposer.Bytes(),
		Version: tmversion.Consensus{
			Block: version.BlockProtocol,
		},
		LastBlockId: tmproto.BlockID{
			Hash: tmhash.Sum([]byte("block_id")),
			PartSetHeader: tmproto.PartSetHeader{
				Total: 11,
				Hash:  tmhash.Sum([]byte("partset_header")),
			},
		},
		DataHash:           tmhash.Sum([]byte("data")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators")),
		ConsensusHash:      tmhash.Sum([]byte("consensus")),
		LastResultsHash:    tmhash.Sum([]byte("last_result")),
		EvidenceHash:       tmhash.Sum([]byte("evidence")),
	}
}

// ReflectChangesToCommitMultiStore commit the current state directly into the base app's commit multistore.
// Since Cosmos-SDK v0.50, the block execution context is maintained internally,
// that make Commit can not pass context to finalize.
func (suite *ChainIntegrationTestSuite) ReflectChangesToCommitMultiStore() {
	ms := suite.CurrentContext.MultiStore()
	if rms, ok := ms.(*rootmulti.Store); ok {
		suite.CurrentContext = suite.CurrentContext.WithMultiStore(rms.CacheMultiStore())
	} else if cms, ok := ms.(storetypes.CommitMultiStore); ok {
		suite.CurrentContext = suite.CurrentContext.WithMultiStore(cms.CacheMultiStore())
	} else if _, ok := suite.CurrentContext.MultiStore().(storetypes.CacheMultiStore); ok {
		// ok
	} else {
		panic(fmt.Sprintf("not supported multistore type %T", ms))
	}

	// write to commit multi store
	suite.CurrentContext.MultiStore().(storetypes.CacheMultiStore).Write()

	// reflect new change to current context
	suite.CurrentContext = suite.CurrentContext.WithMultiStore(suite.BaseApp().CommitMultiStore())
}
