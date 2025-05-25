// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../contracts/TreasuryGuard.sol";

contract TreasuryGuardTest is Test {
    TreasuryGuard guard;

    function setUp() public {
        guard = new TreasuryGuard();
    }

    function testOwnerIsDeployer() public {
        assertEq(guard.owner(), address(this));
    }
}
