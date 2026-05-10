const { ethers } = require("hardhat");

async function main() {
  const [deployer] = await ethers.getSigners();
  const operator = process.env.CHAIN_OPERATOR_ADDRESS || deployer.address;

  const Factory = await ethers.getContractFactory("AuctionAnchor");
  const contract = await Factory.deploy(operator);
  await contract.waitForDeployment();

  console.log("AuctionAnchor deployed");
  console.log("address:", await contract.getAddress());
  console.log("operator:", operator);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
