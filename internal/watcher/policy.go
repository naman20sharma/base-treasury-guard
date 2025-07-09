package watcher

import (
    "math/big"
    "strings"
    "sync"
    "time"

    "github.com/ethereum/go-ethereum/common"
)

type Policy struct {
    allowlist map[common.Address]struct{}
    maxAmount *big.Int
    cooldown  time.Duration
    now       func() time.Time
    mu        sync.Mutex
    lastSeen  map[string]time.Time
}

type PolicyRequest struct {
    Token     common.Address
    To        common.Address
    Amount    *big.Int
    CreatedBy common.Address
}

func NewPolicy(allowlistCSV string, maxAmountWei string, cooldown time.Duration, now func() time.Time) *Policy {
    allowlist := make(map[common.Address]struct{})
    if allowlistCSV != "" {
        for _, entry := range strings.Split(allowlistCSV, ",") {
            cleaned := strings.TrimSpace(entry)
            if cleaned == "" {
                continue
            }
            allowlist[common.HexToAddress(cleaned)] = struct{}{}
        }
    }

    var maxAmount *big.Int
    if maxAmountWei != "" {
        parsed, ok := new(big.Int).SetString(maxAmountWei, 10)
        if ok {
            maxAmount = parsed
        }
    }

    if now == nil {
        now = time.Now
    }

    return &Policy{
        allowlist: allowlist,
        maxAmount: maxAmount,
        cooldown:  cooldown,
        now:       now,
        lastSeen:  make(map[string]time.Time),
    }
}

func (p *Policy) Check(req PolicyRequest) (bool, string) {
    if len(p.allowlist) > 0 {
        if _, ok := p.allowlist[req.Token]; !ok {
            return false, "token_not_allowed"
        }
    }

    if p.maxAmount != nil && req.Amount != nil {
        if req.Amount.Cmp(p.maxAmount) > 0 {
            return false, "amount_exceeds_limit"
        }
    }

    if p.cooldown > 0 {
        key := req.Token.Hex() + ":" + req.To.Hex() + ":" + req.CreatedBy.Hex()
        now := p.now()

        p.mu.Lock()
        defer p.mu.Unlock()

        if last, ok := p.lastSeen[key]; ok {
            if now.Sub(last) < p.cooldown {
                return false, "cooldown_active"
            }
        }
        p.lastSeen[key] = now
    }

    return true, ""
}
