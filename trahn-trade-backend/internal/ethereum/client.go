package ethereum

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	rpc        *ethclient.Client
	privateKey *ecdsa.PrivateKey
	wallet     common.Address
	chainID    *big.Int
	gasLimit   uint64
	gasMul     float64
}

func NewClient(rpcURL, privateKeyHex string, chainID int64, gasLimit int, gasMultiplier float64) (*Client, error) {
	rpc, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial RPC: %w", err)
	}

	pkHex := strings.TrimPrefix(privateKeyHex, "0x")
	pk, err := crypto.HexToECDSA(pkHex)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	addr := crypto.PubkeyToAddress(pk.PublicKey)

	return &Client{
		rpc:        rpc,
		privateKey: pk,
		wallet:     addr,
		chainID:    big.NewInt(chainID),
		gasLimit:   uint64(gasLimit),
		gasMul:     gasMultiplier,
	}, nil
}

func (c *Client) WalletAddress() common.Address { return c.wallet }
func (c *Client) GasLimit() uint64              { return c.gasLimit }
func (c *Client) Close()                         { c.rpc.Close() }

func (c *Client) ETHBalance(ctx context.Context) (*big.Int, error) {
	return c.rpc.BalanceAt(ctx, c.wallet, nil)
}

func (c *Client) GasPrice(ctx context.Context) (*big.Int, error) {
	price, err := c.rpc.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	// Apply multiplier
	mul := new(big.Float).SetFloat64(c.gasMul)
	adjusted := new(big.Float).Mul(new(big.Float).SetInt(price), mul)
	result, _ := adjusted.Int(nil)
	return result, nil
}

func (c *Client) Nonce(ctx context.Context) (uint64, error) {
	return c.rpc.PendingNonceAt(ctx, c.wallet)
}

// SignAndSend signs a legacy transaction and broadcasts it, returning the tx hash.
func (c *Client) SignAndSend(ctx context.Context, to common.Address, value *big.Int, data []byte) (string, error) {
	nonce, err := c.Nonce(ctx)
	if err != nil {
		return "", fmt.Errorf("get nonce: %w", err)
	}
	gasPrice, err := c.GasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("get gas price: %w", err)
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    value,
		Gas:      c.gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})

	signer := types.NewEIP155Signer(c.chainID)
	signed, err := types.SignTx(tx, signer, c.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign tx: %w", err)
	}

	if err := c.rpc.SendTransaction(ctx, signed); err != nil {
		return "", fmt.Errorf("send tx: %w", err)
	}

	return signed.Hash().Hex(), nil
}

// CallContract performs a read-only eth_call and returns the raw result.
func (c *Client) CallContract(ctx context.Context, to common.Address, data []byte) ([]byte, error) {
	msg := map[string]interface{}{
		"to":   to.Hex(),
		"data": fmt.Sprintf("0x%x", data),
	}
	var result string
	err := c.rpc.Client().CallContext(ctx, &result, "eth_call", msg, "latest")
	if err != nil {
		return nil, err
	}
	return common.FromHex(result), nil
}
