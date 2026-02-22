package ethereum

import (
	"io"
	"strings"
)

// Minimal ABIs for Uniswap V2 Router02 and ERC20 — only the methods we call.

func mustRouterABI() io.Reader {
	return strings.NewReader(`[
		{
			"name": "swapExactTokensForETH",
			"type": "function",
			"stateMutability": "nonpayable",
			"inputs": [
				{"name": "amountIn",      "type": "uint256"},
				{"name": "amountOutMin",  "type": "uint256"},
				{"name": "path",          "type": "address[]"},
				{"name": "to",            "type": "address"},
				{"name": "deadline",      "type": "uint256"}
			],
			"outputs": [
				{"name": "amounts", "type": "uint256[]"}
			]
		},
		{
			"name": "swapExactETHForTokens",
			"type": "function",
			"stateMutability": "payable",
			"inputs": [
				{"name": "amountOutMin",  "type": "uint256"},
				{"name": "path",          "type": "address[]"},
				{"name": "to",            "type": "address"},
				{"name": "deadline",      "type": "uint256"}
			],
			"outputs": [
				{"name": "amounts", "type": "uint256[]"}
			]
		}
	]`)
}

func mustERC20ABI() io.Reader {
	return strings.NewReader(`[
		{
			"name": "balanceOf",
			"type": "function",
			"stateMutability": "view",
			"inputs": [{"name": "_owner", "type": "address"}],
			"outputs": [{"name": "balance", "type": "uint256"}]
		},
		{
			"name": "allowance",
			"type": "function",
			"stateMutability": "view",
			"inputs": [
				{"name": "_owner",   "type": "address"},
				{"name": "_spender", "type": "address"}
			],
			"outputs": [{"name": "", "type": "uint256"}]
		},
		{
			"name": "approve",
			"type": "function",
			"stateMutability": "nonpayable",
			"inputs": [
				{"name": "_spender", "type": "address"},
				{"name": "_value",   "type": "uint256"}
			],
			"outputs": [{"name": "", "type": "bool"}]
		}
	]`)
}
