package client

import (
    "context"
    "time"

    "github.com/ethereum/go-ethereum"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/ethclient"
)

type Logger interface {
    Info(msg string, fields ...any)
    Error(msg string, fields ...any)
}

type EthClient struct {
    rpc   *ethclient.Client
    ws    *ethclient.Client
    log   Logger
    guard *TreasuryGuardClient
}

type nopLogger struct{}

func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}

func NewEthClient(rpcURL, wsURL string, log Logger) (*EthClient, error) {
    if log == nil {
        log = nopLogger{}
    }
    rpc, err := ethclient.Dial(rpcURL)
    if err != nil {
        log.Error("failed to connect to rpc", "err", err)
        return nil, err
    }
    ws, err := ethclient.Dial(wsURL)
    if err != nil {
        log.Error("failed to connect to ws", "err", err)
        return nil, err
    }
    return &EthClient{rpc: rpc, ws: ws, log: log}, nil
}

func (c *EthClient) Close() {
    if c.rpc != nil {
        c.rpc.Close()
    }
    if c.ws != nil {
        c.ws.Close()
    }
}

func (c *EthClient) SubscribeRequestCreated(ctx context.Context, contract common.Address, sink chan<- *RequestCreatedEvent) error {
    guard, err := NewTreasuryGuardClient(contract)
    if err != nil {
        return err
    }
    c.guard = guard

    for {
        if err := c.subscribeOnce(ctx, guard, contract, sink); err != nil {
            if ctx.Err() != nil {
                return ctx.Err()
            }
            c.log.Error("subscription error", "err", err)
            time.Sleep(2 * time.Second)
            continue
        }
        return nil
    }
}

func (c *EthClient) subscribeOnce(ctx context.Context, guard *TreasuryGuardClient, contract common.Address, sink chan<- *RequestCreatedEvent) error {
    logsCh := make(chan types.Log)
    query := ethereum.FilterQuery{
        Addresses: []common.Address{contract},
        Topics:    [][]common.Hash{{guard.EventID("RequestCreated")}},
    }

    sub, err := c.ws.SubscribeFilterLogs(ctx, query, logsCh)
    if err != nil {
        return err
    }

    for {
        select {
        case <-ctx.Done():
            sub.Unsubscribe()
            return ctx.Err()
        case err := <-sub.Err():
            return err
        case log := <-logsCh:
            event, err := guard.ParseRequestCreated(log)
            if err != nil {
                c.log.Error("failed to parse RequestCreated", "err", err)
                continue
            }
            select {
            case sink <- event:
            default:
                c.log.Error("event channel full", "event", "RequestCreated")
            }
        }
    }
}
