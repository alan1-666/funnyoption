// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract MockUSDT {
    string public constant name = "FunnyOption Mock USDT";
    string public constant symbol = "USDT";
    uint8 public constant decimals = 6;

    uint256 public totalSupply;
    address public immutable owner;

    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;
    mapping(address => bool) public minters;

    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    event MinterUpdated(address indexed minter, bool allowed);

    error OnlyOwner();
    error OnlyMinter();
    error InvalidReceiver();
    error InsufficientBalance();
    error InsufficientAllowance();

    constructor() {
        owner = msg.sender;
    }

    function addMinter(address minter) external {
        if (msg.sender != owner) revert OnlyOwner();
        minters[minter] = true;
        emit MinterUpdated(minter, true);
    }

    function removeMinter(address minter) external {
        if (msg.sender != owner) revert OnlyOwner();
        minters[minter] = false;
        emit MinterUpdated(minter, false);
    }

    function mint(address to, uint256 value) external returns (bool) {
        if (msg.sender != owner && !minters[msg.sender]) revert OnlyMinter();
        if (to == address(0)) revert InvalidReceiver();

        totalSupply += value;
        balanceOf[to] += value;
        emit Transfer(address(0), to, value);
        return true;
    }

    function approve(address spender, uint256 value) external returns (bool) {
        allowance[msg.sender][spender] = value;
        emit Approval(msg.sender, spender, value);
        return true;
    }

    function transfer(address to, uint256 value) external returns (bool) {
        _transfer(msg.sender, to, value);
        return true;
    }

    function transferFrom(address from, address to, uint256 value) external returns (bool) {
        uint256 allowed = allowance[from][msg.sender];
        if (allowed < value) revert InsufficientAllowance();
        if (allowed != type(uint256).max) {
            allowance[from][msg.sender] = allowed - value;
            emit Approval(from, msg.sender, allowance[from][msg.sender]);
        }
        _transfer(from, to, value);
        return true;
    }

    function _transfer(address from, address to, uint256 value) internal {
        if (to == address(0)) revert InvalidReceiver();
        uint256 fromBalance = balanceOf[from];
        if (fromBalance < value) revert InsufficientBalance();

        balanceOf[from] = fromBalance - value;
        balanceOf[to] += value;
        emit Transfer(from, to, value);
    }
}
