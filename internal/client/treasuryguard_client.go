package client

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type RequestState struct {
	ID              uint64
	Token           common.Address
	To              common.Address
	Amount          *big.Int
	CreatedBy       common.Address
	Approvals       uint64
	ApprovalsNeeded uint64
	CreatedAt       uint64
	EarliestExec    uint64
	ExpiresAt       uint64
	Status          uint8
}

func unpackRequest(values []interface{}) (RequestState, error) {
	if len(values) != 11 {
		return RequestState{}, fmt.Errorf("unexpected request fields")
	}

	id, ok := values[0].(*big.Int)
	if !ok {
		return RequestState{}, fmt.Errorf("invalid id type")
	}
	token, ok := values[1].(common.Address)
	if !ok {
		return RequestState{}, fmt.Errorf("invalid token type")
	}
	to, ok := values[2].(common.Address)
	if !ok {
		return RequestState{}, fmt.Errorf("invalid to type")
	}
	amount, ok := values[3].(*big.Int)
	if !ok {
		return RequestState{}, fmt.Errorf("invalid amount type")
	}
	createdBy, ok := values[4].(common.Address)
	if !ok {
		return RequestState{}, fmt.Errorf("invalid createdBy type")
	}
	approvals, ok := values[5].(*big.Int)
	if !ok {
		return RequestState{}, fmt.Errorf("invalid approvals type")
	}
	approvalsNeeded, ok := values[6].(*big.Int)
	if !ok {
		return RequestState{}, fmt.Errorf("invalid approvalsNeeded type")
	}
	createdAt, ok := asUint64(values[7])
	if !ok {
		return RequestState{}, fmt.Errorf("invalid createdAt type")
	}
	earliestExec, ok := asUint64(values[8])
	if !ok {
		return RequestState{}, fmt.Errorf("invalid earliestExec type")
	}
	expiresAt, ok := asUint64(values[9])
	if !ok {
		return RequestState{}, fmt.Errorf("invalid expiresAt type")
	}
	status, ok := values[10].(uint8)
	if !ok {
		if statusBig, ok := values[10].(*big.Int); ok {
			status = uint8(statusBig.Uint64())
		} else {
			return RequestState{}, fmt.Errorf("invalid status type")
		}
	}

	return RequestState{
		ID:              id.Uint64(),
		Token:           token,
		To:              to,
		Amount:          amount,
		CreatedBy:       createdBy,
		Approvals:       approvals.Uint64(),
		ApprovalsNeeded: approvalsNeeded.Uint64(),
		CreatedAt:       createdAt,
		EarliestExec:    earliestExec,
		ExpiresAt:       expiresAt,
		Status:          status,
	}, nil
}

func asUint64(v any) (uint64, bool) {
	switch t := v.(type) {
	case uint64:
		return t, true
	case uint32:
		return uint64(t), true
	case uint:
		return uint64(t), true
	case int64:
		if t < 0 {
			return 0, false
		}
		return uint64(t), true
	case *big.Int:
		if t == nil || t.Sign() < 0 || t.BitLen() > 64 {
			return 0, false
		}
		return t.Uint64(), true
	default:
		return 0, false
	}
}
