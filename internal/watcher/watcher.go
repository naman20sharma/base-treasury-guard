package watcher

import (
    "context"

    "base-treasury-guard/internal/client"
    "base-treasury-guard/internal/config"

    "github.com/ethereum/go-ethereum/common"
)

type Logger interface {
    Info(msg string, args ...any)
    Error(msg string, args ...any)
}

type Watcher struct {
    cfg config.Config
    log Logger
}

func New(cfg config.Config, log Logger) *Watcher {
    return &Watcher{cfg: cfg, log: log}
}

func (w *Watcher) Run(ctx context.Context) error {
    eth, err := client.NewEthClient(w.cfg.RPCURL, w.cfg.WSURL, w.log)
    if err != nil {
        return err
    }
    defer eth.Close()

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
            w.log.Info(
                "request created",
                "id", ev.ID,
                "token", ev.Token.Hex(),
                "to", ev.To.Hex(),
                "amount", ev.Amount,
                "approvalsNeeded", ev.ApprovalsNeeded,
                "createdBy", ev.CreatedBy.Hex(),
                "earliestExec", ev.EarliestExec,
            )
        }
    }
}
