// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@chainlink/contracts/src/v0.8/interfaces/AggregatorV3Interface.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

interface IPriceConsumer {
    function getLatestPrice() external view returns (int256 price, uint256 timestamp);
    function getDecimals() external view returns (uint8);
}

contract ChainlinkPriceOracle is IPriceConsumer, Ownable {
    AggregatorV3Interface private priceFeed;

    uint256 public constant STALENESS_THRESHOLD = 3600; // 1 hour

    event FeedUpdated(address indexed oldFeed, address indexed newFeed);

    modifier freshPrice(uint256 updatedAt) {
        require(block.timestamp - updatedAt <= STALENESS_THRESHOLD, "Stale price");
        _;
    }

    constructor(address _priceFeed) Ownable(msg.sender) {
        priceFeed = AggregatorV3Interface(_priceFeed);
    }

    function getLatestPrice() external view returns (int256 price, uint256 timestamp) {
        (
            /* uint80 roundId */,
            int256 answer,
            /* uint256 startedAt */,
            uint256 updatedAt,
            /* uint80 answeredInRound */
        ) = priceFeed.latestRoundData();
        require(updatedAt != 0, "Round not complete");
        require(block.timestamp - updatedAt <= STALENESS_THRESHOLD, "Stale price data");
        return (answer, updatedAt);
    }

    function getDecimals() external view returns (uint8) {
        return priceFeed.decimals();
    }

    function updateFeed(address newFeed) external onlyOwner {
        address old = address(priceFeed);
        priceFeed = AggregatorV3Interface(newFeed);
        emit FeedUpdated(old, newFeed);
    }

    function getRoundData(uint80 roundId)
        external view
        returns (int256 answer, uint256 startedAt, uint256 updatedAt)
    {
        (, int256 ans, uint256 start, uint256 updated, ) = priceFeed.getRoundData(roundId);
        return (ans, start, updated);
    }
}
