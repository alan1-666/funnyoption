// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {FunnyVault} from "../src/FunnyVault.sol";
import {MockUSDT} from "../src/MockUSDT.sol";
import {MockPriceFeed} from "../src/MockPriceFeed.sol";

contract FunnyVaultNativeDepositTest is Test {
    MockUSDT token;
    FunnyVault vault;
    MockPriceFeed feed;
    address operator;
    address user;

    function setUp() public {
        operator = address(this);
        user = address(0xBEEF);

        token = new MockUSDT();
        vault = new FunnyVault(address(token), operator);
        feed = new MockPriceFeed(8, 600e8); // 1 BNB = $600, 8 decimals

        token.addMinter(address(vault));
        vault.setNativePriceFeed(address(feed));
        vault.setFallbackNativeRate(500e8);

        vm.deal(user, 100 ether);
    }

    function test_depositNative_basic() public {
        vm.prank(user);
        vault.depositNative{value: 1 ether}();

        // 1 ETH * 600e8 / 10^(18+8-6) = 600e6 = 600 USDT
        assertEq(vault.depositedBalance(user), 600e6);
        assertEq(token.balanceOf(address(vault)), 600e6);
    }

    function test_depositNative_fractional() public {
        vm.prank(user);
        vault.depositNative{value: 0.5 ether}();

        // 0.5 ETH * 600e8 / 10^20 = 300e6 = 300 USDT
        assertEq(vault.depositedBalance(user), 300e6);
    }

    function test_depositNative_smallAmount() public {
        vm.prank(user);
        vault.depositNative{value: 0.001 ether}();

        // 0.001 ETH * 600e8 / 10^20 = 600_000 = 0.60 USDT
        assertEq(vault.depositedBalance(user), 600_000);
    }

    function test_depositNative_emitsDepositedEvent() public {
        vm.expectEmit(true, false, false, true);
        emit FunnyVault.Deposited(user, 600e6);

        vm.prank(user);
        vault.depositNative{value: 1 ether}();
    }

    function test_depositNative_revertsZeroValue() public {
        vm.prank(user);
        vm.expectRevert(FunnyVault.InvalidAmount.selector);
        vault.depositNative{value: 0}();
    }

    function test_depositNative_usesFallbackWhenOracleStale() public {
        // Make oracle stale by warping time
        vm.warp(block.timestamp + 2 hours);

        vm.prank(user);
        vault.depositNative{value: 1 ether}();

        // Falls back to 500e8: 1 ETH * 500e8 / 10^20 = 500e6
        assertEq(vault.depositedBalance(user), 500e6);
    }

    function test_depositNative_usesFallbackWhenNoPriceFeed() public {
        vault.setNativePriceFeed(address(0));

        vm.prank(user);
        vault.depositNative{value: 1 ether}();

        // Uses fallback 500e8: 500 USDT
        assertEq(vault.depositedBalance(user), 500e6);
    }

    function test_depositNative_revertsWhenNoPriceFeedAndNoFallback() public {
        vault.setNativePriceFeed(address(0));
        vault.setFallbackNativeRate(0);

        vm.prank(user);
        vm.expectRevert(FunnyVault.NoPriceFeed.selector);
        vault.depositNative{value: 1 ether}();
    }

    function test_depositNative_multipleDepositsAccumulate() public {
        vm.startPrank(user);
        vault.depositNative{value: 1 ether}();
        vault.depositNative{value: 2 ether}();
        vm.stopPrank();

        // (1 + 2) * 600 = 1800 USDT
        assertEq(vault.depositedBalance(user), 1800e6);
    }

    function test_setNativePriceFeed_onlyOperator() public {
        vm.prank(user);
        vm.expectRevert(FunnyVault.OnlyOperator.selector);
        vault.setNativePriceFeed(address(feed));
    }

    function test_setFallbackNativeRate_onlyOperator() public {
        vm.prank(user);
        vm.expectRevert(FunnyVault.OnlyOperator.selector);
        vault.setFallbackNativeRate(100e8);
    }

    function test_withdrawNative_operatorCanSweep() public {
        vm.prank(user);
        vault.depositNative{value: 5 ether}();

        address payable recipient = payable(address(0xCAFE));
        vault.withdrawNative(recipient, 3 ether);

        assertEq(recipient.balance, 3 ether);
        assertEq(address(vault).balance, 2 ether);
    }

    function test_withdrawNative_onlyOperator() public {
        vm.prank(user);
        vault.depositNative{value: 1 ether}();

        vm.prank(user);
        vm.expectRevert(FunnyVault.OnlyOperator.selector);
        vault.withdrawNative(payable(user), 1 ether);
    }

    function test_mockUSDT_minterRole() public {
        address minter = address(0xABCD);
        token.addMinter(minter);
        assertTrue(token.minters(minter));

        vm.prank(minter);
        token.mint(user, 1000e6);
        assertEq(token.balanceOf(user), 1000e6);

        token.removeMinter(minter);
        assertFalse(token.minters(minter));

        vm.prank(minter);
        vm.expectRevert(MockUSDT.OnlyMinter.selector);
        token.mint(user, 1000e6);
    }

    function test_existingDeposit_stillWorks() public {
        token.mint(user, 1000e6);

        vm.startPrank(user);
        token.approve(address(vault), 1000e6);
        vault.deposit(1000e6);
        vm.stopPrank();

        assertEq(vault.depositedBalance(user), 1000e6);
    }

    function test_priceChangeReflected() public {
        vm.prank(user);
        vault.depositNative{value: 1 ether}();
        assertEq(vault.depositedBalance(user), 600e6);

        feed.setPrice(700e8);

        vm.prank(user);
        vault.depositNative{value: 1 ether}();
        // 600 + 700 = 1300 USDT
        assertEq(vault.depositedBalance(user), 1300e6);
    }
}
