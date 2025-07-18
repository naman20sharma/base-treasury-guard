package client

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type RequestCreatedEvent struct {
	ID     *big.Int
	Token  common.Address
	To     common.Address
	Amount *big.Int
}

func (c *EthClient) SubscribeRequestCreated(ctx context.Context) (<-chan RequestCreatedEvent, <-chan error) {
	out := make(chan RequestCreatedEvent)
	errCh := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errCh)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			evt, ok := c.abi.Events["RequestCreated"]
			if !ok {
				sendErr(errCh, errors.New("missing RequestCreated ABI"))
				return
			}

			if err := c.dialWS(); err != nil {
				sendErr(errCh, err)
				time.Sleep(5 * time.Second)
				continue
			}

			query := ethereum.FilterQuery{
				Addresses: []common.Address{c.contract},
				Topics:    [][]common.Hash{{evt.ID}},
			}

			logs := make(chan types.Log)
			sub, err := c.ws.SubscribeFilterLogs(ctx, query, logs)
			if err != nil {
				sendErr(errCh, err)
				time.Sleep(5 * time.Second)
				continue
			}

			for {
				select {
				case <-ctx.Done():
					sub.Unsubscribe()
					return
				case err := <-sub.Err():
					if err != nil {
						sendErr(errCh, err)
					}
					sub.Unsubscribe()
					time.Sleep(5 * time.Second)
					goto resubscribe
				case lg := <-logs:
					if lg.Removed {
						continue
					}
					evt, err := c.parseRequestCreated(lg)
					if err != nil {
						sendErr(errCh, err)
						continue
					}
					out <- evt
				}
			}
		resubscribe:
			continue
		}
	}()

	return out, errCh
}

func (c *EthClient) parseRequestCreated(lg types.Log) (RequestCreatedEvent, error) {
	if len(lg.Topics) < 4 {
		return RequestCreatedEvent{}, errors.New("invalid RequestCreated topics")
	}

	decoded, err := c.abi.Unpack("RequestCreated", lg.Data)
	if err != nil {
		return RequestCreatedEvent{}, err
	}
	amount, ok := decoded[0].(*big.Int)
	if !ok {
		return RequestCreatedEvent{}, errors.New("invalid amount type")
	}

	id := new(big.Int).SetBytes(lg.Topics[1].Bytes())
	token := common.BytesToAddress(lg.Topics[2].Bytes())
	to := common.BytesToAddress(lg.Topics[3].Bytes())

	return RequestCreatedEvent{
		ID:     id,
		Token:  token,
		To:     to,
		Amount: amount,
	}, nil
}

func sendErr(ch chan error, err error) {
	select {
	case ch <- err:
	default:
	}
}
