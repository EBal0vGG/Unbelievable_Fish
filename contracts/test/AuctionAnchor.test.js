const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("AuctionAnchor", function () {
  async function deployFixture() {
    const [operator, other] = await ethers.getSigners();
    const Factory = await ethers.getContractFactory("AuctionAnchor");
    const contract = await Factory.deploy(operator.address);
    await contract.waitForDeployment();
    return { contract, operator, other };
  }

  it("creates auction, anchors bid, finalizes with nonce progression", async function () {
    const { contract, operator } = await deployFixture();
    const ref = ethers.keccak256(ethers.toUtf8Bytes("auction|a1"));
    const bidHash = ethers.keccak256(ethers.toUtf8Bytes("bid|a1|buyer|120|t"));
    const resultHash = ethers.keccak256(ethers.toUtf8Bytes("result|a1|buyer|120"));
    const winnerHash = ethers.keccak256(ethers.toUtf8Bytes("company|buyer"));

    const now = BigInt(Math.floor(Date.now() / 1000));
    const startsAt = now - 10n;
    const endsAt = now + 1000n;

    await expect(contract.createAuction(ref, 10, startsAt, endsAt, 0))
      .to.emit(contract, "AuctionCreated");

    await expect(contract.anchorBid(ref, bidHash, 1, now))
      .to.emit(contract, "BidAnchored");

    await expect(contract.finalizeAuction(ref, resultHash, winnerHash, 120, 2))
      .to.emit(contract, "AuctionFinalized");

    const state = await contract.getAuction(ref);
    expect(state.finalized).to.equal(true);
    expect(state.nextNonce).to.equal(3n);
  });

  it("rejects wrong nonce", async function () {
    const { contract } = await deployFixture();
    const ref = ethers.keccak256(ethers.toUtf8Bytes("auction|a2"));
    const now = BigInt(Math.floor(Date.now() / 1000));
    await contract.createAuction(ref, 10, now - 10n, now + 1000n, 0);
    const bidHash = ethers.keccak256(ethers.toUtf8Bytes("bid|a2|buyer|120|t"));

    await expect(contract.anchorBid(ref, bidHash, 7, now)).to.be.revertedWithCustomError(contract, "InvalidNonce");
  });

  it("enforces operator-only actions", async function () {
    const { contract, other } = await deployFixture();
    const ref = ethers.keccak256(ethers.toUtf8Bytes("auction|a3"));
    const now = BigInt(Math.floor(Date.now() / 1000));

    await expect(contract.connect(other).createAuction(ref, 10, now - 10n, now + 1000n, 0))
      .to.be.revertedWithCustomError(contract, "NotOperator");
  });
});
