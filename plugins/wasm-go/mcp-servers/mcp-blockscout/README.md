# Blockscout MCP Server

Blockscout is an open-source blockchain explorer for inspecting and analyzing EVM-compatible blockchains. This MCP server for Blockscout wraps the official Blockscout multi-chain API, exposing blockchain data - balances, tokens, NFTs, contract metadata - via MCP so that AI agents and tools can access and analyze it contextually.

- [Blockscout Website](https://blockscout.com/)
- [MCP Server Plugin GitHub repo](https://github.com/blockscout/mcp-server-plugin)

## Features

Blockscout MCP Server provides the following features:

- **Chain Information**: Get a list of supported blockchains.
- **Address and Contract Analysis**: Resolve ENS names, get address balances, contract details, and retrieve contract ABIs.
- **Token and NFT Data**: Look up tokens by symbol, and get detailed information on token holdings and NFTs for a given address.
- **Transaction and Block History**: Retrieve information on blocks, get transaction history for an address, and get detailed information about specific transactions, including human-readable summaries.
- **Event Logs**: Get decoded event logs for transactions or addresses.

## Usage Guide

The Blockscout MCP Server is in Beta and currently does not require authentication. This is subject to change.

### Generate URL

Log in to the [Higress MCP Marketplace](https://mcp.higress.ai) and generate a URL for either Streamable HTTP or SSE.

### Configure MCP Client

In your MCP Client configuration, add the following to the MCP Server list:

```json
"mcpServers": {
    "mcp-blockscout": {
      "url": "https://mcp.higress.ai/mcp-blockscout/{user-specific-id}/sse"
    }
}
```

or in case of Streamable HTTP:

```json
"mcpServers": {
    "mcp-blockscout": {
      "url": "https://mcp.higress.ai/mcp-blockscout/{user-specific-id}"
    }
}
```

## Supported Tools

The following tools are available in the multi-chain configuration.

1.  `__get_instructions__()` - Must be called before any other tool. Initializes the MCP server session.
2.  `get_chains_list()` - Gets the list of supported blockchain chains and their IDs.
3.  `get_address_by_ens_name(name)` - Converts an ENS domain name to its corresponding Ethereum address.
4.  `lookup_token_by_symbol(chain_id, symbol)` - Searches for token addresses by symbol or name.
5.  `get_contract_abi(chain_id, address)` - Retrieves the ABI (Application Binary Interface) for a smart contract.
6.  `get_address_info(chain_id, address)` - Gets comprehensive information about an address (balance, contract status, etc.).
7.  `get_tokens_by_address(chain_id, address)` - Returns detailed ERC20 token holdings for an address.
8.  `get_latest_block(chain_id)` - Returns the latest indexed block number and timestamp.
9.  `get_transactions_by_address(chain_id, address, age_from, age_to, methods)` - Gets native currency transfers and smart contract interactions for an address.
10. `get_token_transfers_by_address(chain_id, address, age_from, age_to, token)` - Returns ERC-20 token transfers for an address.
11. `transaction_summary(chain_id, transaction_hash)` - Provides a human-readable summary of a transaction.
12. `nft_tokens_by_address(chain_id, address)` - Retrieves NFT tokens owned by an address.
13. `get_block_info(chain_id, number_or_hash)` - Returns block information (timestamp, gas used, etc.).
14. `get_transaction_info(chain_id, transaction_hash)` - Gets comprehensive transaction information.
15. `get_transaction_logs(chain_id, transaction_hash)` - Returns transaction logs with decoded event data.
16. `get_address_logs(chain_id, address)` - Gets logs emitted by a specific address with decoded event data.

## Example Prompts

```plaintext
On which popular networks is `ens.eth` deployed as a contract?
```

```plaintext
What are the usual activities performed by `ens.eth` on the Ethereum Mainnet?
Since it is a contract, what is the most used functionality of this contract?
Which address interacts with the contract the most?
```

```plaintext
Calculate the total gas fees paid on Ethereum by address `0xcafe...cafe` in May 2025.
```

```plaintext
Which 10 most recent logs were emitted by `0xFe89cc7aBB2C4183683ab71653C4cdc9B02D44b7`
before `Nov 08 2024 04:21:35 AM (-06:00 UTC)`?
```
