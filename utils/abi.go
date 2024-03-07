package utils

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

func AbiEncodeString(str string) ([]byte, error) {
	return abiArgsSingleString.Pack(str)
}

func MustAbiEncodeString(str string) []byte {
	bz, err := AbiEncodeString(str)
	if err != nil {
		panic(err)
	}
	return bz
}

func AbiDecodeString(bz []byte) (string, error) {
	res, err := abiArgsSingleString.Unpack(bz)
	if err != nil {
		return "", err
	}
	if len(res) != 1 {
		return "", fmt.Errorf("is not a single string")
	}
	if str, ok := res[0].(string); ok {
		return str, nil
	}
	return "", fmt.Errorf("is not string")
}

func MustAbiDecodeString(bz []byte) string {
	str, err := AbiDecodeString(bz)
	if err != nil {
		panic(err)
	}
	return str
}

var abiTypeString abi.Type
var abiArgsSingleString abi.Arguments

func init() {
	var err error
	abiTypeString, err = abi.NewType("string", "string", nil)
	if err != nil {
		panic(err)
	}

	abiArgsSingleString = abi.Arguments{
		abi.Argument{
			Name: "content",
			Type: abiTypeString,
		},
	}
}
