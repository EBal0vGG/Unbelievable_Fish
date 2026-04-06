import { ApiError, apiRequest, isRecoverableApiGap } from "@/shared/api/http-client";
import { getProjectionByAuctionId, getDealByAuctionId } from "@/shared/api/deals-service";
import { mixedMeta, mockMeta } from "@/shared/api/service-helpers";
import {
  addActivity,
  appendBidStore,
  listAuctionsStore,
  listBidsStore,
  listLotsStore,
  upsertAuctionStore,
} from "@/shared/api/mock-store";
import { makeClientId } from "@/shared/lib/id";
import {
  getBidAccessError,
  getAuctionEndAfterBid,
  getBidValidationError,
} from "@/shared/lib/trading-domain";
import type {
  AuctionRecord,
  BidRecord,
  DealProjectionRecord,
  DealRecord,
  ServiceMeta,
  ServiceResult,
  UserSession,
} from "@/shared/types/domain";

interface CreateAuctionInput {
  lotId: string;
  startsAt: string;
  endsAt: string;
}

interface PlaceBidInput {
  auctionId: string;
  amount: number;
}

interface AuctionDetails {
  auction: AuctionRecord | null;
  bids: BidRecord[];
  projection: DealProjectionRecord | null;
  deal: DealRecord | null;
}

function getAuctionFallbackNote(): string {
  return "Trading query endpoints for auction list/details are not exposed in the current backend build, using local auction mirror.";
}

function getMissingTradingSessionError(session: UserSession | null): ApiError | null {
  if (!session?.companyId) {
    return new ApiError("Войдите в профиль, чтобы продолжить", 400, "MISSING_COMPANY_ID");
  }

  if (!session.userId) {
    return new ApiError("Войдите в профиль, чтобы продолжить", 400, "MISSING_USER_ID");
  }

  return null;
}

export async function listAuctions(): Promise<ServiceResult<AuctionRecord[]>> {
  // TODO: replace local list once GET /auctions read-model endpoint exists in Trading.
  return {
    data: listAuctionsStore(),
    meta: mockMeta(getAuctionFallbackNote()),
  };
}

export async function createAuction(
  input: CreateAuctionInput,
  session: UserSession | null,
): Promise<ServiceResult<AuctionRecord>> {
  const sessionError = getMissingTradingSessionError(session);
  if (sessionError) {
    throw sessionError;
  }
  const activeSession = session as UserSession;
  const relatedLot = listLotsStore().find((item) => item.id === input.lotId);
  if (!relatedLot) {
    throw new ApiError("Лот не найден", 404, "LOT_NOT_FOUND");
  }
  if (
    relatedLot.sellerCompanyId !== activeSession.companyId ||
    relatedLot.creatorUserId !== activeSession.userId
  ) {
    throw new ApiError("Аукцион можно создать только для собственного лота", 403, "LOT_ACCESS_DENIED");
  }
  if (relatedLot.status !== "PUBLISHED") {
    throw new ApiError("Сначала опубликуйте лот", 409, "LOT_NOT_PUBLISHED");
  }
  if (relatedLot?.auctionId) {
    throw new ApiError("Этот лот уже участвует в аукционе", 409, "LOT_ALREADY_LINKED");
  }

  const fallbackAuction: AuctionRecord = {
    id: makeClientId("auction"),
    lotId: input.lotId,
    sellerCompanyId: relatedLot?.sellerCompanyId,
    state: "PUBLISHED",
    startsAt: input.startsAt,
    endsAt: input.endsAt,
    source: "mock",
    statusNote:
      "Local mirror is immediately marked as published to match integration runtime behaviour after LotPublished.",
  };

  try {
    await apiRequest("trading", "/auctions", {
      method: "POST",
      session: activeSession,
      body: {
        lot_id: input.lotId,
        starts_at: input.startsAt,
        ends_at: input.endsAt,
      },
    });

    const mirroredAuction = upsertAuctionStore({
      ...fallbackAuction,
      source: "mixed",
      statusNote:
        "Trading command accepted. Backend currently does not expose created auction_id back to the UI, so a local published mirror was created.",
    });
    addActivity("Аукцион выставлен", mirroredAuction.lotId, session);
    return {
      data: mirroredAuction,
      meta: mixedMeta(
        "CreateAuction command was sent, but backend does not return created id and no list query is available yet. The frontend keeps a published mirror so bidding rules continue to work.",
      ),
    };
  } catch (error) {
    if (!isRecoverableApiGap(error)) {
      throw error;
    }

    const createdAuction = upsertAuctionStore(fallbackAuction);
    addActivity("Аукцион выставлен", createdAuction.lotId, session);
    return {
      data: createdAuction,
      meta: mockMeta(
        "CreateAuction fallback is active because Trading create/publish routes are not exposed end-to-end in the current backend build. The frontend created a published mirror.",
      ),
    };
  }
}

export async function getAuctionDetails(
  auctionId: string,
  session: UserSession | null,
): Promise<ServiceResult<AuctionDetails>> {
  let meta: ServiceMeta = mockMeta(getAuctionFallbackNote());
  let auction = listAuctionsStore().find((item) => item.id === auctionId) ?? null;

  try {
    const summary = await apiRequest<{
      auction_id: string;
      state: AuctionRecord["state"];
      winner_company_id?: string | null;
      final_price?: number | null;
    }>("trading", `/auctions/${auctionId}`, { session });

    const existing = listAuctionsStore().find((item) => item.id === summary.auction_id);
    auction = {
      id: summary.auction_id,
      lotId: existing?.lotId ?? "unknown-lot",
      sellerCompanyId: existing?.sellerCompanyId,
      state: summary.state,
      startsAt: existing?.startsAt ?? new Date().toISOString(),
      endsAt: existing?.endsAt ?? new Date().toISOString(),
      currentPrice: existing?.currentPrice,
      finalPrice: summary.final_price ?? existing?.finalPrice,
      winnerCompanyId: summary.winner_company_id ?? existing?.winnerCompanyId ?? undefined,
      leaderCompanyId: existing?.leaderCompanyId,
      source: existing ? "mixed" : "api",
    };
    upsertAuctionStore(auction);
    meta = { source: existing ? "mixed" : "api" };
  } catch (error) {
    if (!isRecoverableApiGap(error)) {
      throw error;
    }
  }

  const [projectionResult, dealResult] = await Promise.all([
    getProjectionByAuctionId(auctionId, session),
    getDealByAuctionId(auctionId, session),
  ]);

  if (auction && !auction.sellerCompanyId && projectionResult.data?.supplierId) {
    auction = upsertAuctionStore({
      ...auction,
      sellerCompanyId: projectionResult.data.supplierId,
    });
  }

  return {
    data: {
      auction,
      bids: listBidsStore(auctionId),
      projection: projectionResult.data,
      deal: dealResult.data,
    },
    meta,
  };
}

export async function placeBid(
  input: PlaceBidInput,
  session: UserSession | null,
): Promise<ServiceResult<BidRecord>> {
  const sessionError = getMissingTradingSessionError(session);
  if (sessionError) {
    throw sessionError;
  }
  const activeSession = session as UserSession;
  const auction = listAuctionsStore().find((item) => item.id === input.auctionId);
  const lot =
    listLotsStore().find((item) => item.auctionId === input.auctionId) ??
    (auction ? listLotsStore().find((item) => item.id === auction.lotId) : undefined);
  const existingBids = listBidsStore(input.auctionId);
  const placedAt = new Date();
  const bidAccessError = getBidAccessError({
    actorCompanyId: activeSession.companyId,
    sellerCompanyId: auction?.sellerCompanyId ?? lot?.sellerCompanyId,
    leaderCompanyId: auction?.leaderCompanyId,
    bids: existingBids,
  });

  if (bidAccessError) {
    throw new ApiError(bidAccessError, 403, "BID_ACCESS_DENIED");
  }

  if (auction) {
    const validationError = getBidValidationError(auction, input.amount, placedAt, existingBids);
    if (validationError) {
      const status = validationError === "Ставка должна быть выше текущей цены" ? 400 : 409;
      const code = validationError === "Ставка должна быть выше текущей цены" ? "INVALID_BID" : "AUCTION_NOT_ACTIVE";
      throw new ApiError(validationError, status, code);
    }
  }

  const fallbackBid: BidRecord = {
    auctionId: input.auctionId,
    bidderCompanyId: activeSession.companyId,
    amount: input.amount,
    placedAt: placedAt.toISOString(),
    source: "mock",
  };
  const nextAuctionEndAt = auction ? getAuctionEndAfterBid(auction, placedAt) : undefined;

  try {
    await apiRequest("trading", `/auctions/${input.auctionId}/bids`, {
      method: "POST",
      session: activeSession,
      body: {
        amount: input.amount,
        placed_at: fallbackBid.placedAt,
      },
    });

    const storedBid = appendBidStore({ ...fallbackBid, source: "mixed" }, { endsAt: nextAuctionEndAt });
    addActivity("Ставка отправлена", `${storedBid.auctionId} · ${storedBid.amount}`, session);
    return {
      data: storedBid,
      meta: {
        source: "api",
      },
    };
  } catch (error) {
    if (!isRecoverableApiGap(error)) {
      throw error;
    }

    const storedBid = appendBidStore(fallbackBid, { endsAt: nextAuctionEndAt });
    addActivity("Ставка отправлена", `${storedBid.auctionId} · ${storedBid.amount}`, session);
    return {
      data: storedBid,
      meta: mockMeta(
        "PlaceBid fallback is active. Real bidding currently requires a reachable Trading auction endpoint and a known auction id.",
      ),
    };
  }
}
