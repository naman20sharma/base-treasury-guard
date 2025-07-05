package watcher

import (
    "math/big"
    "testing"
    "time"

    "github.com/ethereum/go-ethereum/common"
)

func TestPolicyAllowlist(t *testing.T) {
    tokenAllowed := common.HexToAddress("0x1111111111111111111111111111111111111111")
    tokenDenied := common.HexToAddress("0x2222222222222222222222222222222222222222")
    policy := NewPolicy(tokenAllowed.Hex(), "", 0, time.Now)

    ok, _ := policy.Check(PolicyRequest{Token: tokenDenied})
    if ok {
        t.Fatalf("expected token to be rejected")
    }

    ok, _ = policy.Check(PolicyRequest{Token: tokenAllowed})
    if !ok {
        t.Fatalf("expected token to be allowed")
    }
}

func TestPolicyMaxAmount(t *testing.T) {
    policy := NewPolicy("", "100", 0, time.Now)

    ok, _ := policy.Check(PolicyRequest{Amount: big.NewInt(101)})
    if ok {
        t.Fatalf("expected amount to be rejected")
    }

    ok, _ = policy.Check(PolicyRequest{Amount: big.NewInt(100)})
    if !ok {
        t.Fatalf("expected amount to be accepted")
    }
}

func TestPolicyCooldown(t *testing.T) {
    now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
    clock := func() time.Time { return now }

    token := common.HexToAddress("0x3333333333333333333333333333333333333333")
    to := common.HexToAddress("0x4444444444444444444444444444444444444444")
    createdBy := common.HexToAddress("0x5555555555555555555555555555555555555555")

    policy := NewPolicy(token.Hex(), "", time.Minute, clock)
    ok, _ := policy.Check(PolicyRequest{Token: token, To: to, CreatedBy: createdBy})
    if !ok {
        t.Fatalf("expected first request to pass")
    }

    ok, _ = policy.Check(PolicyRequest{Token: token, To: to, CreatedBy: createdBy})
    if ok {
        t.Fatalf("expected cooldown rejection")
    }

    now = now.Add(2 * time.Minute)
    ok, _ = policy.Check(PolicyRequest{Token: token, To: to, CreatedBy: createdBy})
    if !ok {
        t.Fatalf("expected cooldown to expire")
    }
}
