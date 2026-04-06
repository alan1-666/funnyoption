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

    IERC20Minimal public immutable collateralToken;
    address public immutable operator;
    address public rollupCore;

    mapping(address => uint256) public depositedBalance;
    mapping(bytes32 => bool) public processedClaims;
    mapping(bytes32 => ClaimRecord) public processedClaimRecords;

    event Deposited(address indexed wallet, uint256 amount);
    event WithdrawalQueued(bytes32 indexed withdrawalId, address indexed wallet, uint256 amount, address recipient);
    event ClaimProcessed(bytes32 indexed claimId, address indexed wallet, uint256 amount, address recipient);
    event RollupCoreUpdated(address indexed rollupCore);

    error OnlyOperator();
    error OnlyAuthorizedClaimer();
    error InvalidAmount();
    error ClaimAlreadyProcessed();
    error InvalidRollupCore();

    constructor(address collateralToken_, address operator_) {
        collateralToken = IERC20Minimal(collateralToken_);
        operator = operator_;
    }

    function setRollupCore(address rollupCore_) external {
        if (msg.sender != operator) revert OnlyOperator();
        if (rollupCore_ == address(0)) revert InvalidRollupCore();
        rollupCore = rollupCore_;
        emit RollupCoreUpdated(rollupCore_);
    }

    function deposit(uint256 amount) external {
        if (amount == 0) revert InvalidAmount();
        depositedBalance[msg.sender] += amount;
        require(collateralToken.transferFrom(msg.sender, address(this), amount), "TRANSFER_FROM_FAILED");
        emit Deposited(msg.sender, amount);
    }

    function queueWithdrawal(bytes32 withdrawalId, uint256 amount, address recipient) external {
        if (amount == 0) revert InvalidAmount();
        if (depositedBalance[msg.sender] < amount) revert InvalidAmount();
        depositedBalance[msg.sender] -= amount;
        emit WithdrawalQueued(withdrawalId, msg.sender, amount, recipient);
    }

    function processClaim(bytes32 claimId, address wallet, uint256 amount, address recipient) external {
        if (msg.sender != operator && msg.sender != rollupCore) revert OnlyAuthorizedClaimer();
        if (processedClaims[claimId]) revert ClaimAlreadyProcessed();
        if (amount == 0) revert InvalidAmount();

        processedClaims[claimId] = true;
        processedClaimRecords[claimId] = ClaimRecord({wallet: wallet, amount: amount, recipient: recipient});
        require(collateralToken.transfer(recipient, amount), "TRANSFER_FAILED");
        emit ClaimProcessed(claimId, wallet, amount, recipient);
    }
}
