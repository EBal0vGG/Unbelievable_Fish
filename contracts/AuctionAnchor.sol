// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract AuctionAnchor {
    struct AuctionState {
        bool exists;
        bool finalized;
        uint64 startsAt;
        uint64 endsAt;
        uint64 lastBidPlacedAt;
        uint64 nextNonce;
        uint256 minBidStep;
        bytes32 resultHash;
        bytes32 winnerCompanyHash;
        uint256 finalPrice;
    }

    address public immutable operator;

    mapping(bytes32 => AuctionState) private auctions;
    mapping(bytes32 => bool) public anchoredBidHashes;

    event AuctionCreated(
        bytes32 indexed auctionRef,
        uint256 minBidStep,
        uint64 startsAt,
        uint64 endsAt,
        uint64 nonce,
        address sender
    );

    event BidAnchored(
        bytes32 indexed auctionRef,
        bytes32 indexed bidHash,
        uint64 nonce,
        uint64 placedAt,
        address sender
    );

    event AuctionFinalized(
        bytes32 indexed auctionRef,
        bytes32 indexed resultHash,
        bytes32 indexed winnerCompanyHash,
        uint256 finalPrice,
        uint64 nonce,
        address sender
    );

    error NotOperator();
    error AuctionAlreadyExists();
    error AuctionNotFound();
    error AuctionAlreadyFinalized();
    error InvalidAuctionSchedule();
    error InvalidBidHash();
    error BidAlreadyAnchored();
    error InvalidNonce();
    error BidBackdated();
    error InvalidFinalizePayload();

    constructor(address _operator) {
        operator = _operator;
    }

    modifier onlyOperator() {
        if (msg.sender != operator) revert NotOperator();
        _;
    }

    function createAuction(
        bytes32 auctionRef,
        uint256 minBidStep,
        uint64 startsAt,
        uint64 endsAt,
        uint64 nonce
    ) external onlyOperator {
        if (auctions[auctionRef].exists) revert AuctionAlreadyExists();
        if (startsAt == 0 || endsAt == 0 || startsAt >= endsAt) revert InvalidAuctionSchedule();
        if (minBidStep == 0) revert InvalidAuctionSchedule();
        if (nonce != 0) revert InvalidNonce();

        auctions[auctionRef] = AuctionState({
            exists: true,
            finalized: false,
            startsAt: startsAt,
            endsAt: endsAt,
            lastBidPlacedAt: 0,
            nextNonce: 1,
            minBidStep: minBidStep,
            resultHash: bytes32(0),
            winnerCompanyHash: bytes32(0),
            finalPrice: 0
        });

        emit AuctionCreated(auctionRef, minBidStep, startsAt, endsAt, nonce, msg.sender);
    }

    function anchorBid(
        bytes32 auctionRef,
        bytes32 bidHash,
        uint64 nonce,
        uint64 placedAt
    ) external onlyOperator {
        AuctionState storage a = auctions[auctionRef];
        if (!a.exists) revert AuctionNotFound();
        if (a.finalized) revert AuctionAlreadyFinalized();
        if (nonce != a.nextNonce) revert InvalidNonce();
        if (bidHash == bytes32(0)) revert InvalidBidHash();
        if (anchoredBidHashes[bidHash]) revert BidAlreadyAnchored();
        if (placedAt < a.startsAt || placedAt > a.endsAt) revert BidBackdated();
        if (a.lastBidPlacedAt != 0 && placedAt < a.lastBidPlacedAt) revert BidBackdated();

        anchoredBidHashes[bidHash] = true;
        a.lastBidPlacedAt = placedAt;
        a.nextNonce += 1;

        emit BidAnchored(auctionRef, bidHash, nonce, placedAt, msg.sender);
    }

    function finalizeAuction(
        bytes32 auctionRef,
        bytes32 resultHash,
        bytes32 winnerCompanyHash,
        uint256 finalPrice,
        uint64 nonce
    ) external onlyOperator {
        AuctionState storage a = auctions[auctionRef];
        if (!a.exists) revert AuctionNotFound();
        if (a.finalized) revert AuctionAlreadyFinalized();
        if (nonce != a.nextNonce) revert InvalidNonce();
        if (resultHash == bytes32(0) || winnerCompanyHash == bytes32(0) || finalPrice == 0) revert InvalidFinalizePayload();

        a.finalized = true;
        a.nextNonce += 1;
        a.resultHash = resultHash;
        a.winnerCompanyHash = winnerCompanyHash;
        a.finalPrice = finalPrice;

        emit AuctionFinalized(auctionRef, resultHash, winnerCompanyHash, finalPrice, nonce, msg.sender);
    }

    function getAuction(bytes32 auctionRef) external view returns (AuctionState memory) {
        return auctions[auctionRef];
    }
}
