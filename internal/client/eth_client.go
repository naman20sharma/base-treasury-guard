package client

import (
    "context"
    "crypto/ecdsa"
    "math/big"
    "strings"
    "time"

    "github.com/ethereum/go-ethereum"
    "github.com/ethereum/go-ethereum/accounts/abi"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/crypto"
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

func (c *EthClient) ApproveRequest(ctx context.Context, contract common.Address, chainID *big.Int, guardianKey string, id *big.Int) (common.Hash, error) {
    data, err := encodeApproveRequest(id)
    if err != nil {
        return common.Hash{}, err
    }
    return c.sendTx(ctx, contract, chainID, guardianKey, data)
}

func (c *EthClient) ExecuteBatch(ctx context.Context, contract common.Address, chainID *big.Int, executorKey string, ids []*big.Int) (common.Hash, error) {
    data, err := encodeExecuteBatch(ids)
    if err != nil {
        return common.Hash{}, err
    }
    return c.sendTx(ctx, contract, chainID, executorKey, data)
}

func (c *EthClient) sendTx(ctx context.Context, contract common.Address, chainID *big.Int, rawKey string, data []byte) (common.Hash, error) {
    key, err := parseKey(rawKey)
    if err != nil {
        return common.Hash{}, err
    }
    from := crypto.PubkeyToAddress(key.PublicKey)

    nonce, err := c.rpc.PendingNonceAt(ctx, from)
    if err != nil {
        return common.Hash{}, err
    }
    gasPrice, err := c.rpc.SuggestGasPrice(ctx)
    if err != nil {
        return common.Hash{}, err
    }

    msg := ethereum.CallMsg{From: from, To: &contract, Data: data}
    gasLimit, err := c.rpc.EstimateGas(ctx, msg)
    if err != nil {
        return common.Hash{}, err
    }

    tx := types.NewTransaction(nonce, contract, big.NewInt(0), gasLimit, gasPrice, data)
    signer := types.LatestSignerForChainID(chainID)
    signed, err := types.SignTx(tx, signer, key)
    if err != nil {
        return common.Hash{}, err
    }

    if err := c.rpc.SendTransaction(ctx, signed); err != nil {
        return common.Hash{}, err
    }
    return signed.Hash(), nil
}

func parseKey(raw string) (*ecdsa.PrivateKey, error) {
    trimmed := strings.TrimPrefix(raw, "0x")
    return crypto.HexToECDSA(trimmed)
}

func encodeApproveRequest(id *big.Int) ([]byte, error) {
    methodID := crypto.Keccak256([]byte("approveRequest(uint256)"))[:4]
    args := abi.Arguments{{Type: mustType("uint256")}}
    packed, err := args.Pack(id)
    if err != nil {
        return nil, err
    }
    return append(methodID, packed...), nil
}

func encodeExecuteBatch(ids []*big.Int) ([]byte, error) {
    methodID := crypto.Keccak256([]byte("executeBatch(uint256[])"))[:4]
    args := abi.Arguments{{Type: mustType("uint256[]")}}
    packed, err := args.Pack(ids)
    if err != nil {
        return nil, err
    }
    return append(methodID, packed...), nil
}

func mustType(name string) abi.Type {
    typ, err := abi.NewType(name, "", nil)
    if err != nil {
        panic(err)
    }
    return typ
}
