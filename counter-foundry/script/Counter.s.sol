// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Script, console} from "forge-std/Script.sol";
import {Counter} from "../src/Counter.sol";

contract CounterScript is Script {
    Counter public counter;

    function setUp() public {}

    function run() public {
        // 从环境变量读取私钥
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        console.log("deployerPrivateKey:", deployerPrivateKey);

        vm.startBroadcast(deployerPrivateKey);

        counter = new Counter();

        vm.stopBroadcast();

        // 输出部署地址
        console.log("Counter deployed at:", address(counter));
    }
}
