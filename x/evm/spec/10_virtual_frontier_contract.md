<!--
order: 10
-->

# Virtual Frontier Contract
Author: [Victor Pham](https://github.com/VictorTrustyDev)

Virtual Frontier Contract is
- A new type of smart contract in Ethermint fork version of Dymension.
- A contract that can be interacted directly via Metamask or other Ethereum wallets.
- Open ways to interact with Cosmos blockchain business logic via MM or other Ethereum wallets.

Sub-types:
- Virtual Frontier Bank Contract

Technical notes:
- Standing in front of EVM, doing stuffs instead of actually interacting EVM.
- Should not receive funds. If received, it results lost forever. Currently, when an Ethereum tx, with value != 0 (direct transfer or payable method call), are aborted. Not yet any implementation to prevent from Cosmos side.
- New module store: Contract and meta corresponds to the sub-type.
- Still a smart contract with its own _address, account state, nonce, code hash and deployed bytecode_. Except that the deployed bytecode is not used during execution.
- Interaction to VFC contract within EVM actual execution:
  - Will fail if modify the VFC contract state like:
    - Account state like balance, nonce
    - Contract code
    - Storage
  - Will success in other cases, for example transfer ERC-20 token from/into this address, because the execution only modify the state of the token contract, not the VFC contract itself.
- Should have actual solidity code & compiled into bytecode for deployment, accept the inner function going to be failed if called [(check this)](https://github.com/dymensionxyz/ethermint/blob/b2df154ea803a77a7329c0f927382c8a7beb7805/x/evm/types/VFBankContract20.sol).

# Virtual Frontier Bank Contract

Virtual Frontier Bank Contract is
- Virtual Frontier Contract.
- A contract, simulated ERC-20 spec, allowed user to import to MM or other Ethereum wallets and can be used to transfer Cosmos bank assets via the wallets.
- Deployed follow denom metadata created in bank module.
  - On Dymension, new contracts deployment will be triggered upon gov create new bank denom metadata (this type of gov provided by `x/denommetadata` module).
  - On Ethermint dev chain, new contracts deployment will be done automatically in next block, right after new bank denom metadata records are created.

Technical notes:
- New module stores:
  - Holding the contract information, mapped by address.
  - Mapping from denom to contract address.
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
- How to deploy:
  - New contracts for new bank denom metadata records:
    ```golang
    err := k.DeployVirtualFrontierBankContractForAllBankDenomMetadataRecords(ctx, nil)
    ```
  - New contract for a specific bank denom metadata record:
    ```golang
    err := k.DeployVirtualFrontierBankContractForBankDenomMetadataRecord(ctx, "ibc/uatom")
    ```
    or
    ```golang
    err := k.DeployNewVirtualFrontierBankContract(ctx, &types.VirtualFrontierContract{
        Active: true,
    }, &types.VFBankContractMetadata{
        MinDenom: ...,
    }, &banktypes.Metadata{
        ...
    })
    ```
- Compare with `x/erc20` module by Evmos:

| Feature                                                 | Evmos `x/erc20` contract                                  | Dymension `VFBC`                                                        |
|---------------------------------------------------------|-----------------------------------------------------------|-------------------------------------------------------------------------|
| Interactive using wallet like Metamask                  | 🔥 Yes                                                    | 🔥 Yes                                                                  |
| Assets                                                  | ERC-20 representation of native bank assets, must convert | 🔥 Native bank assets, no need convert                                  |
| Asset actual balance                                    | = sum(bank balance + ERC-20 balance)                      | 🔥 = bank balance = ERC-20 balance                                      |
| Support direct transfer (`transfer`)                    | 🔥 Yes                                                    | 🔥 Yes                                                                  |
| Support authorized transfer (`transferFrom`)            | 🔥 Yes                                                    | No                                                                      |
| Support converting ERC-20 token into native token (IBC) | 🔥 Yes                                                    | No                                                                      |
| New contract deployment                                 | gov _(before v17), automatically (from v17)_              | gov, _can be automatically deploy upon new bank denom metadata created_ |
| Interact-able within EVM execution                      | 🔥 Yes                                                    | No                                                                      |