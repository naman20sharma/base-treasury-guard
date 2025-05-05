// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {AccessControl} from "openzeppelin-contracts/contracts/access/AccessControl.sol";
import {Pausable} from "openzeppelin-contracts/contracts/utils/Pausable.sol";
import {ReentrancyGuard} from "openzeppelin-contracts/contracts/security/ReentrancyGuard.sol";
import {IERC20} from "openzeppelin-contracts/contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "openzeppelin-contracts/contracts/token/ERC20/utils/SafeERC20.sol";

contract TreasuryGuard is AccessControl, Pausable, ReentrancyGuard {
    using SafeERC20 for IERC20;

    bytes32 public constant TREASURER_ROLE = keccak256("TREASURER_ROLE");
    bytes32 public constant GUARDIAN_ROLE = keccak256("GUARDIAN_ROLE");
    bytes32 public constant EXECUTOR_ROLE = keccak256("EXECUTOR_ROLE");

    enum RequestStatus {
        Pending,
        Cancelled,
        Executed,
        Expired
    }

    struct PayoutRequest {
        uint256 id;
        address token;
        address to;
        uint256 amount;
        address createdBy;
        uint256 approvals;
        uint256 approvalsNeeded;
        uint64 createdAt;
        uint64 earliestExec;
        uint64 expiresAt;
        RequestStatus status;
    }

    uint64 public minDelay;
    uint64 public maxPendingDuration;
    uint256 public maxPerTx;
    uint8 public maxApprovalsNeeded;

    uint256 public nextRequestId;
    mapping(uint256 => PayoutRequest) public requests;
    mapping(uint256 => mapping(address => bool)) public approvalsByGuardian;
    mapping(address => bool) public tokenAllowlist;

    event RequestCreated(
        uint256 indexed id,
        address indexed token,
        address indexed to,
        uint256 amount,
        uint256 approvalsNeeded,
        address createdBy,
        uint64 earliestExec
    );
    event RequestApproved(uint256 indexed id, address indexed guardian, uint256 approvalsCount);
    event RequestCancelled(uint256 indexed id, address indexed cancelledBy);
    event RequestExpired(uint256 indexed id, address indexed expiredBy);
    event RequestExecuted(uint256 indexed id, address indexed executor, uint256 gasUsed);
    event BatchExecuted(uint256[] idsProcessed, address indexed executor, uint256 gasUsed);
    event ParamsUpdated(uint64 minDelay, uint64 maxPendingDuration, uint256 maxPerTx, uint8 maxApprovalsNeeded);
    event TokenAllowlistUpdated(address indexed token, bool allowed);

    constructor(address admin, uint64 defaultMinDelay, uint256 defaultMaxPerTx) {
        address owner = admin == address(0) ? msg.sender : admin;

        _grantRole(DEFAULT_ADMIN_ROLE, owner);
        _grantRole(TREASURER_ROLE, owner);
        _grantRole(GUARDIAN_ROLE, owner);
        _grantRole(EXECUTOR_ROLE, owner);

        minDelay = defaultMinDelay == 0 ? 10 minutes : defaultMinDelay;
        maxPendingDuration = 7 days;
        maxApprovalsNeeded = 10;
        maxPerTx = defaultMaxPerTx;

        emit ParamsUpdated(minDelay, maxPendingDuration, maxPerTx, maxApprovalsNeeded);
    }

    function setParams(
        uint64 newMinDelay,
        uint64 newMaxPendingDuration,
        uint256 newMaxPerTx,
        uint8 newMaxApprovalsNeeded
    ) external onlyRole(DEFAULT_ADMIN_ROLE) {
        require(newMaxPendingDuration > 0, "PENDING_DURATION_ZERO");
        require(newMaxApprovalsNeeded > 0, "APPROVALS_CAP_ZERO");

        minDelay = newMinDelay;
        maxPendingDuration = newMaxPendingDuration;
        maxPerTx = newMaxPerTx;
        maxApprovalsNeeded = newMaxApprovalsNeeded;

        emit ParamsUpdated(newMinDelay, newMaxPendingDuration, newMaxPerTx, newMaxApprovalsNeeded);
    }

    function setTokenAllowed(address token, bool allowed) external onlyRole(DEFAULT_ADMIN_ROLE) {
        tokenAllowlist[token] = allowed;
        emit TokenAllowlistUpdated(token, allowed);
    }

    function pause() external onlyRole(GUARDIAN_ROLE) {
        _pause();
    }

    function unpause() external onlyRole(DEFAULT_ADMIN_ROLE) {
        _unpause();
    }

    function createRequest(
        address token,
        address to,
        uint256 amount,
        uint256 approvalsNeeded
    ) external whenNotPaused onlyRole(TREASURER_ROLE) returns (uint256 id) {
        require(token == address(0) || tokenAllowlist[token], "TOKEN_NOT_ALLOWED");
        if (maxPerTx > 0) {
            require(amount <= maxPerTx, "AMOUNT_EXCEEDS_CAP");
        }
        require(approvalsNeeded > 0, "APPROVALS_REQUIRED");
        require(approvalsNeeded <= maxApprovalsNeeded, "APPROVALS_EXCEEDS_CAP");
        require(to != address(0), "INVALID_RECIPIENT");

        id = nextRequestId++;

        uint64 createdAt = uint64(block.timestamp);
        uint64 earliestExec = createdAt + minDelay;
        uint64 expiresAt = createdAt + maxPendingDuration;

        requests[id] = PayoutRequest({
            id: id,
            token: token,
            to: to,
            amount: amount,
            createdBy: msg.sender,
            approvals: 0,
            approvalsNeeded: approvalsNeeded,
            createdAt: createdAt,
            earliestExec: earliestExec,
            expiresAt: expiresAt,
            status: RequestStatus.Pending
        });

        emit RequestCreated(id, token, to, amount, approvalsNeeded, msg.sender, earliestExec);
    }

    function approve(uint256 id) external whenNotPaused onlyRole(GUARDIAN_ROLE) {
        PayoutRequest storage request = _getRequest(id);
        require(request.status == RequestStatus.Pending, "NOT_PENDING");
        require(block.timestamp <= request.expiresAt, "REQUEST_EXPIRED");
        require(!approvalsByGuardian[id][msg.sender], "ALREADY_APPROVED");

        approvalsByGuardian[id][msg.sender] = true;
        request.approvals += 1;

        emit RequestApproved(id, msg.sender, request.approvals);
    }

    function cancel(uint256 id) external whenNotPaused {
        PayoutRequest storage request = _getRequest(id);
        require(
            hasRole(TREASURER_ROLE, msg.sender) || hasRole(DEFAULT_ADMIN_ROLE, msg.sender),
            "NOT_AUTHORIZED"
        );
        require(request.status == RequestStatus.Pending, "NOT_PENDING");
        require(block.timestamp <= request.expiresAt, "REQUEST_EXPIRED");

        request.status = RequestStatus.Cancelled;
        emit RequestCancelled(id, msg.sender);
    }

    function expire(uint256 id) external whenNotPaused {
        PayoutRequest storage request = _getRequest(id);
        require(request.status == RequestStatus.Pending, "NOT_PENDING");
        require(block.timestamp > request.expiresAt, "NOT_EXPIRED");

        request.status = RequestStatus.Expired;
        emit RequestExpired(id, msg.sender);
    }

    function execute(uint256 id) external whenNotPaused onlyRole(EXECUTOR_ROLE) nonReentrant {
        uint256 gasStart = gasleft();
        _executeInternal(id);
        uint256 gasUsed = gasStart - gasleft();
        emit RequestExecuted(id, msg.sender, gasUsed);
    }

    function executeBatch(uint256[] calldata ids, uint256 gasFloor)
        external
        whenNotPaused
        onlyRole(EXECUTOR_ROLE)
        nonReentrant
    {
        uint256 gasStart = gasleft();
        uint256[] memory processed = new uint256[](ids.length);
        uint256 processedCount;

        for (uint256 i = 0; i < ids.length; i++) {
            if (gasleft() < gasFloor) {
                break;
            }

            uint256 id = ids[i];
            if (!_isReady(id)) {
                continue;
            }

            try this.executeFromBatch(id, msg.sender) {
                processed[processedCount++] = id;
            } catch {
                continue;
            }
        }

        uint256[] memory trimmed = new uint256[](processedCount);
        for (uint256 i = 0; i < processedCount; i++) {
            trimmed[i] = processed[i];
        }

        uint256 gasUsed = gasStart - gasleft();
        emit BatchExecuted(trimmed, msg.sender, gasUsed);
    }

    function executeFromBatch(uint256 id, address executor) external {
        require(msg.sender == address(this), "ONLY_SELF");
        _executeInternal(id);
        emit RequestExecuted(id, executor, 0);
    }

    function _executeInternal(uint256 id) internal {
        PayoutRequest storage request = _getRequest(id);
        require(request.status == RequestStatus.Pending, "NOT_PENDING");
        require(block.timestamp <= request.expiresAt, "REQUEST_EXPIRED");
        require(request.approvals >= request.approvalsNeeded, "INSUFFICIENT_APPROVALS");
        require(block.timestamp >= request.earliestExec, "DELAY_NOT_MET");
        require(_hasSufficientBalance(request.token, request.amount), "INSUFFICIENT_BALANCE");

        request.status = RequestStatus.Executed;

        if (request.token == address(0)) {
            _safeTransferETH(request.to, request.amount);
        } else {
            IERC20(request.token).safeTransfer(request.to, request.amount);
        }
    }

    function _getRequest(uint256 id) internal view returns (PayoutRequest storage request) {
        require(id < nextRequestId, "INVALID_ID");
        request = requests[id];
    }

    function _isReady(uint256 id) internal view returns (bool) {
        if (id >= nextRequestId) {
            return false;
        }

        PayoutRequest storage request = requests[id];
        if (request.status != RequestStatus.Pending) {
            return false;
        }
        if (block.timestamp > request.expiresAt) {
            return false;
        }
        if (request.approvals < request.approvalsNeeded) {
            return false;
        }
        if (block.timestamp < request.earliestExec) {
            return false;
        }
        if (!_hasSufficientBalance(request.token, request.amount)) {
            return false;
        }

        return true;
    }

    function _hasSufficientBalance(address token, uint256 amount) internal view returns (bool) {
        if (token == address(0)) {
            return address(this).balance >= amount;
        }
        return IERC20(token).balanceOf(address(this)) >= amount;
    }

    function _safeTransferETH(address to, uint256 amount) internal {
        (bool success, ) = to.call{value: amount}("");
        require(success, "ETH_TRANSFER_FAILED");
    }

    receive() external payable {}
}
