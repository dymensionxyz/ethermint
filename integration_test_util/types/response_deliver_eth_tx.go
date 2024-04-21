package types

import (
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

type ResponseDeliverEthTx struct {
	CosmosTxHash         string
	EthTxHash            string
	EvmError             string
	ResponseDeliverEthTx *abci.ResponseDeliverTx
}

func NewResponseDeliverEthTx(responseDeliverTx *abci.ResponseDeliverTx) *ResponseDeliverEthTx {
	if responseDeliverTx == nil {
		return nil
	}

	response := &ResponseDeliverEthTx{
		ResponseDeliverEthTx: responseDeliverTx,
	}

	for _, event := range responseDeliverTx.Events {
		if event.Type == evmtypes.TypeMsgEthereumTx {
			for _, attribute := range event.Attributes {
				//fmt.Println(evmtypes.TypeMsgEthereumTx, "attribute.Key", attribute.Key, "attribute.Value", attribute.Value)
				if string(attribute.Key) == evmtypes.AttributeKeyTxHash {
					if len(attribute.Value) > 0 && response.CosmosTxHash == "" {
						response.CosmosTxHash = string(attribute.Value)
					}
				} else if string(attribute.Key) == evmtypes.AttributeKeyEthereumTxHash {
					if len(attribute.Value) > 0 && response.EthTxHash == "" {
						response.EthTxHash = string(attribute.Value)
					}
				} else if string(attribute.Key) == evmtypes.AttributeKeyEthereumTxFailed {
					if len(attribute.Value) > 0 && response.EvmError == "" {
						response.EvmError = string(attribute.Value)
					}
				}
			}
		}
	}

	return response
}
