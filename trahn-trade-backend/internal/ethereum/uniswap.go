package ethereum

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const explorerTxPrefix = "https://etherscan.io/tx/"

// UniswapV2 wraps an Ethereum Client and provides Uniswap V2 Router swap methods.
type UniswapV2 struct {
	client       *Client
	routerAddr   common.Address
	wethAddr     common.Address
	quoteAddr    common.Address
	quoteSymbol  string
	quoteDec     int
	slippagePct  float64
	routerABI    abi.ABI
	erc20ABI     abi.ABI
}

func NewUniswapV2(
	client *Client,
	routerAddr, wethAddr, quoteAddr string,
	quoteSymbol string,
	quoteDecimals int,
	slippagePct float64,
) (*UniswapV2, error) {
	rABI, err := abi.JSON(mustRouterABI())
	if err != nil {
		return nil, fmt.Errorf("parse router ABI: %w", err)
	}
	eABI, err := abi.JSON(mustERC20ABI())
	if err != nil {
		return nil, fmt.Errorf("parse ERC20 ABI: %w", err)
	}
	return &UniswapV2{
		client:      client,
		routerAddr:  common.HexToAddress(routerAddr),
		wethAddr:    common.HexToAddress(wethAddr),
		quoteAddr:   common.HexToAddress(quoteAddr),
		quoteSymbol: quoteSymbol,
		quoteDec:    quoteDecimals,
		slippagePct: slippagePct,
		routerABI:   rABI,
		erc20ABI:    eABI,
	}, nil
}

func (u *UniswapV2) ExplorerURL(txHash string) string {
	return explorerTxPrefix + txHash
}

// TokenBalance returns the ERC20 token balance as a human-readable float.
func (u *UniswapV2) TokenBalance(ctx context.Context) (float64, error) {
	data, err := u.erc20ABI.Pack("balanceOf", u.client.wallet)
	if err != nil {
		return 0, err
	}
	result, err := u.client.CallContract(ctx, u.quoteAddr, data)
	if err != nil {
		return 0, fmt.Errorf("balanceOf call: %w", err)
	}
	bal := new(big.Int).SetBytes(result)
	divisor := math.Pow10(u.quoteDec)
	f, _ := new(big.Float).Quo(new(big.Float).SetInt(bal), new(big.Float).SetFloat64(divisor)).Float64()
	return f, nil
}

// ETHBalance returns wallet ETH balance as a human-readable float.
func (u *UniswapV2) ETHBalance(ctx context.Context) (float64, error) {
	bal, err := u.client.ETHBalance(ctx)
	if err != nil {
		return 0, err
	}
	f, _ := new(big.Float).Quo(new(big.Float).SetInt(bal), new(big.Float).SetFloat64(1e18)).Float64()
	return f, nil
}

// EnsureAllowance checks the router's allowance for the quote token and approves max if needed.
func (u *UniswapV2) EnsureAllowance(ctx context.Context, requiredAmount float64) error {
	data, err := u.erc20ABI.Pack("allowance", u.client.wallet, u.routerAddr)
	if err != nil {
		return err
	}
	result, err := u.client.CallContract(ctx, u.quoteAddr, data)
	if err != nil {
		return fmt.Errorf("allowance call: %w", err)
	}
	current := new(big.Int).SetBytes(result)

	requiredWei := toTokenWei(requiredAmount*2, u.quoteDec)
	if current.Cmp(requiredWei) >= 0 {
		return nil
	}

	fmt.Printf("Setting %s allowance for Uniswap Router...\n", u.quoteSymbol)
	maxUint256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	approveData, err := u.erc20ABI.Pack("approve", u.routerAddr, maxUint256)
	if err != nil {
		return err
	}

	txHash, err := u.client.SignAndSend(ctx, u.quoteAddr, big.NewInt(0), approveData)
	if err != nil {
		return fmt.Errorf("approve tx: %w", err)
	}
	fmt.Printf("Allowance TX confirmed: %s\n", u.ExplorerURL(txHash))
	return nil
}

// SwapUSDCForETH executes swapExactTokensForETH on the Uniswap V2 Router.
// Returns the transaction hash.
func (u *UniswapV2) SwapUSDCForETH(ctx context.Context, usdcAmount, minETHOut float64) (string, error) {
	if err := u.EnsureAllowance(ctx, usdcAmount); err != nil {
		return "", err
	}

	path := []common.Address{u.quoteAddr, u.wethAddr}
	deadline := big.NewInt(time.Now().Unix() + 20*60)
	amountIn := toTokenWei(usdcAmount, u.quoteDec)
	// Apply slippage to minETHOut
	minOutWei := toEthWei(minETHOut * (1 - u.slippagePct/100))

	data, err := u.routerABI.Pack("swapExactTokensForETH",
		amountIn, minOutWei, path, u.client.wallet, deadline)
	if err != nil {
		return "", fmt.Errorf("pack swapExactTokensForETH: %w", err)
	}

	return u.client.SignAndSend(ctx, u.routerAddr, big.NewInt(0), data)
}

// SwapETHForUSDC executes swapExactETHForTokens on the Uniswap V2 Router.
// Returns the transaction hash.
func (u *UniswapV2) SwapETHForUSDC(ctx context.Context, ethAmount float64) (string, error) {
	path := []common.Address{u.wethAddr, u.quoteAddr}
	deadline := big.NewInt(time.Now().Unix() + 20*60)
	value := toEthWei(ethAmount)

	data, err := u.routerABI.Pack("swapExactETHForTokens",
		big.NewInt(0), path, u.client.wallet, deadline)
	if err != nil {
		return "", fmt.Errorf("pack swapExactETHForTokens: %w", err)
	}

	return u.client.SignAndSend(ctx, u.routerAddr, value, data)
}

// GasCostETH estimates the gas cost for a transaction in ETH.
func (u *UniswapV2) GasCostETH(ctx context.Context) (float64, error) {
	gasPrice, err := u.client.GasPrice(ctx)
	if err != nil {
		return 0, err
	}
	cost := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(u.client.GasLimit()))
	f, _ := new(big.Float).Quo(new(big.Float).SetInt(cost), new(big.Float).SetFloat64(1e18)).Float64()
	return f, nil
}

// --- helpers ---

func toEthWei(eth float64) *big.Int {
	// eth * 1e18
	f := new(big.Float).Mul(new(big.Float).SetFloat64(eth), new(big.Float).SetFloat64(1e18))
	i, _ := f.Int(nil)
	return i
}

func toTokenWei(amount float64, decimals int) *big.Int {
	f := new(big.Float).Mul(
		new(big.Float).SetFloat64(amount),
		new(big.Float).SetFloat64(math.Pow10(decimals)),
	)
	i, _ := f.Int(nil)
	return i
}
