package client

import (
    "math/big"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
)

type RequestCreatedEvent struct {
    ID              *big.Int
    Token           common.Address
    To              common.Address
    Amount          *big.Int
    ApprovalsNeeded *big.Int
    CreatedBy       common.Address
    EarliestExec    uint64
    Raw             types.Log
}
