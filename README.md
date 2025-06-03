# Base Treasury Guard
Role based spending limits with off chain policy enforcement on Base L2.

## Overview
Base Treasury Guard is a small on‑chain treasury controller plus a Go daemon. The smart contract (`TreasuryGuard`) holds ETH and ERC20, and only executes payouts after a treasurer creates a request, guardians approve it, and a time delay passes. The daemon (`guardd`) watches events, enforces off‑chain policy (amount limits, token allowlist), approves requests, and batches executions to reduce gas.

## Architecture
**On chain** handles custody, approvals, and execution guarantees. **Off chain** handles policy decisions and operational automation.

```
[treasurer/guardian/executor keys]
            |
            v
   +-------------------+            +--------------------+
   |  guardd (Go)      |<---------->| Base L2 (RPC/WS)   |
   |  policy + batching|            | TreasuryGuard.sol  |
   +-------------------+            +--------------------+
            |                                   |
            | events (RequestCreated, ...)      | ETH/ERC20 held
            +-----------------------------------+
```

## Quick start
### Prerequisites
- Foundry (forge)
- Go 1.22+
- RPC endpoint (local Anvil or Base Sepolia)
- Env vars (see `.env.example` comment in `internal/config/config.go`)

### Run tests
```
forge test -vv
```

### Deploy to local Anvil
Start Anvil:
```
anvil
```
Export env vars (example):
```
export PRIVATE_KEY=0x...
export TREASURER_ADDR=0x...
export GUARDIAN_ADDR=0x...
export EXECUTOR_ADDR=0x...
```
Deploy:
```
forge script script/DeployTreasuryGuard.s.sol:DeployTreasuryGuard \
  --rpc-url http://127.0.0.1:8545 --broadcast
```

### Run guardd against local chain
Set required env vars (example):
```
export RPC_URL=http://127.0.0.1:8545
export WS_URL=ws://127.0.0.1:8545
export CHAIN_ID=31337
export CONTRACT_ADDRESS=0x...
export GUARDIAN_KEY=0x...
export EXECUTOR_KEY=0x...
```
Run:
```
go run ./cmd/guardd
```
Metrics are served at `HTTP_LISTEN_ADDR` (default `127.0.0.1:9000`) on `/metrics`.

## How it works
- **Request creation**: A treasurer submits a payout request (token, recipient, amount, approvals needed). The contract stores it and emits `RequestCreated`.
- **Approvals**: Guardians approve once each. The daemon can auto‑approve if policy checks pass.
- **Delay and execution**: Requests can only execute after `minDelay` has passed and approvals meet threshold.
- **Batch execution and gas floor**: The daemon groups ready requests and calls `executeBatch`, stopping early if gas remaining drops below `gasFloor`.

## Base mainnet deployment
- Contract address: 0xa7C570d2AD90f6c9Af1f45e6f8462A672115A052
- BaseScan contract: https://basescan.org/address/0xa7C570d2AD90f6c9Af1f45e6f8462A672115A052
- BaseScan transaction: https://basescan.org/tx/0xc256792f3830d364cabe7ecb86674e170fc1e9c701a30f97953e345a195fa25a

## Interview notes
- Role separation makes the control plane explicit: treasurer proposes, guardian approves, executor executes.
- Off‑chain policy is flexible and auditable without weakening on‑chain safety.
- Gas‑aware batching cuts execution overhead while keeping single‑request execution available.
