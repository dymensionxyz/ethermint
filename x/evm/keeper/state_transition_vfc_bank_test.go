package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/evmos/ethermint/utils"
	"github.com/evmos/ethermint/x/evm/types"
	"math"
	"math/big"
)

func (suite *KeeperTestSuite) TestApplyMessageWithConfig_VFC_Ops() {
	vfbcContractAddrOfNative, found := suite.app.EvmKeeper.GetVirtualFrontierBankContractAddressByDenom(suite.ctx, suite.denom)
	suite.Require().True(found, "require setup for virtual frontier bank contract of evm native denom")

	randomVFBCSenderAddress := common.BytesToAddress([]byte{0x01, 0x01, 0x39, 0x40})
	randomVFBCReceiverAddress := common.BytesToAddress([]byte{0x02, 0x02, 0x39, 0x40})
	vfbcSenderInitialBalance := new(big.Int).SetUint64(math.MaxUint64)

	vfc := suite.app.EvmKeeper.GetVirtualFrontierContract(suite.ctx, vfbcContractAddrOfNative)
	suite.Require().NotNil(vfc)

	bankDenomMetadataOfNative, found := suite.app.BankKeeper.GetDenomMetaData(suite.ctx, suite.denom)
	suite.Require().True(found)
	vfbcDenomMetadataOfNative, valid := types.CollectMetadataForVirtualFrontierBankContract(bankDenomMetadataOfNative)
	suite.Require().True(valid)

	senderNonce := func() uint64 {
		return suite.app.EvmKeeper.GetNonce(suite.ctx, randomVFBCSenderAddress)
	}

	feeCompute := func(msg core.Message, gasUsed uint64) *big.Int {
		// tx fee was designed to be deducted in AnteHandler so this place does not cost fee
		return common.Big0
	}

	callDataName := []byte{0x06, 0xfd, 0xde, 0x03}
	callDataSymbol := []byte{0x95, 0xd8, 0x9b, 0x41}
	callDataDecimals := []byte{0x31, 0x3c, 0xe5, 0x67}
	callDataTotalSupply := []byte{0x18, 0x16, 0x0d, 0xdd}
	callDataBalanceOf := []byte{0x70, 0xa0, 0x82, 0x31}
	callDataTransferSig := []byte{0xa9, 0x05, 0x9c, 0xbb}
	callDataTransferTo := func(receiver common.Address, amount *big.Int) (ret []byte) {
		ret = append(callDataTransferSig, common.BytesToHash(receiver.Bytes()).Bytes()...)
		ret = append(ret, common.BytesToHash(amount.Bytes()).Bytes()...)
		return ret
	}
	callDataTransfer := func(amount *big.Int) (ret []byte) {
		return callDataTransferTo(randomVFBCReceiverAddress, amount)
	}

	computeIntrinsicGas := func(msg core.Message) uint64 {
		gas, err := core.IntrinsicGas(msg.Data(), msg.AccessList(), msg.To() == nil, true, true)
		if err != nil {
			panic(err)
		}
		return gas
	}

	bytesOfAbiEncodedTrue := func() []byte {
		b := make([]byte, 32)
		b[31] = 0x1
		return b
	}()

	tests := []struct {
		name                              string
		prepare                           func() core.Message
		wantExecError                     bool
		wantVmError                       bool
		wantSenderBalanceAdjustmentOrZero *big.Int
		testOnNonExecError                func(core.Message, *types.MsgEthereumTxResponse)
	}{
		{
			name: "prohibit transfer to VFBC contract (no call data)",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					nil,                       // call data
					nil,                       // access list
					false,                     // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "not allowed to receive")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg), response.GasUsed, "exec revert consume no gas except intrinsic gas")
			},
		},
		{
			name: "prohibit transfer to VFBC contract (non-zero value)",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					common.Big1,               // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataName,              // call data
					nil,                       // access list
					false,                     // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "not allowed to receive")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg), response.GasUsed, "exec revert consume no gas except intrinsic gas")
			},
		},
		{
			name: "method signature does not exists",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,    // from
					&vfbcContractAddrOfNative,  // to
					senderNonce(),              // nonce
					nil,                        // amount
					40_000,                     // gas limit
					big.NewInt(1),              // gas price
					big.NewInt(1),              // gas fee cap
					big.NewInt(1),              // gas tip cap
					[]byte{0x1, 0x2, 0x3, 0x4}, // call data
					nil,                        // access list
					false,                      // is fake
				)
			},
			wantExecError: false,
			wantVmError:   false,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed(), "execution should not fail")
				suite.Empty(response.Ret)
				suite.Equal(computeIntrinsicGas(msg), response.GasUsed, "should consume no gas except intrinsic gas")
			},
		},
		{
			name: "name() but lacking gas",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,          // from
					&vfbcContractAddrOfNative,        // to
					senderNonce(),                    // nonce
					nil,                              // amount
					params.TxGas+types.VFBCopgName-1, // gas limit
					big.NewInt(1),                    // gas price
					big.NewInt(1),                    // gas fee cap
					big.NewInt(1),                    // gas tip cap
					callDataName,                     // call data
					nil,                              // access list
					false,                            // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "out of gas")
				suite.Equal(vm.ErrOutOfGas.Error(), response.VmError)
				suite.Equal(msg.Gas(), response.GasUsed, "out of gas consume all gas")
			},
		},
		{
			name: "name() but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					append(callDataName, 0x1), // call data
					nil,                       // access list
					false,                     // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgName_Revert, response.GasUsed)
			},
		},
		{
			name: "name()",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataName,              // call data
					nil,                       // access list
					false,                     // is fake
				)
			},
			wantExecError: false,
			wantVmError:   false,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				suite.Equal(vfbcDenomMetadataOfNative.Name, utils.MustAbiDecodeString(response.Ret))
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgName, response.GasUsed)
			},
		},
		{
			name: "symbol() but lacking gas",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,            // from
					&vfbcContractAddrOfNative,          // to
					senderNonce(),                      // nonce
					nil,                                // amount
					params.TxGas+types.VFBCopgSymbol-1, // gas limit
					big.NewInt(1),                      // gas price
					big.NewInt(1),                      // gas fee cap
					big.NewInt(1),                      // gas tip cap
					callDataSymbol,                     // call data
					nil,                                // access list
					false,                              // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "out of gas")
				suite.Equal(vm.ErrOutOfGas.Error(), response.VmError)
				suite.Equal(msg.Gas(), response.GasUsed, "out of gas consume all gas")
			},
		},
		{
			name: "symbol() but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,     // from
					&vfbcContractAddrOfNative,   // to
					senderNonce(),               // nonce
					nil,                         // amount
					40_000,                      // gas limit
					big.NewInt(1),               // gas price
					big.NewInt(1),               // gas fee cap
					big.NewInt(1),               // gas tip cap
					append(callDataSymbol, 0x1), // call data
					nil,                         // access list
					false,                       // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgSymbol_Revert, response.GasUsed)
			},
		},
		{
			name: "symbol()",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataSymbol,            // call data
					nil,                       // access list
					false,                     // is fake
				)
			},
			wantExecError: false,
			wantVmError:   false,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				suite.Equal(vfbcDenomMetadataOfNative.Symbol, utils.MustAbiDecodeString(response.Ret))
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgSymbol, response.GasUsed)
			},
		},
		{
			name: "decimals() but lacking gas",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,              // from
					&vfbcContractAddrOfNative,            // to
					senderNonce(),                        // nonce
					nil,                                  // amount
					params.TxGas+types.VFBCopgDecimals-1, // gas limit
					big.NewInt(1),                        // gas price
					big.NewInt(1),                        // gas fee cap
					big.NewInt(1),                        // gas tip cap
					callDataDecimals,                     // call data
					nil,                                  // access list
					false,                                // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "out of gas")
				suite.Equal(vm.ErrOutOfGas.Error(), response.VmError)
				suite.Equal(msg.Gas(), response.GasUsed, "out of gas consume all gas")
			},
		},
		{
			name: "decimals() but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,       // from
					&vfbcContractAddrOfNative,     // to
					senderNonce(),                 // nonce
					nil,                           // amount
					40_000,                        // gas limit
					big.NewInt(1),                 // gas price
					big.NewInt(1),                 // gas fee cap
					big.NewInt(1),                 // gas tip cap
					append(callDataDecimals, 0x1), // call data
					nil,                           // access list
					false,                         // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgDecimals_Revert, response.GasUsed)
			},
		},
		{
			name: "decimals()",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataDecimals,          // call data
					nil,                       // access list
					false,                     // is fake
				)
			},
			wantExecError: false,
			wantVmError:   false,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				suite.Equal(vfbcDenomMetadataOfNative.Decimals, uint32(new(big.Int).SetBytes(response.Ret).Uint64()))
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgDecimals, response.GasUsed)
			},
		},
		{
			name: "totalSupply() but lacking gas",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					params.TxGas+types.VFBCopgTotalSupply-1, // gas limit
					big.NewInt(1),       // gas price
					big.NewInt(1),       // gas fee cap
					big.NewInt(1),       // gas tip cap
					callDataTotalSupply, // call data
					nil,                 // access list
					false,               // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "out of gas")
				suite.Equal(vm.ErrOutOfGas.Error(), response.VmError)
				suite.Equal(msg.Gas(), response.GasUsed, "out of gas consume all gas")
			},
		},
		{
			name: "totalSupply() but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,          // from
					&vfbcContractAddrOfNative,        // to
					senderNonce(),                    // nonce
					nil,                              // amount
					40_000,                           // gas limit
					big.NewInt(1),                    // gas price
					big.NewInt(1),                    // gas fee cap
					big.NewInt(1),                    // gas tip cap
					append(callDataTotalSupply, 0x1), // call data
					nil,                              // access list
					false,                            // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTotalSupply_Revert, response.GasUsed)
			},
		},
		{
			name: "totalSupply()",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataTotalSupply,       // call data
					nil,                       // access list
					false,                     // is fake
				)
			},
			wantExecError: false,
			wantVmError:   false,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				retBigInt := new(big.Int).SetBytes(response.Ret)
				totalSupply := suite.app.BankKeeper.GetSupply(suite.ctx, suite.denom).Amount.BigInt()
				suite.Truef(retBigInt.Cmp(totalSupply) == 0, "total supply mismatch, want %s, got %s", totalSupply, retBigInt)
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTotalSupply, response.GasUsed)
			},
		},
		{
			name: "balanceOf(address) but lacking gas",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					params.TxGas+types.VFBCopgBalanceOf-1, // gas limit
					big.NewInt(1), // gas price
					big.NewInt(1), // gas fee cap
					big.NewInt(1), // gas tip cap
					append(callDataBalanceOf, common.BytesToHash(randomVFBCSenderAddress.Bytes()).Bytes()...), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "out of gas")
				suite.Equal(vm.ErrOutOfGas.Error(), response.VmError)
				suite.Equal(msg.Gas(), response.GasUsed, "out of gas consume all gas")
			},
		},
		{
			name: "balanceOf(address) but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,        // from
					&vfbcContractAddrOfNative,      // to
					senderNonce(),                  // nonce
					nil,                            // amount
					40_000,                         // gas limit
					big.NewInt(1),                  // gas price
					big.NewInt(1),                  // gas fee cap
					big.NewInt(1),                  // gas tip cap
					append(callDataBalanceOf, 0x1), // call data
					nil,                            // access list
					false,                          // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgBalanceOf_Revert, response.GasUsed)
			},
		},
		{
			name: "balanceOf(address) but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					append(callDataBalanceOf, make([]byte, 33)...), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgBalanceOf_Revert, response.GasUsed)
			},
		},
		{
			name: "balanceOf(address) but invalid address, take only last 20 bytes",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					append(callDataBalanceOf, func() []byte {
						invalidAddress := common.BytesToHash(randomVFBCSenderAddress.Bytes()).Bytes()
						for i := 0; i < 12; i++ {
							invalidAddress[i] = 0xff
						}
						return invalidAddress
					}()...), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError: false,
			wantVmError:   false,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				retBigInt := new(big.Int).SetBytes(response.Ret)
				balance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCSenderAddress)
				suite.Truef(retBigInt.Cmp(balance) == 0, "balance mismatch, want %s, got %s", balance, retBigInt)
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgBalanceOf, response.GasUsed)
			},
		},
		{
			name: "balanceOf(address)",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					append(callDataBalanceOf, common.BytesToHash(randomVFBCSenderAddress.Bytes()).Bytes()...), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError: false,
			wantVmError:   false,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				retBigInt := new(big.Int).SetBytes(response.Ret)
				balance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCSenderAddress)
				suite.Truef(retBigInt.Cmp(balance) == 0, "balance mismatch, want %s, got %s", balance, retBigInt)
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgBalanceOf, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256) but lacking gas",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,              // from
					&vfbcContractAddrOfNative,            // to
					senderNonce(),                        // nonce
					nil,                                  // amount
					params.TxGas+types.VFBCopgTransfer-1, // gas limit
					big.NewInt(1),                        // gas price
					big.NewInt(1),                        // gas fee cap
					big.NewInt(1),                        // gas tip cap
					callDataTransfer(common.Big1),        // call data
					nil,                                  // access list
					false,                                // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "out of gas")
				suite.Equal(vm.ErrOutOfGas.Error(), response.VmError)
				suite.Equal(msg.Gas(), response.GasUsed, "out of gas consume all gas")
			},
		},
		{
			name: "transfer(address,uint256) but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,          // from
					&vfbcContractAddrOfNative,        // to
					senderNonce(),                    // nonce
					nil,                              // amount
					40_000,                           // gas limit
					big.NewInt(1),                    // gas price
					big.NewInt(1),                    // gas fee cap
					big.NewInt(1),                    // gas tip cap
					append(callDataTransferSig, 0x1), // call data
					nil,                              // access list
					false,                            // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer_Revert, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256) but invalid call data",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					append(callDataTransferSig, make([]byte, 32+32+1)...), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError: false,
			wantVmError:   true,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "invalid call data")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer_Revert, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), transfer more than balance",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataTransfer(new(big.Int).Add(vfbcSenderInitialBalance, common.Big1)), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       true,
			wantSenderBalanceAdjustmentOrZero: common.Big0,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().True(response.Failed())
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress)
				suite.Truef(receiverBalance.Sign() == 0, "receiver should receives nothing but got %s", receiverBalance)
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "transfer amount exceeds balance")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer_Revert, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), transfer 1",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,       // from
					&vfbcContractAddrOfNative,     // to
					senderNonce(),                 // nonce
					nil,                           // amount
					40_000,                        // gas limit
					big.NewInt(1),                 // gas price
					big.NewInt(1),                 // gas fee cap
					big.NewInt(1),                 // gas tip cap
					callDataTransfer(common.Big1), // call data
					nil,                           // access list
					false,                         // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       false,
			wantSenderBalanceAdjustmentOrZero: common.Big1,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				suite.Equal(bytesOfAbiEncodedTrue, response.Ret)
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress)
				suite.Truef(receiverBalance.Cmp(common.Big1) == 0, "balance mismatch, want %s, got %s", common.Big1, receiverBalance)
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), transfer zero",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,       // from
					&vfbcContractAddrOfNative,     // to
					senderNonce(),                 // nonce
					nil,                           // amount
					40_000,                        // gas limit
					big.NewInt(1),                 // gas price
					big.NewInt(1),                 // gas fee cap
					big.NewInt(1),                 // gas tip cap
					callDataTransfer(common.Big0), // call data
					nil,                           // access list
					false,                         // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       false,
			wantSenderBalanceAdjustmentOrZero: common.Big0,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				suite.Equal(bytesOfAbiEncodedTrue, response.Ret)
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress)
				suite.Truef(receiverBalance.Sign() == 0, "balance mismatch, want %s, got %s", common.Big0, receiverBalance)
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), transfer value equals to max uint64",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataTransfer(new(big.Int).SetUint64(math.MaxUint64)), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       false,
			wantSenderBalanceAdjustmentOrZero: new(big.Int).SetUint64(math.MaxUint64),
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				suite.Equal(bytesOfAbiEncodedTrue, response.Ret)
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress)
				wantReceiverBalance := new(big.Int).SetUint64(math.MaxUint64)
				suite.Truef(receiverBalance.Cmp(wantReceiverBalance) == 0, "balance mismatch, want %s, got %s", new(big.Int).SetUint64(math.MaxUint64), receiverBalance)
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), but invalid address, take the first 20 bytes",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					func() []byte {
						callData := callDataTransfer(common.Big1)
						for i := 5; i < 15; i++ {
							callData[i] = 0xff
						}
						return callData
					}(), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       false,
			wantSenderBalanceAdjustmentOrZero: common.Big1,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().False(response.Failed())
				suite.Equal(bytesOfAbiEncodedTrue, response.Ret)
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress)
				suite.Truef(receiverBalance.Cmp(common.Big1) == 0, "balance mismatch, want %s, got %s", common.Big1, receiverBalance)
				suite.Empty(response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), can not transfer to module account",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataTransferTo(types.VirtualFrontierContractDeployerAddress, common.Big1), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       true,
			wantSenderBalanceAdjustmentOrZero: common.Big0,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().True(response.Failed())
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, types.VirtualFrontierContractDeployerAddress)
				suite.Truef(receiverBalance.Sign() == 0, "receiver should receives nothing but got %s", receiverBalance)
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "can not transfer to module account")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer_Revert, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), can not transfer to VFC contract",
			prepare: func() core.Message {
				return ethtypes.NewMessage(
					randomVFBCSenderAddress,   // from
					&vfbcContractAddrOfNative, // to
					senderNonce(),             // nonce
					nil,                       // amount
					40_000,                    // gas limit
					big.NewInt(1),             // gas price
					big.NewInt(1),             // gas fee cap
					big.NewInt(1),             // gas tip cap
					callDataTransferTo(vfbcContractAddrOfNative, common.Big1), // call data
					nil,   // access list
					false, // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       true,
			wantSenderBalanceAdjustmentOrZero: common.Big0,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().True(response.Failed())
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, vfbcContractAddrOfNative)
				suite.Truef(receiverBalance.Sign() == 0, "receiver should receives nothing but got %s", receiverBalance)
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "not allowed to receive")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer_Revert, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), can not transfer coin that not send-able",
			prepare: func() core.Message {
				bankParams := suite.app.BankKeeper.GetParams(suite.ctx)
				bankParams.SendEnabled = []*banktypes.SendEnabled{{
					Denom:   suite.denom,
					Enabled: false,
				}}
				suite.app.BankKeeper.SetParams(suite.ctx, bankParams)

				return ethtypes.NewMessage(
					randomVFBCSenderAddress,       // from
					&vfbcContractAddrOfNative,     // to
					senderNonce(),                 // nonce
					nil,                           // amount
					40_000,                        // gas limit
					big.NewInt(1),                 // gas price
					big.NewInt(1),                 // gas fee cap
					big.NewInt(1),                 // gas tip cap
					callDataTransfer(common.Big1), // call data
					nil,                           // access list
					false,                         // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       true,
			wantSenderBalanceAdjustmentOrZero: common.Big0,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().True(response.Failed())
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress)
				suite.Truef(receiverBalance.Sign() == 0, "receiver should receives nothing but got %s", receiverBalance)
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "transfers are currently disabled")
				suite.Equal(vm.ErrExecutionReverted.Error(), response.VmError)
				suite.Equal(computeIntrinsicGas(msg)+types.VFBCopgTransfer_Revert, response.GasUsed)
			},
		},
		{
			name: "transfer(address,uint256), can not transfer if contract was de-activated",
			prepare: func() core.Message {
				contract := *vfc
				contract.Active = false
				suite.app.EvmKeeper.SetVirtualFrontierContract(suite.ctx, vfbcContractAddrOfNative, &contract)

				return ethtypes.NewMessage(
					randomVFBCSenderAddress,       // from
					&vfbcContractAddrOfNative,     // to
					senderNonce(),                 // nonce
					nil,                           // amount
					40_000,                        // gas limit
					big.NewInt(1),                 // gas price
					big.NewInt(1),                 // gas fee cap
					big.NewInt(1),                 // gas tip cap
					callDataTransfer(common.Big1), // call data
					nil,                           // access list
					false,                         // is fake
				)
			},
			wantExecError:                     false,
			wantVmError:                       true,
			wantSenderBalanceAdjustmentOrZero: common.Big0,
			testOnNonExecError: func(msg core.Message, response *types.MsgEthereumTxResponse) {
				suite.Require().True(response.Failed())
				receiverBalance := suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress)
				suite.Truef(receiverBalance.Sign() == 0, "receiver should receives nothing but got %s", receiverBalance)
				suite.Contains(utils.MustAbiDecodeString(response.Ret[4:]), "is not active")
				suite.Contains(response.VmError, "is not active")
				suite.Equal(msg.Gas(), response.GasUsed, "error tx consumes all gas")
			},
		},
	}
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			// minting

			coins := sdk.NewCoins(sdk.NewCoin(suite.denom, sdkmath.NewIntFromBigInt(vfbcSenderInitialBalance)))
			suite.app.BankKeeper.MintCoins(suite.ctx, minttypes.ModuleName, coins)
			suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, minttypes.ModuleName, randomVFBCSenderAddress.Bytes(), coins)

			// prepare exec ctx

			proposerAddress := suite.ctx.BlockHeader().ProposerAddress
			config, err := suite.app.EvmKeeper.EVMConfig(suite.ctx, proposerAddress, big.NewInt(9000))
			suite.Require().NoError(err)

			txConfig := suite.app.EvmKeeper.TxConfig(suite.ctx, common.Hash{})

			msg := tc.prepare()

			preExecNonce := suite.app.EvmKeeper.GetNonce(suite.ctx, randomVFBCSenderAddress)

			res, err := suite.app.EvmKeeper.ApplyMessageWithConfig(suite.ctx, msg, nil, true, config, txConfig)

			suite.Equal(preExecNonce, suite.app.EvmKeeper.GetNonce(suite.ctx, randomVFBCSenderAddress), "nonce increment must be handled by AnteHandler")

			if tc.wantExecError {
				suite.Require().Error(err)
				suite.Equal(vfbcSenderInitialBalance, suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCSenderAddress), "balance should not change on failed txs")
				return
			}

			suite.Require().NoError(err)

			wantBalanceAdjustmentOfSender := tc.wantSenderBalanceAdjustmentOrZero
			if wantBalanceAdjustmentOfSender == nil {
				wantBalanceAdjustmentOfSender = common.Big0
			}

			actualBalanceAdjustmentOfSender := new(big.Int).Sub(vfbcSenderInitialBalance, suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCSenderAddress))
			actualBalanceAdjustmentOfSender = new(big.Int).Sub(actualBalanceAdjustmentOfSender, feeCompute(msg, res.GasUsed))

			suite.Truef(wantBalanceAdjustmentOfSender.Cmp(actualBalanceAdjustmentOfSender) == 0, "balance adjustment mismatch, want %s, got %s", wantBalanceAdjustmentOfSender, actualBalanceAdjustmentOfSender)

			if tc.wantVmError {
				suite.Require().True(res.Failed())
				suite.Require().NotEmpty(res.VmError)
				suite.True(suite.app.EvmKeeper.GetBalance(suite.ctx, randomVFBCReceiverAddress).Sign() == 0, "receiver should receives nothing")
			} else {
				suite.Require().False(res.Failed())
				suite.Require().Empty(res.VmError)
			}

			suite.Require().NotNil(tc.testOnNonExecError, "post-test is required")
			tc.testOnNonExecError(msg, res)
		})
	}
}
