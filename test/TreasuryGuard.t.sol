// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../contracts/TreasuryGuard.sol";

contract TreasuryGuardTest is Test {
    TreasuryGuard guard;
    address treasurer;
    address guardian;
    address executor;
    address recipient;
    address outsider;

    function setUp() public {
        guard = new TreasuryGuard(address(this), 1, 0);
        treasurer = makeAddr("treasurer");
        guardian = makeAddr("guardian");
        executor = makeAddr("executor");
        recipient = makeAddr("recipient");
        outsider = makeAddr("outsider");

        guard.grantRole(guard.TREASURER_ROLE(), treasurer);
        guard.grantRole(guard.GUARDIAN_ROLE(), guardian);
        guard.grantRole(guard.EXECUTOR_ROLE(), executor);
    }

    function testAdminIsDeployer() public {
        assertTrue(guard.hasRole(guard.DEFAULT_ADMIN_ROLE(), address(this)));
    }

    function testCreateApproveExecuteETH() public {
        uint256 amount = 1 ether;
        vm.deal(address(guard), amount);

        uint256 id = _createRequest(address(0), recipient, amount, 1);
        _approve(id);

        vm.warp(_requestEarliestExec(id));

        uint256 beforeBal = recipient.balance;
        vm.prank(executor);
        guard.execute(id);

        assertEq(uint256(_requestStatus(id)), uint256(TreasuryGuard.RequestStatus.Executed));
        assertEq(recipient.balance, beforeBal + amount);
    }

    function testExecuteRevertsBeforeDelay() public {
        uint256 amount = 1 ether;
        vm.deal(address(guard), amount);

        uint256 id = _createRequest(address(0), recipient, amount, 1);
        _approve(id);

        vm.prank(executor);
        vm.expectRevert(bytes("DELAY_NOT_MET"));
        guard.execute(id);
    }

    function testExecuteRevertsWithoutApprovals() public {
        uint256 amount = 1 ether;
        vm.deal(address(guard), amount);

        uint256 id = _createRequest(address(0), recipient, amount, 1);
        vm.warp(_requestEarliestExec(id));

        vm.prank(executor);
        vm.expectRevert(bytes("INSUFFICIENT_APPROVALS"));
        guard.execute(id);
    }

    function testCancelBlocksExecute() public {
        uint256 amount = 1 ether;
        vm.deal(address(guard), amount);

        uint256 id = _createRequest(address(0), recipient, amount, 1);
        _approve(id);

        vm.prank(treasurer);
        guard.cancel(id);

        vm.warp(_requestEarliestExec(id));

        vm.prank(executor);
        vm.expectRevert(bytes("NOT_PENDING"));
        guard.execute(id);
    }

    function testDoubleExecuteReverts() public {
        uint256 amount = 1 ether;
        vm.deal(address(guard), amount);

        uint256 id = _createRequest(address(0), recipient, amount, 1);
        _approve(id);

        vm.warp(_requestEarliestExec(id));

        vm.prank(executor);
        guard.execute(id);

        vm.prank(executor);
        vm.expectRevert(bytes("NOT_PENDING"));
        guard.execute(id);
    }

    function testOnlyTreasurerCanCreate() public {
        vm.prank(outsider);
        vm.expectRevert();
        guard.createRequest(address(0), recipient, 1 ether, 1);
    }

    function testOnlyGuardianCanApprove() public {
        uint256 id = _createRequest(address(0), recipient, 1 ether, 1);
        vm.prank(outsider);
        vm.expectRevert();
        guard.approve(id);
    }

    function testOnlyExecutorCanExecute() public {
        uint256 amount = 1 ether;
        vm.deal(address(guard), amount);

        uint256 id = _createRequest(address(0), recipient, amount, 1);
        _approve(id);

        vm.warp(_requestEarliestExec(id));

        vm.prank(outsider);
        vm.expectRevert();
        guard.execute(id);
    }

    function _createRequest(
        address token,
        address to,
        uint256 amount,
        uint256 approvalsNeeded
    ) internal returns (uint256) {
        vm.prank(treasurer);
        return guard.createRequest(token, to, amount, approvalsNeeded);
    }

    function _approve(uint256 id) internal {
        vm.prank(guardian);
        guard.approve(id);
    }

    function _requestEarliestExec(uint256 id) internal view returns (uint64 earliestExec) {
        (
            uint256 _id,
            address _token,
            address _to,
            uint256 _amount,
            address _createdBy,
            uint256 _approvals,
            uint256 _approvalsNeeded,
            uint64 _createdAt,
            uint64 earliest,
            uint64 _expiresAt,
            TreasuryGuard.RequestStatus _status
        ) = guard.requests(id);
        _id;
        _token;
        _to;
        _amount;
        _createdBy;
        _approvals;
        _approvalsNeeded;
        _createdAt;
        _expiresAt;
        _status;
        return earliest;
    }

    function _requestStatus(uint256 id) internal view returns (TreasuryGuard.RequestStatus status) {
        (
            uint256 _id,
            address _token,
            address _to,
            uint256 _amount,
            address _createdBy,
            uint256 _approvals,
            uint256 _approvalsNeeded,
            uint64 _createdAt,
            uint64 _earliestExec,
            uint64 _expiresAt,
            TreasuryGuard.RequestStatus st
        ) = guard.requests(id);
        _id;
        _token;
        _to;
        _amount;
        _createdBy;
        _approvals;
        _approvalsNeeded;
        _createdAt;
        _earliestExec;
        _expiresAt;
        return st;
    }
}
