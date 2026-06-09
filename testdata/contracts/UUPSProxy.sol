// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC20/ERC20Upgradeable.sol";

/**
 * @title UpgradeableToken
 * @dev UUPS upgradeable ERC20 token.
 */
contract UpgradeableToken is
    Initializable,
    ERC20Upgradeable,
    OwnableUpgradeable,
    UUPSUpgradeable
{
    /// @custom:oz-upgrades-unsafe-allow constructor
    constructor() {
        _disableInitializers();
    }

    function initialize(
        string memory name,
        string memory symbol,
        address initialOwner
    ) public initializer {
        __ERC20_init(name, symbol);
        __Ownable_init(initialOwner);
        __UUPSUpgradeable_init();
    }

    function mint(address to, uint256 amount) public onlyOwner {
        _mint(to, amount);
    }

    function _authorizeUpgrade(address newImplementation)
        internal override onlyOwner
    {}

    function getImplementation() external view returns (address) {
        return _getImplementation();
    }
}

/**
 * @title ProxyAdmin
 * @dev Minimal proxy admin for managing upgrades.
 */
contract ProxyAdmin is Ownable {
    event ProxyUpgraded(address indexed proxy, address indexed implementation);

    constructor() Ownable(msg.sender) {}

    function upgrade(address proxy, address implementation) external onlyOwner {
        (bool success, ) = proxy.call(
            abi.encodeWithSignature("upgradeTo(address)", implementation)
        );
        require(success, "Upgrade failed");
        emit ProxyUpgraded(proxy, implementation);
    }

    function upgradeAndCall(
        address proxy,
        address implementation,
        bytes calldata data
    ) external onlyOwner {
        (bool success, ) = proxy.call(
            abi.encodeWithSignature(
                "upgradeToAndCall(address,bytes)",
                implementation,
                data
            )
        );
        require(success, "Upgrade with call failed");
        emit ProxyUpgraded(proxy, implementation);
    }
}
