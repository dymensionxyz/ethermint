package keeper_test

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	"github.com/evmos/ethermint/tests"
	"github.com/evmos/ethermint/testutil"
	"github.com/evmos/ethermint/x/feemarket/types"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

var _ = Describe("Feemarket", func() {
	var privKey *ethsecp256k1.PrivKey

	Describe("Performing EVM transactions", func() {
		type txParams struct {
			gasLimit  uint64
			gasPrice  *big.Int
			gasFeeCap *big.Int
			gasTipCap *big.Int
			accesses  *ethtypes.AccessList
		}
		type getprices func() txParams

		Context("with MinGasPrices (feemarket param) < BaseFee (feemarket)", func() {
			var (
				baseFee      int64
				minGasPrices int64
				err          error
			)

			BeforeEach(func() {
				baseFee = 10_000_000_000
				minGasPrices = baseFee - 5_000_000_000

				// Note that the tests run the same transactions with `gasLimit =
				// 100_000`. With the fee calculation `Fee = (baseFee + tip) * gasLimit`,
				// a `minGasPrices = 5_000_000_000` results in `minGlobalFee =
				// 500_000_000_000_000`
				privKey, _, err = setupTestWithContext("1", math.LegacyNewDec(minGasPrices), sdkmath.NewInt(baseFee))
				Expect(err).To(BeNil())
			})

			Context("during CheckTx", func() {
				DescribeTable("should accept transactions with gas Limit > 0",
					func(malleate getprices) {
						p := malleate()
						to := tests.GenerateAddress()
						msgEthereumTx := buildEthTx(privKey, &to, p.gasLimit, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						res, err := testutil.CheckEthTx(s.ctx, s.app, privKey, msgEthereumTx)
						Expect(err).To(BeNil())
						Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
					},
					Entry("legacy tx", func() txParams {
						return txParams{100000, big.NewInt(baseFee), nil, nil, nil}
					}),
					Entry("dynamic tx", func() txParams {
						return txParams{100000, nil, big.NewInt(baseFee), big.NewInt(0), &ethtypes.AccessList{}}
					}),
				)
				DescribeTable("should not accept transactions with gas Limit > 0",
					func(malleate getprices) {
						p := malleate()
						to := tests.GenerateAddress()
						msgEthereumTx := buildEthTx(privKey, &to, p.gasLimit, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						res, err := testutil.CheckEthTx(s.ctx, s.app, privKey, msgEthereumTx)
						Expect(err).To(BeNil())
						Expect(res.IsOK()).To(Equal(false), "transaction should have succeeded", res.GetLog())
					},
					Entry("legacy tx", func() txParams {
						return txParams{0, big.NewInt(baseFee), nil, nil, nil}
					}),
					Entry("dynamic tx", func() txParams {
						return txParams{0, nil, big.NewInt(baseFee), big.NewInt(0), &ethtypes.AccessList{}}
					}),
				)
			})

			Context("during DeliverTx", func() {
				DescribeTable("should accept transactions with gas Limit > 0",
					func(malleate getprices) {
						p := malleate()
						to := tests.GenerateAddress()
						msgEthereumTx := buildEthTx(privKey, &to, p.gasLimit, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						res, err := testutil.DeliverEthTx(s.ctx, s.app, privKey, msgEthereumTx)
						Expect(err).To(BeNil())
						Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
					},
					Entry("legacy tx", func() txParams {
						return txParams{100000, big.NewInt(baseFee), nil, nil, nil}
					}),
					Entry("dynamic tx", func() txParams {
						return txParams{100000, nil, big.NewInt(baseFee), big.NewInt(0), &ethtypes.AccessList{}}
					}),
				)
				DescribeTable("should not accept transactions with gas Limit > 0",
					func(malleate getprices) {
						p := malleate()
						to := tests.GenerateAddress()
						msgEthereumTx := buildEthTx(privKey, &to, p.gasLimit, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						res, err := testutil.CheckEthTx(s.ctx, s.app, privKey, msgEthereumTx)
						Expect(err).To(BeNil())
						Expect(res.IsOK()).To(Equal(false), "transaction should have succeeded", res.GetLog())
					},
					Entry("legacy tx", func() txParams {
						return txParams{0, big.NewInt(baseFee), nil, nil, nil}
					}),
					Entry("dynamic tx", func() txParams {
						return txParams{0, nil, big.NewInt(baseFee), big.NewInt(0), &ethtypes.AccessList{}}
					}),
				)
			})
		})
	})
})

// setupTestWithContext sets up a test chain with an example Cosmos send msg,
// given a local (validator config) and a gloabl (feemarket param) minGasPrice
func setupTestWithContext(valMinGasPrice string, minGasPrice math.LegacyDec, baseFee math.Int) (*ethsecp256k1.PrivKey, banktypes.MsgSend, error) {
	s = new(KeeperTestSuite)
	s.SetupTest()

	valMinGasPriceInt, ok := math.NewIntFromString(valMinGasPrice)
	if !ok {
		return nil, banktypes.MsgSend{}, fmt.Errorf("invalid minGasPrice: %s", valMinGasPrice)
	}
	s.ctx = s.ctx.WithMinGasPrices(sdk.DecCoins{sdk.NewDecCoin(s.denom, valMinGasPriceInt)})

	params := types.DefaultParams()
	params.MinGasPrice = minGasPrice
	s.app.FeeMarketKeeper.SetParams(s.ctx, params)
	s.app.FeeMarketKeeper.SetBaseFee(s.ctx, baseFee.BigInt())

	privKey, address := generateKey()
	amount, ok := math.NewIntFromString("10000000000000000000")
	s.Require().True(ok)
	initBalance := sdk.Coins{sdk.Coin{
		Denom:  s.denom,
		Amount: amount,
	}}
	testutil.FundAccount(s.app.BankKeeper, s.ctx, address, initBalance)

	msg := banktypes.MsgSend{
		FromAddress: address.String(),
		ToAddress:   address.String(),
		Amount: sdk.Coins{sdk.Coin{
			Denom:  s.denom,
			Amount: sdkmath.NewInt(10000),
		}},
	}
	s.Commit()
	return privKey, msg, nil
}

func generateKey() (*ethsecp256k1.PrivKey, sdk.AccAddress) {
	address, priv := tests.NewAddrKey()
	return priv, sdk.AccAddress(address.Bytes())
}

func getNonce(addressBytes []byte) uint64 {
	return s.app.EvmKeeper.GetNonce(
		s.ctx,
		common.BytesToAddress(addressBytes),
	)
}

func buildEthTx(
	priv *ethsecp256k1.PrivKey,
	to *common.Address,
	gasLimit uint64,
	gasPrice *big.Int,
	gasFeeCap *big.Int,
	gasTipCap *big.Int,
	accesses *ethtypes.AccessList,
) *evmtypes.MsgEthereumTx {
	chainID := s.app.EvmKeeper.ChainID()
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	nonce := getNonce(from.Bytes())
	data := make([]byte, 0)
	msgEthereumTx := evmtypes.NewTx(
		chainID,
		nonce,
		to,
		nil,
		gasLimit,
		gasPrice,
		gasFeeCap,
		gasTipCap,
		data,
		accesses,
	)
	msgEthereumTx.From = from.String()
	return msgEthereumTx
}
