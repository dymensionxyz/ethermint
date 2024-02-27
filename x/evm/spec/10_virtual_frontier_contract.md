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
- List of contract address is persisted into `x/evm` params (prohibited to be updated via gov. Gov proposal should leave it empty and historical record is preserved).
- Contract metadata is persisted in `x/evm` store.
- Should not receive funds. If received, it results lost forever. Currently, when an Ethereum tx, with value != 0 (direct transfer or payable method call), are aborted. Not yet any implementation to prevent from Cosmos side.

# Virtual Frontier Bank Contract

Virtual Frontier Bank Contract is
- Virtual Frontier Contract.
- A contract, simulated ERC-20 spec, allowed user to import to MM or other Ethereum wallets and can be used to transfer Cosmos bank assets via the wallets.
- Automatically register new IBC assets upon the very first incoming transfer into the chain, each denom will have it own contract address.
- Metadata contains "min denom", "display name" and "exponent". Metadata can be updated via gov: `ethermintd tx gov submit-legacy-proposal update-vfc-bank proposal_file.json`.