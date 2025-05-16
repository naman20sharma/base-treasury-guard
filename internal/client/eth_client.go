package client

import (
    "context"
    "math/big"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/ethclient"
    "go.uber.org/zap"
)

// EthClient wraps an ethclient with structured logging.
type EthClient struct {
    rpc *ethclient.Client
    log *zap.Logger
}

func NewEthClient(rpcURL string, log *zap.Logger) (*EthClient, error) {
    if log == nil {
        log = zap.NewNop()
    }
    rpc, err := ethclient.Dial(rpcURL)
    if err != nil {
        log.Error("failed to connect to Ethereum RPC", zap.Error(err))
        return nil, err
    }
    return &EthClient{rpc: rpc, log: log}, nil
}

func (c *EthClient) Close() {
    if c.rpc != nil {
        c.rpc.Close()
    }
}

func (c *EthClient) ChainID(ctx context.Context) (*big.Int, error) {
    id, err := c.rpc.ChainID(ctx)
    if err != nil {
        c.log.Error("failed to fetch chain id", zap.Error(err))
        return nil, err
    }
    return id, nil
}

func (c *EthClient) BalanceAt(ctx context.Context, account common.Address) (*big.Int, error) {
    bal, err := c.rpc.BalanceAt(ctx, account, nil)
    if err != nil {
        c.log.Error("failed to fetch balance", zap.Error(err), zap.String("address", account.Hex()))
        return nil, err
    }
    return bal, nil
}
