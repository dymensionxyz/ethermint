package types

import (
	"cosmossdk.io/errors"
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/ethereum/go-ethereum/common"
	"strings"
)

type VirtualFrontierContractType uint32

const (
	VirtualFrontierContractTypeUnknown VirtualFrontierContractType = iota
	VirtualFrontierContractTypeBankContract
)

// ValidateBasic performs basic validation of the VirtualFrontierContract fields
func (m *VirtualFrontierContract) ValidateBasic(cdc codec.BinaryCodec) error {
	emptyAddress := common.Address{}
	if m.Address == emptyAddress.String() {
		return fmt.Errorf("address cannot be nil address")
	}
	if !common.IsHexAddress(m.Address) {
		return fmt.Errorf("malformed address format: %s", m.Address)
	}
	if !strings.HasPrefix(m.Address, "0x") {
		return fmt.Errorf("address must start with 0x")
	}
	if m.Address != strings.ToLower(m.Address) {
		return fmt.Errorf("address must be in lowercase")
	}

	if len(m.Metadata) == 0 {
		return fmt.Errorf("metadata cannot be empty")
	}

	switch VirtualFrontierContractType(m.Type) {
	case VirtualFrontierContractTypeBankContract:
		var bankContractMetadata VFBankContractMetadata
		var err error

		err = cdc.Unmarshal(m.Metadata, &bankContractMetadata)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal bank contract metadata")
		}

		if err = bankContractMetadata.ValidateBasic(); err != nil {
			return errors.Wrap(err, "the inner bank contract metadata does not pass validation")
		}

		break
	case VirtualFrontierContractTypeUnknown:
		return fmt.Errorf("type must be specified")
	default:
		return fmt.Errorf("type must be specified")
	}

	return nil
}

// ContractAddress returns the contract address of the VirtualFrontierContract
func (m *VirtualFrontierContract) ContractAddress() common.Address {
	return common.HexToAddress(m.Address)
}

// GetTypeName returns the human-readable type name of the type of the VirtualFrontierContract
func (m *VirtualFrontierContract) GetTypeName() string {
	switch VirtualFrontierContractType(m.Type) {
	case VirtualFrontierContractTypeBankContract:
		return "bank"
	default:
		return ""
	}
}
