package integration_test_util

//goland:noinspection SpellCheckingInspection

// HasCometBFT indicate if the integration chain has CometBFT enabled.
func (suite *ChainIntegrationTestSuite) HasCometBFT() bool {
	return !suite.TestConfig.DisableCometBFT && suite.CometBFTApp != nil
}

// EnsureCometBFT trigger test failure immediately if CometBFT is not enabled on integration chain.
func (suite *ChainIntegrationTestSuite) EnsureCometBFT() {
	if !suite.HasCometBFT() {
		suite.Require().FailNow("CometBFT node must be initialized")
	}
}
