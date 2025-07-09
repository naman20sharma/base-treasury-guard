package client

import (
    "math/big"
    "strings"
    "testing"

    "github.com/ethereum/go-ethereum/accounts/abi"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
)

func TestParseRequestCreated(t *testing.T) {
    parsed, err := abi.JSON(strings.NewReader(TreasuryGuardABI))
    if err != nil {
        t.Fatalf("parse abi: %v", err)
    }

    event := parsed.Events["RequestCreated"]
    id := big.NewInt(7)
    token := common.HexToAddress("0x1111111111111111111111111111111111111111")
    to := common.HexToAddress("0x2222222222222222222222222222222222222222")
    amount := big.NewInt(42)
    approvals := big.NewInt(2)
    createdBy := common.HexToAddress("0x3333333333333333333333333333333333333333")
    earliest := uint64(123456)

    data, err := event.Inputs.NonIndexed().Pack(amount, approvals, createdBy, earliest)
    if err != nil {
        t.Fatalf("pack: %v", err)
    }

    log := types.Log{
        Topics: []common.Hash{
            event.ID,
            common.BigToHash(id),
            common.BytesToHash(token.Bytes()),
            common.BytesToHash(to.Bytes()),
        },
        Data: data,
    }

    client, err := NewTreasuryGuardClient(common.HexToAddress("0x0000000000000000000000000000000000000001"))
    if err != nil {
        t.Fatalf("client: %v", err)
    }

    parsedEvent, err := client.ParseRequestCreated(log)
    if err != nil {
        t.Fatalf("parse: %v", err)
    }

    if parsedEvent.ID.Cmp(id) != 0 {
        t.Fatalf("id mismatch")
    }
    if parsedEvent.Token != token {
        t.Fatalf("token mismatch")
    }
    if parsedEvent.To != to {
        t.Fatalf("to mismatch")
    }
    if parsedEvent.Amount.Cmp(amount) != 0 {
        t.Fatalf("amount mismatch")
    }
    if parsedEvent.ApprovalsNeeded.Cmp(approvals) != 0 {
        t.Fatalf("approvals mismatch")
    }
    if parsedEvent.CreatedBy != createdBy {
        t.Fatalf("createdBy mismatch")
    }
    if parsedEvent.EarliestExec != earliest {
        t.Fatalf("earliest mismatch")
    }
}
