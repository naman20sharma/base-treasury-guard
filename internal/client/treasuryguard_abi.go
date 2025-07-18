package client

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const treasuryGuardABIJSON = `[
  {
    "anonymous": false,
    "inputs": [
      {"indexed": true, "internalType": "uint256", "name": "id", "type": "uint256"},
      {"indexed": true, "internalType": "address", "name": "token", "type": "address"},
      {"indexed": true, "internalType": "address", "name": "to", "type": "address"},
      {"indexed": false, "internalType": "uint256", "name": "amount", "type": "uint256"},
      {"indexed": false, "internalType": "uint256", "name": "approvalsNeeded", "type": "uint256"},
      {"indexed": false, "internalType": "address", "name": "createdBy", "type": "address"},
      {"indexed": false, "internalType": "uint64", "name": "earliestExec", "type": "uint64"}
    ],
    "name": "RequestCreated",
    "type": "event"
  },
  {
    "inputs": [
      {"internalType": "uint256", "name": "id", "type": "uint256"}
    ],
    "name": "approve",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType": "uint256[]", "name": "ids", "type": "uint256[]"},
      {"internalType": "uint256", "name": "gasFloor", "type": "uint256"}
    ],
    "name": "executeBatch",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType": "uint256", "name": "", "type": "uint256"}
    ],
    "name": "requests",
    "outputs": [
      {"internalType": "uint256", "name": "id", "type": "uint256"},
      {"internalType": "address", "name": "token", "type": "address"},
      {"internalType": "address", "name": "to", "type": "address"},
      {"internalType": "uint256", "name": "amount", "type": "uint256"},
      {"internalType": "address", "name": "createdBy", "type": "address"},
      {"internalType": "uint256", "name": "approvals", "type": "uint256"},
      {"internalType": "uint256", "name": "approvalsNeeded", "type": "uint256"},
      {"internalType": "uint64", "name": "createdAt", "type": "uint64"},
      {"internalType": "uint64", "name": "earliestExec", "type": "uint64"},
      {"internalType": "uint64", "name": "expiresAt", "type": "uint64"},
      {"internalType": "uint8", "name": "status", "type": "uint8"}
    ],
    "stateMutability": "view",
    "type": "function"
  }
]`

func ParseTreasuryGuardABI() (abi.ABI, error) {
	return abi.JSON(strings.NewReader(treasuryGuardABIJSON))
}
