// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";
import "forge-std/console.sol";
import "../contracts/TreasuryGuard.sol";

contract CreateRequest is Script {
    struct Log {
        bytes32[] topics;
        bytes data;
        address emitter;
    }

    function run() external {
        uint256 privateKey = vm.envUint("EXECUTOR_KEY");
        address sender = vm.addr(privateKey);

        address guardAddr = vm.envAddress("CONTRACT_ADDRESS");
        address to = vm.envAddress("REQUEST_TO");
        uint256 amount = vm.envUint("REQUEST_AMOUNT_WEI");
        uint256 approvalsNeeded = vm.envUint("REQUEST_APPROVALS_NEEDED");
        address token = _readOptionalAddress("REQUEST_TOKEN", address(0));

        TreasuryGuard guard = TreasuryGuard(payable(guardAddr));

        vm.recordLogs();
        vm.startBroadcast(privateKey);
        uint256 id = guard.createRequest(token, to, amount, approvalsNeeded);
        vm.stopBroadcast();

        console.log("contract", guardAddr);
        console.log("sender", sender);
        console.log("requestId", id);

        bytes32 sig = keccak256(
            "RequestCreated(uint256,address,address,uint256,uint256,address,uint64)"
        );
        Log[] memory logs = abi.decode(abi.encode(vm.getRecordedLogs()), (Log[]));
        for (uint256 i = 0; i < logs.length; i++) {
            Log memory lg = logs[i];
            if (lg.topics.length == 0) {
                continue;
            }
            if (lg.topics[0] != sig) {
                continue;
            }
            if (lg.topics.length > 1) {
                uint256 emittedId = uint256(lg.topics[1]);
                console.log("eventRequestId", emittedId);
            }
            (uint256 emittedAmount, uint256 emittedApprovals, address createdBy, uint64 earliestExec) = abi
                .decode(lg.data, (uint256, uint256, address, uint64));
            emittedAmount;
            emittedApprovals;
            createdBy;
            console.log("earliestExec", uint256(earliestExec));
            break;
        }
    }

    function _readOptionalAddress(string memory key, address fallbackValue) internal view returns (address) {
        try vm.envAddress(key) returns (address value) {
            return value;
        } catch {
            return fallbackValue;
        }
    }
}
