// Copyright 2021 Evmos Foundation
// This file is part of Evmos' Ethermint library.
//
// The Ethermint library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Ethermint library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Ethermint library. If not, see https://github.com/evmos/ethermint/blob/main/LICENSE
package ante_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/evmos/ethermint/app/ante"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	"github.com/evmos/ethermint/ethereum/eip712"
	"github.com/evmos/ethermint/tests"
	ethermint "github.com/evmos/ethermint/types"
)

func TestEIP712AnteHandlerSuite(t *testing.T) {
	suite.Run(t, new(EIP712AnteHandlerSuite))
}

// TestEIP712SigVerificationWrongAccountNumber tests verification with incorrect account number
func (suite *EIP712AnteHandlerSuite) TestEIP712SigVerificationWrongAccountNumber() {
	suite.SetupTest() // reset
	suite.txBuilder = suite.CreateTestEIP712TxBuilder()

	// Create private key for the test account
	privKey, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	pubKey := privKey.PubKey()

	// Create a test account and fund it
	addr := sdk.AccAddress(pubKey.Address())
	account := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr)
	suite.Require().NoError(err)
	
	// Set the account number and sequence
	acc := account.(*ethermint.EthAccount)
	acc.SetSequence(0)
	acc.SetAccountNumber(5) // Set an account number different from what we'll use in the transaction
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	// Fund the account
	amount := sdk.NewCoins(sdk.NewCoin(suite.evmParams.EvmDenom, sdk.NewInt(100000000000000)))
	err = suite.app.BankKeeper.MintCoins(suite.ctx, "evm", amount)
	suite.Require().NoError(err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, "evm", addr, amount)
	suite.Require().NoError(err)

	// Create a bank MsgSend for testing
	sendMsg := banktypes.NewMsgSend(addr, tests.GenerateAddress().Bytes(), sdk.NewCoins(sdk.NewCoin(suite.evmParams.EvmDenom, sdk.NewInt(1000))))
	msgs := []sdk.Msg{sendMsg}

	// Chain ID for EIP-712 signature
	chainID := suite.app.EvmKeeper.ChainID()
	suite.Require().NotNil(chainID)
	chainIDInt := chainID.ToBig()

	// Create EIP-712 transaction with extension options
	option, err := eip712.WrapTxToExtensionOptionsWeb3Tx(
		sdk.NewInt(0),
		0,
		&eip712.FeeDelegationOptions{
			FeePayer: addr,
		},
	)
	suite.Require().NoError(err)

	// Build EIP-712 transaction with account number 0 (different from the account's actual number 5)
	extOpt, ok := option.(*ethermint.ExtensionOptionsWeb3Tx)
	suite.Require().True(ok)
	extOpt.TypedDataChainID = chainIDInt.Uint64()
	tx, err := suite.CreateTestEIP712CosmosTxBuilder(
		privKey, suite.ctx.ChainID(), chainIDInt, msgs, extOpt, 
		suite.ctx.BlockHeight(), suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx),
	).GetTx()
	suite.Require().NoError(err)

	// Create the ante handler
	decorator := ante.NewLegacyEip712SigVerificationDecorator(suite.app.AccountKeeper)

	// Run the handler
	_, err = decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	
	// Expect error due to account number mismatch
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "signature verification failed")
}

// TestEIP712SigVerificationSimulationMode tests that signature verification passes in simulation mode
func (suite *EIP712AnteHandlerSuite) TestEIP712SigVerificationSimulationMode() {
	suite.SetupTest() // reset
	suite.txBuilder = suite.CreateTestEIP712TxBuilder()

	// Create private key for the test account
	privKey, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	pubKey := privKey.PubKey()

	// Create a test account and fund it
	addr := sdk.AccAddress(pubKey.Address())
	account := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr)
	suite.Require().NoError(err)
	
	// Set the account number and sequence
	acc := account.(*ethermint.EthAccount)
	acc.SetSequence(0)
	acc.SetAccountNumber(0)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	// Fund the account
	amount := sdk.NewCoins(sdk.NewCoin(suite.evmParams.EvmDenom, sdk.NewInt(100000000000000)))
	err = suite.app.BankKeeper.MintCoins(suite.ctx, "evm", amount)
	suite.Require().NoError(err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, "evm", addr, amount)
	suite.Require().NoError(err)

	// Create a bank MsgSend for testing
	sendMsg := banktypes.NewMsgSend(addr, tests.GenerateAddress().Bytes(), sdk.NewCoins(sdk.NewCoin(suite.evmParams.EvmDenom, sdk.NewInt(1000))))
	msgs := []sdk.Msg{sendMsg}

	// Chain ID for EIP-712 signature
	chainID := suite.app.EvmKeeper.ChainID()
	suite.Require().NotNil(chainID)
	chainIDInt := chainID.ToBig()

	// Create EIP-712 transaction with extension options
	option, err := eip712.WrapTxToExtensionOptionsWeb3Tx(
		sdk.NewInt(0),
		0,
		&eip712.FeeDelegationOptions{
			FeePayer: addr,
		},
	)
	suite.Require().NoError(err)

	// Build EIP-712 transaction with a wrong chain ID to force a signature verification error
	// This would normally fail, but should pass in simulation mode
	extOpt, ok := option.(*ethermint.ExtensionOptionsWeb3Tx)
	suite.Require().True(ok)
	// Use wrong chain ID in the transaction
	wrongChainIDInt := big.NewInt(chainIDInt.Int64() + 1)
	extOpt.TypedDataChainID = wrongChainIDInt.Uint64()
	tx, err := suite.CreateTestEIP712CosmosTxBuilder(
		privKey, suite.ctx.ChainID(), chainIDInt, msgs, extOpt, 
		suite.ctx.BlockHeight(), suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx),
	).GetTx()
	suite.Require().NoError(err)

	// Create the ante handler
	decorator := ante.NewLegacyEip712SigVerificationDecorator(suite.app.AccountKeeper)

	// Run the handler in simulation mode (simulate=true)
	ctx, err := decorator.AnteHandle(suite.ctx, tx, true, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	
	// Expect no error because we are in simulation mode, where signature verification should be skipped
	suite.Require().NoError(err)
	suite.Require().NotNil(ctx)
}

// EIP712AnteHandlerSuite tests the EIP-712 signature verification ante handler
type EIP712AnteHandlerSuite struct {
	suite.Suite
	ante.AnteTestSuite
}

// SetupTest sets up the test suite
func (suite *EIP712AnteHandlerSuite) SetupTest() {
	suite.AnteTestSuite.SetupTest()
}

// TestEIP712SigVerification tests the verification of EIP-712 signatures
func (suite *EIP712AnteHandlerSuite) TestEIP712SigVerification() {
	suite.SetupTest() // reset
	suite.txBuilder = suite.CreateTestEIP712TxBuilder()

	// Create and set up a private key for the test account
	privKey, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	pubKey := privKey.PubKey()

	// Create a test account and fund it
	addr := sdk.AccAddress(pubKey.Address())
	account := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr)
	suite.Require().NoError(err)
	
	// Set the account number and sequence
	acc := account.(*ethermint.EthAccount)
	acc.SetSequence(0)
	acc.SetAccountNumber(0)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	// Fund the account
	amount := sdk.NewCoins(sdk.NewCoin(suite.evmParams.EvmDenom, sdk.NewInt(100000000000000)))
	err = suite.app.BankKeeper.MintCoins(suite.ctx, "evm", amount)
	suite.Require().NoError(err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, "evm", addr, amount)
	suite.Require().NoError(err)

	// Create a bank MsgSend for testing
	sendMsg := banktypes.NewMsgSend(addr, tests.GenerateAddress().Bytes(), sdk.NewCoins(sdk.NewCoin(suite.evmParams.EvmDenom, sdk.NewInt(1000))))
	msgs := []sdk.Msg{sendMsg}

	// Chain ID for EIP-712 signature
	chainID := suite.app.EvmKeeper.ChainID()
	suite.Require().NotNil(chainID)
	chainIDInt := chainID.ToBig()

	// Create EIP-712 transaction with extension options
	option, err := eip712.WrapTxToExtensionOptionsWeb3Tx(
		sdk.NewInt(0),
		0,
		&eip712.FeeDelegationOptions{
			FeePayer: addr,
		},
	)
	suite.Require().NoError(err)

	// Build EIP-712 transaction
	extOpt, ok := option.(*ethermint.ExtensionOptionsWeb3Tx)
	suite.Require().True(ok)
	extOpt.TypedDataChainID = chainIDInt.Uint64()
	tx, err := suite.CreateTestEIP712CosmosTxBuilder(
		privKey, suite.ctx.ChainID(), chainIDInt, msgs, extOpt, 
		suite.ctx.BlockHeight(), suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx),
	).GetTx()
	suite.Require().NoError(err)

	// Create the ante handler
	decorator := ante.NewLegacyEip712SigVerificationDecorator(suite.app.AccountKeeper)

	// Run the handler
	ctx, err := decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	
	// Expect no error in signature verification
	suite.Require().NoError(err)
	suite.Require().NotNil(ctx)
}

// TestEIP712SigVerificationInvalidSignature tests verification failure with invalid signature
func (suite *EIP712AnteHandlerSuite) TestEIP712SigVerificationInvalidSignature() {
	suite.SetupTest() // reset
	suite.txBuilder = suite.CreateTestEIP712TxBuilder()

	// Create two private keys - one for the account and another for an invalid signature
	privKey, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	invalidPrivKey, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	pubKey := privKey.PubKey()

	// Create a test account and fund it
	addr := sdk.AccAddress(pubKey.Address())
	account := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr)
	suite.Require().NoError(err)
	
	// Set the account number and sequence
	acc := account.(*ethermint.EthAccount)
	acc.SetSequence(0)
	acc.SetAccountNumber(0)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	// Fund the account
	amount := sdk.NewCoins(sdk.NewCoin(suite.evmParams.EvmDenom, sdk.NewInt(100000000000000)))
	err = suite.app.BankKeeper.MintCoins(suite.ctx, "evm", amount)
	suite.Require().NoError(err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, "evm", addr, amount)
	suite.Require().NoError(err)

	// Create a bank MsgSend for testing
	sendMsg := banktypes.NewMsgSend(addr, tests.GenerateAddress().Bytes(), sdk.NewCoins(sdk.NewCoin(suite.evmParams.EvmDenom, sdk.NewInt(1000))))
	msgs := []sdk.Msg{sendMsg}

	// Chain ID for EIP-712 signature
	chainID := suite.app.EvmKeeper.ChainID()
	suite.Require().NotNil(chainID)
	chainIDInt := chainID.ToBig()

	// Create EIP-712 transaction with extension options
	option, err := eip712.WrapTxToExtensionOptionsWeb3Tx(
		sdk.NewInt(0),
		0,
		&eip712.FeeDelegationOptions{
			FeePayer: addr,
		},
	)
	suite.Require().NoError(err)

	// Build EIP-712 transaction using an invalid private key
	extOpt, ok := option.(*ethermint.ExtensionOptionsWeb3Tx)
	suite.Require().True(ok)
	extOpt.TypedDataChainID = chainIDInt.Uint64()
	tx, err := suite.CreateTestEIP712CosmosTxBuilder(
		invalidPrivKey, suite.ctx.ChainID(), chainIDInt, msgs, extOpt, 
		suite.ctx.BlockHeight(), suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx),
	).GetTx()
	suite.Require().NoError(err)

	// Create the ante handler
	decorator := ante.NewLegacyEip712SigVerificationDecorator(suite.app.AccountKeeper)

	// Run the handler
	_, err = decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	
	// Expect error in signature verification because we signed with a different key
	suite.Require().Error(err)
}

// TestEIP712SigVerificationWrongChainID tests verification with incorrect chain ID
func (suite *EIP712AnteHandlerSuite) TestEIP712SigVerificationWrongChainID() {
	suite.SetupTest() // reset
	suite.txBuilder = suite.CreateTestEIP712TxBuilder()

	// Create private key for the test account
	privKey, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	pubKey := privKey.PubKey()

	// Create a test account and fund it
	addr := sdk.AccAddress(pubKey.Address())
	account := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr)
	suite.Require().NoError(err)
	
	// Set the account number and sequence
	acc := account.(*ethermint.EthAccount)
	acc.SetSequence(0)
	acc.SetAccountNumber(0)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	// Fund the account
	amount := sdk.NewCoins(sdk.NewCoin(suite.evmParams.EvmDenom, sdk.NewInt(100000000000000)))
	err = suite.app.BankKeeper.MintCoins(suite.ctx, "evm", amount)
	suite.Require().NoError(err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, "evm", addr, amount)
	suite.Require().NoError(err)

	// Create a bank MsgSend for testing
	sendMsg := banktypes.NewMsgSend(addr, tests.GenerateAddress().Bytes(), sdk.NewCoins(sdk.NewCoin(suite.evmParams.EvmDenom, sdk.NewInt(1000))))
	msgs := []sdk.Msg{sendMsg}

	// Chain ID for EIP-712 signature
	chainID := suite.app.EvmKeeper.ChainID()
	suite.Require().NotNil(chainID)
	chainIDInt := chainID.ToBig()
	
	// Use an incorrect chain ID for testing
	incorrectChainIDInt := big.NewInt(chainIDInt.Int64() + 1)

	// Create EIP-712 transaction with extension options
	option, err := eip712.WrapTxToExtensionOptionsWeb3Tx(
		sdk.NewInt(0),
		0,
		&eip712.FeeDelegationOptions{
			FeePayer: addr,
		},
	)
	suite.Require().NoError(err)

	// Build EIP-712 transaction with incorrect chain ID in the extension options
	extOpt, ok := option.(*ethermint.ExtensionOptionsWeb3Tx)
	suite.Require().True(ok)
	extOpt.TypedDataChainID = incorrectChainIDInt.Uint64()
	tx, err := suite.CreateTestEIP712CosmosTxBuilder(
		privKey, suite.ctx.ChainID(), chainIDInt, msgs, extOpt, 
		suite.ctx.BlockHeight(), suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx),
	).GetTx()
	suite.Require().NoError(err)

	// Create the ante handler
	decorator := ante.NewLegacyEip712SigVerificationDecorator(suite.app.AccountKeeper)

	// Run the handler
	_, err = decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	
	// Expect error due to chain ID mismatch
	suite.Require().Error(err)
}

// TestEIP712SigVerificationWrongSequence tests verification with incorrect sequence number
func (suite *EIP712AnteHandlerSuite) TestEIP712SigVerificationWrongSequence() {
	suite.SetupTest() // reset
	suite.txBuilder = suite.CreateTestEIP712TxBuilder()

	// Create private key for the test account
	privKey, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	pubKey := privKey.PubKey()

	// Create a test account and fund it
	addr := sdk.AccAddress(pubKey.Address())
	account := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr)
	suite.Require().NoError(err)
	
	// Set the account number and sequence
	acc := account.(*ethermint.EthAccount)
	acc.SetSequence(5) // Set a sequence different from what we'll use in the transaction
	acc.SetAccountNumber(0)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	// Fund the account
	amount := sdk.NewCoins(sdk.NewCoin(suite.evmParams.EvmDenom, sdk.NewInt(100000000000000)))
	err = suite.app.BankKeeper.MintCoins(suite.ctx, "evm", amount)
	suite.Require().NoError(err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, "evm", addr, amount)
	suite.Require().NoError(err)

	// Create a bank MsgSend for testing
	sendMsg := banktypes.NewMsgSend(addr, tests.GenerateAddress().Bytes(), sdk.NewCoins(sdk.NewCoin(suite.evmParams.EvmDenom, sdk.NewInt(1000))))
	msgs := []sdk.Msg{sendMsg}

	// Chain ID for EIP-712 signature
	chainID := suite.app.EvmKeeper.ChainID()
	suite.Require().NotNil(chainID)
	chainIDInt := chainID.ToBig()

	// Create EIP-712 transaction with extension options
	option, err := eip712.WrapTxToExtensionOptionsWeb3Tx(
		sdk.NewInt(0),
		0, // Using sequence 0 when account has sequence 5
		&eip712.FeeDelegationOptions{
			FeePayer: addr,
		},
	)
	suite.Require().NoError(err)

	// Build EIP-712 transaction
	extOpt, ok := option.(*ethermint.ExtensionOptionsWeb3Tx)
	suite.Require().True(ok)
	extOpt.TypedDataChainID = chainIDInt.Uint64()
	tx, err := suite.CreateTestEIP712CosmosTxBuilder(
		privKey, suite.ctx.ChainID(), chainIDInt, msgs, extOpt, 
		suite.ctx.BlockHeight(), suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx),
	).GetTx()
	suite.Require().NoError(err)

	// Create the ante handler
	decorator := ante.NewLegacyEip712SigVerificationDecorator(suite.app.AccountKeeper)

	// Run the handler
	_, err = decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	
	// Expect error due to sequence mismatch
	suite.Require().Error(err)
}

