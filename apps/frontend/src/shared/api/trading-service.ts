import { ApiError, apiRequest, isRecoverableApiGap } from "@/shared/api/http-client";
import { getProjectionByAuctionId, getDealByAuctionId } from "@/shared/api/deals-service";
import { canFallbackCommand, mixedMeta, mockMeta } from "@/shared/api/service-helpers";
import { isSellerSession } from "@/shared/lib/access";
import {
  addActivity,
  appendBidStore,
  listAuctionsStore,
  listBidsStore,
  listLotsStore,
  upsertAuctionStore,
  upsertLotStore,
} from "@/shared/api/mock-store";
import { makeClientId } from "@/shared/lib/id";
import {
  getBidAccessError,
  getAuctionEndAfterBid,
  getBidValidationError,
  withEffectiveAuctionState,
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

interface AuctionSummaryDTO {
  auction_id: string;
  lot_id: string;
  state: AuctionRecord["state"];
  starts_at: string;
  ends_at: string;
  current_price?: number | null;
  min_bid_step?: number | null;
  leader_company_id?: string | null;
  winner_company_id?: string | null;
  final_price?: number | null;
}

function getAuctionFallbackNote(): string {
  return "Trading query endpoints for auction list/details are not exposed in the current backend build, using local auction mirror.";
}

function deriveAuctionFromLot(lot: ReturnType<typeof listLotsStore>[number]): AuctionRecord {
  const startsAt = lot.auctionStartsAt;
  const startTs = new Date(startsAt).getTime();
  const endsAt = Number.isNaN(startTs)
    ? new Date().toISOString()
    : new Date(startTs + lot.auctionDurationMinutes * 60_000).toISOString();
  return withEffectiveAuctionState({
    id: lot.auctionId ?? makeClientId("auction"),
    lotId: lot.id,
    sellerCompanyId: lot.sellerCompanyId,
    state: lot.status === "PUBLISHED" ? "PUBLISHED" : lot.status === "CLOSED" ? "WON" : "CANCELLED",
    startsAt,
    endsAt,
    currentPrice: lot.currentPrice ?? lot.startPrice,
    minBidStep: lot.minBidStep ?? 1,
    finalPrice: lot.finalPrice,
    source: "mixed",
    statusNote: "Собрано из Catalog lot read-model до появления Trading query endpoint.",
  });
}

function mapAuctionSummary(summary: AuctionSummaryDTO, existing?: AuctionRecord): AuctionRecord {
  return withEffectiveAuctionState({
    id: summary.auction_id,
    lotId: summary.lot_id ?? existing?.lotId ?? "unknown-lot",
    sellerCompanyId: existing?.sellerCompanyId,
    state: summary.state,
    startsAt: summary.starts_at ?? existing?.startsAt ?? new Date().toISOString(),
    endsAt: summary.ends_at ?? existing?.endsAt ?? new Date().toISOString(),
    currentPrice: summary.current_price ?? existing?.currentPrice,
    minBidStep: summary.min_bid_step ?? existing?.minBidStep ?? 1,
    finalPrice: summary.final_price ?? existing?.finalPrice,
    winnerCompanyId: summary.winner_company_id ?? existing?.winnerCompanyId ?? undefined,
    leaderCompanyId: summary.leader_company_id ?? existing?.leaderCompanyId,
    source: existing ? "mixed" : "api",
  });
}

function getMissingTradingSessionError(session: UserSession | null): ApiError | null {
  if (!session) {
    return new ApiError("Войдите в профиль, чтобы продолжить", 400, "MISSING_SESSION");
  }

  if (!session.userId) {
    return new ApiError("Войдите в профиль, чтобы продолжить", 400, "MISSING_USER_ID");
  }

  if (!session.companyId) {
    return new ApiError(
      "В профиле не указана компания — торги и каталог ожидают привязанную организацию. Зарегистрируйтесь с блоком «Привязать компанию».",
      400,
      "MISSING_COMPANY_ID",
    );
  }

  return null;
}

function isCreateAuctionCompatibilityGap(error: unknown, session: UserSession): boolean {
  if (!canFallbackCommand()) {
    return false;
  }
  if (error instanceof ApiError && error.code === "AUCTION_NOT_READY") {
    return false;
  }
  if (isRecoverableApiGap(error)) {
    return true;
  }

  return isSellerSession(session) && error instanceof ApiError && error.status === 403 && error.code === "FORBIDDEN";
}

async function waitAuctionSummaryByLot(
  lotId: string,
  session: UserSession,
  attempts = 24,
  delayMs = 500,
): Promise<AuctionSummaryDTO | null> {
  for (let i = 0; i < attempts; i += 1) {
    try {
      const result = await apiRequest<AuctionSummaryDTO>("trading", `/auctions/by-lot/${lotId}`, {
        method: "GET",
        session,
      });
      if (result.auction_id) {
        return result;
      }
    } catch (error) {
      if (!isRecoverableApiGap(error)) {
        throw error;
      }
    }
    await new Promise((resolve) => setTimeout(resolve, delayMs));
  }
  return null;
}

export async function listAuctions(session: UserSession | null): Promise<ServiceResult<AuctionRecord[]>> {
  const localAuctions = listAuctionsStore()
    .filter((item) => (canFallbackCommand() ? true : item.source !== "mock"))
    .map((item) => withEffectiveAuctionState(item));

  try {
    const summaries = await apiRequest<AuctionSummaryDTO[]>("trading", "/auctions", { session });
    const byId = new Map(localAuctions.map((item) => [item.id, item]));
    const apiAuctions = summaries.map((summary) => {
      const mapped = mapAuctionSummary(summary, byId.get(summary.auction_id));
      upsertAuctionStore(mapped);
      byId.set(mapped.id, mapped);
      return mapped;
    });
    const apiIds = new Set(apiAuctions.map((item) => item.id));
    const fallbackAuctions = canFallbackCommand()
      ? localAuctions.filter((item) => item.source === "mock" && !apiIds.has(item.id))
      : [];

    return {
      data: [...apiAuctions, ...fallbackAuctions].sort((left, right) => right.startsAt.localeCompare(left.startsAt)),
      meta: fallbackAuctions.length
        ? mixedMeta("Список торгов получен из backend и дополнен локальными демо-записями.")
        : { source: "api" },
    };
  } catch (error) {
    if (!isRecoverableApiGap(error) && !(error instanceof ApiError && error.status === 401)) {
      throw error;
    }
  }

  const existing = localAuctions;
  const knownIDs = new Set(existing.map((item) => item.id));
  const derived = listLotsStore()
    .filter((lot) => lot.status === "PUBLISHED" || lot.status === "CLOSED" || lot.status === "CANCELLED")
    .filter((lot) => Boolean(lot.auctionId))
    .filter((lot) => !knownIDs.has(lot.auctionId as string))
    .map((lot) => deriveAuctionFromLot(lot));

  return {
    data: [...derived, ...existing].sort((left, right) => right.startsAt.localeCompare(left.startsAt)),
    meta: canFallbackCommand() ? mockMeta(getAuctionFallbackNote()) : mixedMeta("Список строится только по данным backend."),
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
    currentPrice: relatedLot.startPrice,
    source: "mock",
    statusNote: "Аукцион выставлен и готов принимать ставки.",
  };

  try {
    // Trading HTTP API does not expose POST /auctions — the aggregate is created when LotPublished
    // is processed (integration/outbox). We sync by lot and publish if still DRAFT.
    const summary = await waitAuctionSummaryByLot(input.lotId, activeSession);
    if (!summary?.auction_id) {
      throw new ApiError(
        "Аукцион для лота ещё не создан в trading. После публикации лота должен отработать integration (outbox / chain_runner): проверьте, что он запущен, и подождите несколько секунд.",
        422,
        "AUCTION_NOT_READY",
      );
    }

    const auctionId = summary.auction_id;
    if (summary.state === "DRAFT") {
      await apiRequest("trading", `/auctions/${auctionId}/publish`, {
        method: "POST",
        session: activeSession,
      });
    }

    const refreshed = await apiRequest<AuctionSummaryDTO>("trading", `/auctions/${auctionId}`, {
      session: activeSession,
    });
    const existingAuction = listAuctionsStore().find((item) => item.id === refreshed.auction_id);
    const mapped = mapAuctionSummary(refreshed, existingAuction);
    const mirroredAuction = upsertAuctionStore({
      ...mapped,
      source: existingAuction ? "mixed" : "api",
    });
    upsertLotStore({
      ...relatedLot,
      auctionId,
      source: relatedLot.source === "mock" ? "mixed" : relatedLot.source,
    });
    addActivity("Аукцион выставлен", mirroredAuction.lotId, session);
    return {
      data: mirroredAuction,
      meta: { source: "api" },
    };
  } catch (error) {
    if (!isCreateAuctionCompatibilityGap(error, activeSession)) {
      throw error;
    }

    const createdAuction = upsertAuctionStore(fallbackAuction);
    addActivity("Аукцион выставлен", createdAuction.lotId, session);
    return {
      data: createdAuction,
      meta: mockMeta("Аукцион выставлен локально. Пересоберите trading-сервис, чтобы команда уходила в backend."),
    };
  }
}

export async function getAuctionDetails(
  auctionId: string,
  session: UserSession | null,
): Promise<ServiceResult<AuctionDetails>> {
  let meta: ServiceMeta = mockMeta(getAuctionFallbackNote());
  let auction = listAuctionsStore().find((item) => item.id === auctionId) ?? null;
  if (!auction) {
    const lot = listLotsStore().find((item) => item.auctionId === auctionId);
    auction = lot ? deriveAuctionFromLot(lot) : null;
  }
  if (auction) {
    auction = upsertAuctionStore(withEffectiveAuctionState(auction));
  }

  try {
    const summary = await apiRequest<AuctionSummaryDTO>("trading", `/auctions/${auctionId}`, { session });

    const existing = listAuctionsStore().find((item) => item.id === summary.auction_id);
    auction = mapAuctionSummary(summary, existing);
    upsertAuctionStore(auction);
    meta = { source: existing ? "mixed" : "api" };
  } catch (error) {
    if (!isRecoverableApiGap(error) || !canFallbackCommand()) {
      throw error;
    }
  }

  const [projectionResult, dealResult] = await Promise.all([
    auction?.source === "mock" && !canFallbackCommand()
      ? Promise.resolve({ data: null, meta: { source: "mock" as const } })
      : getProjectionByAuctionId(auctionId, session),
    auction?.source === "mock" && !canFallbackCommand()
      ? Promise.resolve({ data: null, meta: { source: "mock" as const } })
      : getDealByAuctionId(auctionId, session),
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
      const tooSmall = validationError.startsWith("Ставка слишком маленькая");
      const status = tooSmall ? 400 : 409;
      const code = tooSmall ? "BID_TOO_SMALL" : "AUCTION_NOT_ACTIVE";
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
