<!--
order: 10
-->

# Virtual Frontier Contract
Author: [Victor Pham](https://github.com/VictorTrustyDev)

Virtual Frontier Contract is
- A new type of smart contract in Ethermint fork version of Dymension.
- A contract that can be interacted directly via Metamask or other Ethereum wallets.
- A contract that is virtualized and does not exist on the blockchain.
- Open ways to interact with Cosmos blockchain business logic via MM or other Ethereum wallets.

Sub-types:
- Virtual Frontier Bank Contract

Technical notes:
- Standing in front of EVM, doing stuffs instead of actually interacting EVM.
- Prohibit to communicated within EVM.
- Should not receive funds. If received, it results lost forever. Currently, when an Ethereum tx, with value != 0 (direct transfer or payable method call), are aborted. Not yet any implementation to prevent from Cosmos side.
- New module store: Contract and meta corresponds to the sub-type.

# Virtual Frontier Bank Contract

Virtual Frontier Bank Contract is
- Virtual Frontier Contract.
- A contract, simulated ERC-20 spec, allowed user to import to MM or other Ethereum wallets and can be used to transfer Cosmos bank assets via the wallets.
- Automatically deployed when there is a new denom metadata created in bank module.

Technical notes:
- New module store: Mapping from denom to contract address.
- Check bank denom metadata in begin block and deploy the valid ones.
- Can be switch activation state via gov: `ethermintd tx gov submit-legacy-proposal update-vfc-bank proposal_file.json`.
- ERC-20 compatible:
  - Support:
    - `name()`
    - `symbol()`
    - `decimals()`
    - `totalSupply()`
    - `balanceOf(address)`
    - `transfer(address, uint256)`
    - event `Transfer(address, address, uint256)`
  - Not yet support (due to security concern and not necessary for the purpose of this contract):
    - `transferFrom(address, address, uint256)`
    - `approve(address, uint256)`
    - `allowance(address, address)`
    - event `Approval(address, address, uint256)`