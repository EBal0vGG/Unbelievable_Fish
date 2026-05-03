import type { AuctionRecord, AuctionState, BidRecord } from "@/shared/types/domain";

const AUCTION_EXTENSION_WINDOW_MS = 5 * 60 * 1000;
const AUCTION_EXTENSION_DURATION_MS = 5 * 60 * 1000;

export function getAuctionEffectiveCurrentPrice(
  auction: Pick<AuctionRecord, "currentPrice" | "finalPrice"> | null | undefined,
  bids: BidRecord[] = [],
): number {
  return Math.max(auction?.currentPrice ?? 0, auction?.finalPrice ?? 0, ...bids.map((bid) => bid.amount));
}

function normalizeCompanyId(value?: string | null): string {
  return value?.trim().toLowerCase() ?? "";
}

export function getBidAccessError(input: {
  actorCompanyId?: string | null;
  sellerCompanyId?: string | null;
  leaderCompanyId?: string | null;
  bids?: BidRecord[];
}): string | null {
  const actorCompanyId = normalizeCompanyId(input.actorCompanyId);
  const sellerCompanyId = normalizeCompanyId(input.sellerCompanyId);
  const leaderCompanyId = normalizeCompanyId(input.leaderCompanyId);
  const bids = input.bids ?? [];

  if (!actorCompanyId) {
    return null;
  }

  if (sellerCompanyId && actorCompanyId === sellerCompanyId) {
    return "нельзя ставить на свой товар";
  }

  if (leaderCompanyId && actorCompanyId === leaderCompanyId) {
    return "нельзя перебивать свою же ставку";
  }

  if (bids.some((bid) => normalizeCompanyId(bid.bidderCompanyId) === actorCompanyId)) {
    return "нельзя перебивать свою же ставку";
  }

  return null;
}

export function getBidValidationError(
  auction: Pick<AuctionRecord, "state" | "startsAt" | "endsAt" | "currentPrice" | "finalPrice" | "minBidStep" | "leaderCompanyId">,
  amount: number,
  placedAt: Date,
  bids: BidRecord[] = [],
): string | null {
  if (auction.state !== "PUBLISHED") {
    return "Аукцион сейчас не активен";
  }

  const startsAt = new Date(auction.startsAt);
  const endsAt = new Date(auction.endsAt);

  if (placedAt < startsAt) {
    return "Аукцион еще не начался";
  }

  if (placedAt > endsAt) {
    return "Аукцион уже завершен";
  }

  const current = getAuctionEffectiveCurrentPrice(auction, bids);
  const minStep = Math.max(auction.minBidStep ?? 1, 1);
  const isFirstBid = !auction.leaderCompanyId && bids.length === 0;
  const minAllowed = isFirstBid ? current : current + minStep;
  if (amount < minAllowed) {
    return `Ставка слишком маленькая. Минимум: ${minAllowed}`;
  }

  return null;
}

export function getMinAllowedBid(
  auction: Pick<AuctionRecord, "currentPrice" | "finalPrice" | "minBidStep" | "leaderCompanyId">,
  bids: BidRecord[] = [],
): number {
  const current = getAuctionEffectiveCurrentPrice(auction, bids);
  const minStep = Math.max(auction.minBidStep ?? 1, 1);
  const isFirstBid = !auction.leaderCompanyId && bids.length === 0;
  return isFirstBid ? current : current + minStep;
}

export function getAuctionEndAfterBid(
  auction: Pick<AuctionRecord, "endsAt">,
  placedAt: Date,
): string {
  const currentEnd = new Date(auction.endsAt);

  if (currentEnd.getTime() - placedAt.getTime() > AUCTION_EXTENSION_WINDOW_MS) {
    return auction.endsAt;
  }

  return new Date(currentEnd.getTime() + AUCTION_EXTENSION_DURATION_MS).toISOString();
}

export function getEffectiveAuctionState(
  auction: Pick<AuctionRecord, "state" | "endsAt" | "leaderCompanyId" | "winnerCompanyId" | "finalPrice">,
  now = new Date(),
): AuctionState {
  if (auction.state !== "PUBLISHED") {
    return auction.state;
  }

  const endsAt = new Date(auction.endsAt);
  if (Number.isNaN(endsAt.getTime()) || endsAt > now) {
    return auction.state;
  }

  return auction.winnerCompanyId || auction.leaderCompanyId || auction.finalPrice ? "WON" : "CANCELLED";
}

export function withEffectiveAuctionState(auction: AuctionRecord, now = new Date()): AuctionRecord {
  const state = getEffectiveAuctionState(auction, now);
  if (state === auction.state) {
    return auction;
  }
  return {
    ...auction,
    state,
    finalPrice: state === "WON" ? auction.finalPrice ?? auction.currentPrice : auction.finalPrice,
    winnerCompanyId: state === "WON" ? auction.winnerCompanyId ?? auction.leaderCompanyId : auction.winnerCompanyId,
    statusNote:
      state === "WON"
        ? "Аукцион завершен по времени. Backend close job фиксирует финальное состояние в БД."
        : "Аукцион завершен без ставок. Backend close job фиксирует отмену в БД.",
  };
}
