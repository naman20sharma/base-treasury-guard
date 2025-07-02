package watcher

import (
    "context"
    "math/big"
    "time"

    "base-treasury-guard/internal/client"
    "base-treasury-guard/internal/config"
    "base-treasury-guard/internal/metrics"

    "github.com/ethereum/go-ethereum/common"
)

type Logger interface {
    Info(msg string, args ...any)
    Error(msg string, args ...any)
}

type Watcher struct {
    cfg     config.Config
    log     Logger
    metrics *metrics.Metrics
    policy  *Policy
    batch   *BatchExecutor
}

func New(cfg config.Config, log Logger, metrics *metrics.Metrics) *Watcher {
    cooldown := time.Duration(cfg.CooldownSeconds) * time.Second
    policy := NewPolicy(cfg.TokenAllowlist, cfg.MaxAmountWei, cooldown, time.Now)
    batch := NewBatchExecutor(cfg.GasFloor, cfg.GasPerTx)
    return &Watcher{cfg: cfg, log: log, metrics: metrics, policy: policy, batch: batch}
}

func (w *Watcher) Run(ctx context.Context) error {
    eth, err := client.NewEthClient(w.cfg.RPCURL, w.cfg.WSURL, w.log)
    if err != nil {
        return err
    }
    defer eth.Close()

    chainID, err := parseChainID(w.cfg.ChainID)
    if err != nil {
        return err
    }

    contract := common.HexToAddress(w.cfg.ContractAddress)
    events := make(chan *client.RequestCreatedEvent, 32)

    go func() {
        if err := eth.SubscribeRequestCreated(ctx, contract, events); err != nil {
            w.log.Error("subscription exited", "err", err)
        }
    }()

    for {
        select {
        case <-ctx.Done():
            return nil
        case ev := <-events:
            if ev == nil {
                continue
            }
            w.handleRequest(ctx, eth, chainID, contract, ev)
        }
    }
}

func (w *Watcher) handleRequest(ctx context.Context, eth *client.EthClient, chainID *big.Int, contract common.Address, ev *client.RequestCreatedEvent) {
    ok, reason := w.policy.Check(PolicyRequest{
        Token:     ev.Token,
        To:        ev.To,
        Amount:    ev.Amount,
        CreatedBy: ev.CreatedBy,
    })
    if !ok {
        w.metrics.IncFailures()
        w.log.Info("policy rejected request", "id", ev.ID, "reason", reason)
        return
    }

    if _, err := eth.ApproveRequest(ctx, contract, chainID, w.cfg.GuardianKey, ev.ID); err != nil {
        w.metrics.IncFailures()
        w.log.Error("approve failed", "id", ev.ID, "err", err)
        return
    }
    w.metrics.IncApprovals()

    if err := w.waitUntil(ctx, ev.EarliestExec); err != nil {
        w.metrics.IncFailures()
        w.log.Error("wait failed", "id", ev.ID, "err", err)
        return
    }

    ids := []*big.Int{ev.ID}
    if err := w.batch.Execute(ctx, ids, func(ctx context.Context, selected []*big.Int) error {
        if _, err := eth.ExecuteBatch(ctx, contract, chainID, w.cfg.ExecutorKey, selected); err != nil {
            return err
        }
        return nil
    }); err != nil {
        w.metrics.IncFailures()
        w.log.Error("execute batch failed", "id", ev.ID, "err", err)
        return
    }

    w.metrics.IncExecutions()
    w.log.Info("request executed", "id", ev.ID)
}

func (w *Watcher) waitUntil(ctx context.Context, earliest uint64) error {
    if earliest == 0 {
        return nil
    }
    target := time.Unix(int64(earliest), 0)
    if time.Now().After(target) {
        return nil
    }
    timer := time.NewTimer(time.Until(target))
    defer timer.Stop()

    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-timer.C:
        return nil
    }
}

func parseChainID(raw string) (*big.Int, error) {
    parsed, ok := new(big.Int).SetString(raw, 10)
    if !ok {
        return nil, errInvalidChainID(raw)
    }
    return parsed, nil
}

type errInvalidChainID string

func (e errInvalidChainID) Error() string {
    return "invalid chain id: " + string(e)
}
