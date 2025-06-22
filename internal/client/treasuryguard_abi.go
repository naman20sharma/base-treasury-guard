package client

const TreasuryGuardABI = `[
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
  }
]`
