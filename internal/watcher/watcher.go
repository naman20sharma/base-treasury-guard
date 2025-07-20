package watcher

import (
	"context"
	"math/big"
	"time"

	"base-treasury-guard/internal/client"
	"base-treasury-guard/internal/config"
	"base-treasury-guard/internal/metrics"

	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

type Watcher struct {
	cfg               config.Config
	log               *zap.Logger
	metrics           *metrics.Registry
	allowedTokens     map[common.Address]struct{}
	maxAmount         *big.Int
	execCooldownUntil map[uint64]time.Time
}

type requestClient interface {
	ChainTime(ctx context.Context) (uint64, error)
	GetRequest(ctx context.Context, id uint64) (client.RequestState, error)
}

func New(cfg config.Config, log *zap.Logger, metrics *metrics.Registry) *Watcher {
	if log == nil {
		log = zap.NewNop()
	}
	w := &Watcher{cfg: cfg, log: log, metrics: metrics, execCooldownUntil: make(map[uint64]time.Time)}
	w.allowedTokens = make(map[common.Address]struct{})
	for _, token := range cfg.PolicyAllowedTokens {
		if common.IsHexAddress(token) {
			w.allowedTokens[common.HexToAddress(token)] = struct{}{}
		}
	}
	if cfg.PolicyMaxAmount != "" && cfg.PolicyMaxAmount != "0" {
		if amt, ok := new(big.Int).SetString(cfg.PolicyMaxAmount, 10); ok {
			w.maxAmount = amt
		}
	}
	return w
}

func (w *Watcher) Run(ctx context.Context) error {
	ethClient, err := client.New(w.cfg, w.log)
	if err != nil {
		return err
	}
	defer ethClient.Close()

	chainID, err := ethClient.CheckChainID(ctx, w.cfg.ChainID)
	if err != nil {
		w.log.Error("chain id mismatch", zap.Error(err))
		return err
	}
	w.log.Info("connected", zap.Uint64("chain_id", chainID), zap.String("contract", w.cfg.ContractAddress))

	events, errs := ethClient.SubscribeRequestCreated(ctx)
	active := make(map[uint64]struct{})
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("watcher stopped")
			return nil
		case err := <-errs:
			if err != nil {
				w.log.Error("subscription error", zap.Error(err))
			}
		case evt, ok := <-events:
			if !ok {
				return nil
			}
			if evt.ID == nil || !evt.ID.IsUint64() {
				continue
			}
			id := evt.ID.Uint64()
			active[id] = struct{}{}

			req, err := ethClient.GetRequest(ctx, id)
			if err != nil {
				w.log.Error("request fetch failed", zap.Error(err))
				w.metrics.IncFailures()
				continue
			}
			if !w.policyAllows(req) {
				w.log.Info("request rejected",
					zap.Uint64("id", id),
					zap.String("token", req.Token.Hex()),
					zap.String("amount", req.Amount.String()),
				)
				continue
			}

			hash, err := ethClient.Approve(ctx, id)
			if err != nil {
				w.metrics.IncFailures()
				w.log.Error("approve failed", zap.Uint64("id", id), zap.Error(err))
				continue
			}
			w.metrics.IncApprovals()
			w.log.Info("approve sent", zap.Uint64("id", id), zap.String("tx", hash.Hex()))
		case <-ticker.C:
			batch := w.buildReadyBatch(ctx, ethClient, active)
			if len(batch) == 0 {
				continue
			}
			hash, err := ethClient.ExecuteBatch(ctx, batch, w.cfg.GasFloor)
			if err != nil {
				w.metrics.IncFailures()
				w.log.Error("execute batch failed", zap.Error(err))
				continue
			}
			for _, id := range batch {
				w.execCooldownUntil[id] = time.Now().Add(30 * time.Second)
			}
			w.metrics.IncExecutions()
			w.log.Info("execute batch sent", zap.Int("count", len(batch)), zap.String("tx", hash.Hex()))
		}
	}
}

func (w *Watcher) buildReadyBatch(ctx context.Context, ethClient requestClient, active map[uint64]struct{}) []uint64 {
	now, err := ethClient.ChainTime(ctx)
	if err != nil {
		w.log.Error("chain time fetch failed", zap.Error(err))
		w.metrics.IncFailures()
		return nil
	}
	batch := make([]uint64, 0, w.cfg.MaxBatch)
	for id := range active {
		req, err := ethClient.GetRequest(ctx, id)
		if err != nil {
			w.log.Error("request fetch failed", zap.Uint64("id", id), zap.Error(err))
			w.metrics.IncFailures()
			continue
		}
		if req.Status != 0 {
			delete(active, id)
			delete(w.execCooldownUntil, id)
			w.log.Info("request finalized", zap.Uint64("id", id), zap.Uint8("status", req.Status))
			continue
		}
		if until, ok := w.execCooldownUntil[id]; ok && time.Now().Before(until) {
			continue
		}
		if req.ApprovalsNeeded > 0 && req.Approvals < req.ApprovalsNeeded {
			continue
		}
		if now < req.EarliestExec {
			continue
		}
		if req.ExpiresAt > 0 && now > req.ExpiresAt {
			continue
		}
		batch = append(batch, id)
		if len(batch) >= w.cfg.MaxBatch {
			break
		}
	}
	return batch
}

func (w *Watcher) policyAllows(req client.RequestState) bool {
	if w.maxAmount != nil && req.Amount.Cmp(w.maxAmount) > 0 {
		return false
	}
	if len(w.allowedTokens) == 0 {
		return true
	}
	_, ok := w.allowedTokens[req.Token]
	return ok
}
