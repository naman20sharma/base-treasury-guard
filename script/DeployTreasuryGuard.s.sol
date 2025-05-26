// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";
import "../contracts/TreasuryGuard.sol";

contract DeployTreasuryGuard is Script {
    function run() external {
        vm.startBroadcast();
        new TreasuryGuard();
        vm.stopBroadcast();
    }
}
