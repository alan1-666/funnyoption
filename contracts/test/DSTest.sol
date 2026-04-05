// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract DSTest {
    function assertTrue(bool condition, string memory message) internal pure {
        require(condition, message);
    }

    function assertEq(uint256 left, uint256 right, string memory message) internal pure {
        require(left == right, message);
    }

    function assertEqBytes32(bytes32 left, bytes32 right, string memory message) internal pure {
        require(left == right, message);
    }
}
