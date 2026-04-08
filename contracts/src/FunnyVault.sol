// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IERC20Minimal {
    function transferFrom(address from, address to, uint256 value) external returns (bool);
    function transfer(address to, uint256 value) external returns (bool);
}

contract FunnyVault {
    struct ClaimRecord {
        address wallet;
        uint256 amount;
        address recipient;
    }

    struct PendingClaim {
        address wallet;
        uint256 amount;
        address recipient;
        uint256 executeAfter;
        bool cancelled;
    }

    IERC20Minimal public immutable collateralToken;
    address public immutable operator;
    address public rollupCore;

    uint256 public operatorEpochClaimCap;
    uint256 public currentEpochStart;
    uint256 public currentEpochClaimed;
    uint256 public constant EPOCH_DURATION = 1 days;

    uint256 public timelockDelay;
    uint256 public timelockThreshold;

    uint256 private _reentrancyStatus;
    uint256 private constant _NOT_ENTERED = 1;
    uint256 private constant _ENTERED = 2;

    mapping(address => uint256) public depositedBalance;
    mapping(bytes32 => bool) public processedClaims;
    mapping(bytes32 => ClaimRecord) public processedClaimRecords;
    mapping(bytes32 => PendingClaim) public pendingClaims;

    event Deposited(address indexed wallet, uint256 amount);
    event WithdrawalQueued(bytes32 indexed withdrawalId, address indexed wallet, uint256 amount, address recipient);
    event ClaimProcessed(bytes32 indexed claimId, address indexed wallet, uint256 amount, address recipient);
    event RollupCoreUpdated(address indexed rollupCore);
    event OperatorEpochClaimCapUpdated(uint256 cap);
    event TimelockConfigUpdated(uint256 threshold, uint256 delay);
    event ClaimQueued(bytes32 indexed claimId, address indexed wallet, uint256 amount, address recipient, uint256 executeAfter);
    event QueuedClaimCancelled(bytes32 indexed claimId);

    error OnlyOperator();
    error OnlyAuthorizedClaimer();
    error InvalidAmount();
    error ClaimAlreadyProcessed();
    error InvalidRollupCore();
    error OperatorEpochCapExceeded();
    error ClaimTimelocked();
    error ClaimNotReady();
    error ClaimNotQueued();
    error ClaimCancelled();
    error ReentrantCall();

    modifier nonReentrant() {
        if (_reentrancyStatus == _ENTERED) revert ReentrantCall();
        _reentrancyStatus = _ENTERED;
        _;
        _reentrancyStatus = _NOT_ENTERED;
    }

    constructor(address collateralToken_, address operator_) {
        collateralToken = IERC20Minimal(collateralToken_);
        operator = operator_;
        _reentrancyStatus = _NOT_ENTERED;
    }

    function setRollupCore(address rollupCore_) external {
        if (msg.sender != operator) revert OnlyOperator();
        if (rollupCore_ == address(0)) revert InvalidRollupCore();
        rollupCore = rollupCore_;
        emit RollupCoreUpdated(rollupCore_);
    }

    function setOperatorEpochClaimCap(uint256 cap) external {
        if (msg.sender != operator) revert OnlyOperator();
        operatorEpochClaimCap = cap;
        emit OperatorEpochClaimCapUpdated(cap);
    }

    function setTimelockConfig(uint256 threshold, uint256 delay) external {
        if (msg.sender != operator) revert OnlyOperator();
        timelockThreshold = threshold;
        timelockDelay = delay;
        emit TimelockConfigUpdated(threshold, delay);
    }

    function deposit(uint256 amount) external nonReentrant {
        if (amount == 0) revert InvalidAmount();
        require(collateralToken.transferFrom(msg.sender, address(this), amount), "TRANSFER_FROM_FAILED");
        depositedBalance[msg.sender] += amount;
        emit Deposited(msg.sender, amount);
    }

    function queueWithdrawal(bytes32 withdrawalId, uint256 amount, address recipient) external {
        if (amount == 0) revert InvalidAmount();
        if (depositedBalance[msg.sender] < amount) revert InvalidAmount();
        depositedBalance[msg.sender] -= amount;
        emit WithdrawalQueued(withdrawalId, msg.sender, amount, recipient);
    }

    function processClaim(bytes32 claimId, address wallet, uint256 amount, address recipient) external nonReentrant {
        if (msg.sender != operator && msg.sender != rollupCore) revert OnlyAuthorizedClaimer();
        if (processedClaims[claimId]) revert ClaimAlreadyProcessed();
        if (amount == 0) revert InvalidAmount();

        if (msg.sender == operator) {
            if (operatorEpochClaimCap > 0) {
                uint256 epochStart = (block.timestamp / EPOCH_DURATION) * EPOCH_DURATION;
                if (epochStart != currentEpochStart) {
                    currentEpochStart = epochStart;
                    currentEpochClaimed = 0;
                }
                if (currentEpochClaimed + amount > operatorEpochClaimCap) revert OperatorEpochCapExceeded();
                currentEpochClaimed += amount;
            }

            if (timelockThreshold > 0 && amount >= timelockThreshold && timelockDelay > 0) {
                if (pendingClaims[claimId].executeAfter == 0) {
                    revert ClaimTimelocked();
                }
            }
        }

        if (pendingClaims[claimId].executeAfter > 0) {
            PendingClaim storage pending = pendingClaims[claimId];
            if (pending.cancelled) revert ClaimCancelled();
            if (block.timestamp < pending.executeAfter) revert ClaimNotReady();
            if (pending.wallet != wallet || pending.amount != amount || pending.recipient != recipient) revert InvalidAmount();
        }

        processedClaims[claimId] = true;
        processedClaimRecords[claimId] = ClaimRecord({wallet: wallet, amount: amount, recipient: recipient});
        require(collateralToken.transfer(recipient, amount), "TRANSFER_FAILED");
        emit ClaimProcessed(claimId, wallet, amount, recipient);
    }

    function queueClaim(bytes32 claimId, address wallet, uint256 amount, address recipient) external {
        if (msg.sender != operator) revert OnlyOperator();
        if (processedClaims[claimId]) revert ClaimAlreadyProcessed();
        if (amount == 0) revert InvalidAmount();
        if (pendingClaims[claimId].executeAfter > 0) revert ClaimAlreadyProcessed();

        uint256 executeAfter = block.timestamp + timelockDelay;
        pendingClaims[claimId] = PendingClaim({
            wallet: wallet,
            amount: amount,
            recipient: recipient,
            executeAfter: executeAfter,
            cancelled: false
        });
        emit ClaimQueued(claimId, wallet, amount, recipient, executeAfter);
    }

    function cancelQueuedClaim(bytes32 claimId) external {
        if (msg.sender != operator) revert OnlyOperator();
        if (pendingClaims[claimId].executeAfter == 0) revert ClaimNotQueued();
        pendingClaims[claimId].cancelled = true;
        emit QueuedClaimCancelled(claimId);
    }
}
