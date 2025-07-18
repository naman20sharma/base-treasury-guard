package watcher

import (
	"context"
	"math/big"
	"testing"
	"time"

	"base-treasury-guard/internal/client"
	"base-treasury-guard/internal/config"
	"base-treasury-guard/internal/metrics"

	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

type fakeClient struct {
	now uint64
	req client.RequestState
	err error
}

func (f *fakeClient) ChainTime(ctx context.Context) (uint64, error) {
	return f.now, f.err
}

func (f *fakeClient) GetRequest(ctx context.Context, id uint64) (client.RequestState, error) {
	if f.err != nil {
		return client.RequestState{}, f.err
	}
	return f.req, nil
}

func TestCooldownPreventsResubmit(t *testing.T) {
	cfg := config.Config{MaxBatch: 10}
	w := New(cfg, zap.NewNop(), metrics.NewRegistry("test"))

	req := client.RequestState{
		ID:              1,
		Token:           common.Address{},
		Amount:          big.NewInt(1),
		Approvals:       1,
		ApprovalsNeeded: 1,
		EarliestExec:    1,
		ExpiresAt:       1000,
		Status:          0,
	}
	fc := &fakeClient{now: 10, req: req}
	active := map[uint64]struct{}{1: {}}

	w.execCooldownUntil[1] = time.Now().Add(30 * time.Second)
	batch := w.buildReadyBatch(context.Background(), fc, active)
	if len(batch) != 0 {
		t.Fatalf("expected empty batch during cooldown")
	}

	w.execCooldownUntil[1] = time.Now().Add(-1 * time.Second)
	batch = w.buildReadyBatch(context.Background(), fc, active)
	if len(batch) != 1 || batch[0] != 1 {
		t.Fatalf("expected batch to include id after cooldown")
	}
}

func TestPolicyAllowsUnderMaxAmount(t *testing.T) {
	cfg := config.Config{PolicyMaxAmount: "100"}
	w := New(cfg, zap.NewNop(), metrics.NewRegistry("test"))
	req := client.RequestState{Amount: big.NewInt(100), Token: common.Address{}}
	if !w.policyAllows(req) {
		t.Fatalf("expected request under max amount to be allowed")
	}
}

func TestPolicyAllowlistEnforced(t *testing.T) {
	allowed := common.HexToAddress("0x1111111111111111111111111111111111111111")
	cfg := config.Config{PolicyAllowedTokens: []string{allowed.Hex()}}
	w := New(cfg, zap.NewNop(), metrics.NewRegistry("test"))

	bad := client.RequestState{Amount: big.NewInt(1), Token: common.HexToAddress("0x2222222222222222222222222222222222222222")}
	if w.policyAllows(bad) {
		t.Fatalf("expected non-allowlisted token to be rejected")
	}

	good := client.RequestState{Amount: big.NewInt(1), Token: allowed}
	if !w.policyAllows(good) {
		t.Fatalf("expected allowlisted token to be allowed")
	}
}
