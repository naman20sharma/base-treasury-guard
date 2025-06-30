package watcher

import (
    "context"
    "math/big"
)

type BatchExecutor struct {
    gasFloor uint64
    gasPerTx uint64
}

func NewBatchExecutor(gasFloor, gasPerTx uint64) *BatchExecutor {
    return &BatchExecutor{gasFloor: gasFloor, gasPerTx: gasPerTx}
}

func (b *BatchExecutor) Select(ids []*big.Int) []*big.Int {
    if len(ids) == 0 {
        return nil
    }
    gasRemaining := b.gasPerTx * uint64(len(ids))
    selected := make([]*big.Int, 0, len(ids))
    for _, id := range ids {
        if gasRemaining < b.gasFloor {
            break
        }
        selected = append(selected, id)
        if gasRemaining >= b.gasPerTx {
            gasRemaining -= b.gasPerTx
        } else {
            gasRemaining = 0
        }
    }
    return selected
}

func (b *BatchExecutor) Execute(ctx context.Context, ids []*big.Int, exec func(context.Context, []*big.Int) error) error {
    selected := b.Select(ids)
    if len(selected) == 0 {
        return nil
    }
    return exec(ctx, selected)
}
