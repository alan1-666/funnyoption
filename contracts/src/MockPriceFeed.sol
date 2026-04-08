// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/// @dev Minimal Chainlink AggregatorV3Interface mock for local/test environments.
contract MockPriceFeed {
    uint8 public immutable override_decimals;
    int256 public price;
    uint256 public updatedAt;
    address public immutable owner;

    error OnlyOwner();

    constructor(uint8 decimals_, int256 initialPrice) {
        override_decimals = decimals_;
        price = initialPrice;
        updatedAt = block.timestamp;
        owner = msg.sender;
    }

    function setPrice(int256 newPrice) external {
        if (msg.sender != owner) revert OnlyOwner();
        price = newPrice;
        updatedAt = block.timestamp;
    }

    function decimals() external view returns (uint8) {
        return override_decimals;
    }

    function latestRoundData()
        external
        view
        returns (uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt_, uint80 answeredInRound)
    {
        return (1, price, updatedAt, updatedAt, 1);
    }
}
