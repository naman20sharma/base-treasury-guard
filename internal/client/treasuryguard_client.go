package client

import (
    "math/big"
    "strings"

    "github.com/ethereum/go-ethereum/accounts/abi"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
)

type TreasuryGuardClient struct {
    address common.Address
    abi     abi.ABI
}

func NewTreasuryGuardClient(address common.Address) (*TreasuryGuardClient, error) {
    parsed, err := abi.JSON(strings.NewReader(TreasuryGuardABI))
    if err != nil {
        return nil, err
    }
    return &TreasuryGuardClient{address: address, abi: parsed}, nil
}

func (c *TreasuryGuardClient) Address() common.Address {
    return c.address
}

func (c *TreasuryGuardClient) EventID(name string) common.Hash {
    return c.abi.Events[name].ID
}

func (c *TreasuryGuardClient) ParseRequestCreated(log types.Log) (*RequestCreatedEvent, error) {
    var event RequestCreatedEvent
    if err := c.abi.UnpackIntoInterface(&event, "RequestCreated", log.Data); err != nil {
        return nil, err
    }
    if len(log.Topics) > 1 {
        event.ID = new(big.Int).SetBytes(log.Topics[1].Bytes())
    }
    if len(log.Topics) > 2 {
        event.Token = common.BytesToAddress(log.Topics[2].Bytes())
    }
    if len(log.Topics) > 3 {
        event.To = common.BytesToAddress(log.Topics[3].Bytes())
    }
    event.Raw = log
    return &event, nil
}
