package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"base-treasury-guard/internal/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

type EthClient struct {
	rpc         *ethclient.Client
	ws          *ethclient.Client
	wsURL       string
	contract    common.Address
	abi         abi.ABI
	log         *zap.Logger
	guardianKey string
	executorKey string
	chainID     *big.Int
	mu          sync.Mutex
}

func New(cfg config.Config, log *zap.Logger) (*EthClient, error) {
	if log == nil {
		log = zap.NewNop()
	}
	if !common.IsHexAddress(cfg.ContractAddress) {
		return nil, fmt.Errorf("invalid contract address")
	}

	rpc, err := ethclient.Dial(cfg.RPCUrl)
	if err != nil {
		return nil, err
	}

	parsed, err := ParseTreasuryGuardABI()
	if err != nil {
		rpc.Close()
		return nil, err
	}

	chainID := new(big.Int).SetUint64(cfg.ChainID)

	client := &EthClient{
		rpc:         rpc,
		wsURL:       cfg.WSUrl,
		contract:    common.HexToAddress(cfg.ContractAddress),
		abi:         parsed,
		log:         log,
		guardianKey: cfg.GuardianKey,
		executorKey: cfg.ExecutorKey,
		chainID:     chainID,
	}

	if err := client.dialWS(); err != nil {
		rpc.Close()
		return nil, err
	}

	return client, nil
}

func (c *EthClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ws != nil {
		c.ws.Close()
		c.ws = nil
	}
	if c.rpc != nil {
		c.rpc.Close()
		c.rpc = nil
	}
}

func (c *EthClient) CheckChainID(ctx context.Context, expected uint64) (uint64, error) {
	id, err := c.rpc.ChainID(ctx)
	if err != nil {
		return 0, err
	}
	if id.Uint64() != expected {
		return id.Uint64(), fmt.Errorf("chain id mismatch: got %d expected %d", id.Uint64(), expected)
	}
	return id.Uint64(), nil
}

func (c *EthClient) ChainTime(ctx context.Context) (uint64, error) {
	header, err := c.rpc.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, err
	}
	return header.Time, nil
}

func (c *EthClient) Approve(ctx context.Context, id uint64) (common.Hash, error) {
	data, err := c.abi.Pack("approve", new(big.Int).SetUint64(id))
	if err != nil {
		return common.Hash{}, err
	}
	return c.sendTx(ctx, c.guardianKey, data, 120000)
}

func (c *EthClient) ExecuteBatch(ctx context.Context, ids []uint64, gasFloor uint64) (common.Hash, error) {
	packedIDs := make([]*big.Int, 0, len(ids))
	for _, id := range ids {
		packedIDs = append(packedIDs, new(big.Int).SetUint64(id))
	}
	data, err := c.abi.Pack("executeBatch", packedIDs, new(big.Int).SetUint64(gasFloor))
	if err != nil {
		return common.Hash{}, err
	}
	return c.sendTx(ctx, c.executorKey, data, 800000)
}

func (c *EthClient) GetRequest(ctx context.Context, id uint64) (RequestState, error) {
	data, err := c.abi.Pack("requests", new(big.Int).SetUint64(id))
	if err != nil {
		return RequestState{}, err
	}

	msg := ethereum.CallMsg{To: &c.contract, Data: data}
	res, err := c.rpc.CallContract(ctx, msg, nil)
	if err != nil {
		return RequestState{}, err
	}

	decoded, err := c.abi.Unpack("requests", res)
	if err != nil {
		return RequestState{}, err
	}
	if len(decoded) != 11 {
		return RequestState{}, fmt.Errorf("unexpected request fields")
	}

	return unpackRequest(decoded)
}

func (c *EthClient) sendTx(ctx context.Context, keyHex string, data []byte, gasLimit uint64) (common.Hash, error) {
	keyHex = strings.TrimPrefix(keyHex, "0x")
	priv, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return common.Hash{}, err
	}
	from := crypto.PubkeyToAddress(priv.PublicKey)

	nonce, err := c.rpc.PendingNonceAt(ctx, from)
	if err != nil {
		return common.Hash{}, err
	}

	tryDynamic := true
	var tipCap *big.Int
	if tip, err := c.rpc.SuggestGasTipCap(ctx); err == nil {
		tipCap = tip
	} else {
		tryDynamic = false
	}

	if tryDynamic {
		feeCap := new(big.Int)
		if price, err := c.rpc.SuggestGasPrice(ctx); err == nil {
			feeCap.Set(price)
		} else {
			feeCap.Set(tipCap)
			feeCap.Mul(feeCap, big.NewInt(2))
		}
		if feeCap.Cmp(tipCap) < 0 {
			feeCap.Set(tipCap)
			feeCap.Mul(feeCap, big.NewInt(2))
		}

		tx := types.NewTx(&types.DynamicFeeTx{
			ChainID:   c.chainID,
			Nonce:     nonce,
			To:        &c.contract,
			Gas:       gasLimit,
			GasTipCap: tipCap,
			GasFeeCap: feeCap,
			Value:     big.NewInt(0),
			Data:      data,
		})

		signed, err := types.SignTx(tx, types.LatestSignerForChainID(c.chainID), priv)
		if err == nil {
			err = c.rpc.SendTransaction(ctx, signed)
		}
		if err == nil {
			return signed.Hash(), nil
		}
		if isNonceTooLow(err) {
			return c.retryWithNonce(ctx, priv, gasLimit, data, true)
		}
		return common.Hash{}, err
	}

	return c.sendLegacy(ctx, priv, nonce, gasLimit, data)
}

func (c *EthClient) retryWithNonce(ctx context.Context, priv *ecdsa.PrivateKey, gasLimit uint64, data []byte, dynamic bool) (common.Hash, error) {
	from := crypto.PubkeyToAddress(priv.PublicKey)
	nonce, err := c.rpc.PendingNonceAt(ctx, from)
	if err != nil {
		return common.Hash{}, err
	}
	if dynamic {
		return c.sendDynamic(ctx, priv, nonce, gasLimit, data)
	}
	return c.sendLegacy(ctx, priv, nonce, gasLimit, data)
}

func (c *EthClient) sendDynamic(ctx context.Context, priv *ecdsa.PrivateKey, nonce uint64, gasLimit uint64, data []byte) (common.Hash, error) {
	tipCap, err := c.rpc.SuggestGasTipCap(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	feeCap := new(big.Int)
	if price, err := c.rpc.SuggestGasPrice(ctx); err == nil {
		feeCap.Set(price)
	} else {
		feeCap.Set(tipCap)
		feeCap.Mul(feeCap, big.NewInt(2))
	}
	if feeCap.Cmp(tipCap) < 0 {
		feeCap.Set(tipCap)
		feeCap.Mul(feeCap, big.NewInt(2))
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   c.chainID,
		Nonce:     nonce,
		To:        &c.contract,
		Gas:       gasLimit,
		GasTipCap: tipCap,
		GasFeeCap: feeCap,
		Value:     big.NewInt(0),
		Data:      data,
	})

	signed, err := types.SignTx(tx, types.LatestSignerForChainID(c.chainID), priv)
	if err != nil {
		return common.Hash{}, err
	}
	if err := c.rpc.SendTransaction(ctx, signed); err != nil {
		return common.Hash{}, err
	}
	return signed.Hash(), nil
}

func (c *EthClient) sendLegacy(ctx context.Context, priv *ecdsa.PrivateKey, nonce uint64, gasLimit uint64, data []byte) (common.Hash, error) {
	price, err := c.rpc.SuggestGasPrice(ctx)
	if err != nil {
		return common.Hash{}, err
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &c.contract,
		Gas:      gasLimit,
		GasPrice: price,
		Value:    big.NewInt(0),
		Data:     data,
	})

	signed, err := types.SignTx(tx, types.LatestSignerForChainID(c.chainID), priv)
	if err != nil {
		return common.Hash{}, err
	}
	if err := c.rpc.SendTransaction(ctx, signed); err != nil {
		if isNonceTooLow(err) {
			return c.retryWithNonce(ctx, priv, gasLimit, data, false)
		}
		return common.Hash{}, err
	}
	return signed.Hash(), nil
}

func isNonceTooLow(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "nonce too low")
}

func (c *EthClient) dialWS() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ws != nil {
		c.ws.Close()
		c.ws = nil
	}
	ws, err := ethclient.Dial(c.wsURL)
	if err != nil {
		return err
	}
	c.ws = ws
	return nil
}
