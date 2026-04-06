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
  placedAt: string;
  sellerCompanyId?: string;
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

function normalizeCompanyId(value?: string | null): string {
  return (value ?? "").trim().toLowerCase();
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
  const relatedLot = listLotsStore().find((item) => item.id === input.lotId);
  const fallbackAuction: AuctionRecord = {
    id: makeClientId("auction"),
    lotId: input.lotId,
    sellerCompanyId: relatedLot?.sellerCompanyId,
    state: "DRAFT",
    startsAt: input.startsAt,
    endsAt: input.endsAt,
    source: "mock",
    statusNote:
      "Temporary placeholder until Trading CreateAuction returns a stable identifier or a query endpoint is available.",
  };

  try {
    await apiRequest("trading", "/auctions", {
      method: "POST",
      session,
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
        "Trading command accepted. Backend currently does not expose created auction_id back to the UI, so a local mirror was created.",
    });
    addActivity("Аукцион выставлен", `${mirroredAuction.lotId} · command accepted`);
    return {
      data: mirroredAuction,
      meta: mixedMeta(
        "CreateAuction command was sent, but backend does not return created id and no list query is available yet.",
      ),
    };
  } catch (error) {
    if (!isRecoverableApiGap(error)) {
      throw error;
    }

    const createdAuction = upsertAuctionStore(fallbackAuction);
    addActivity("Аукцион выставлен", `${createdAuction.lotId} · local placeholder`);
    return {
      data: createdAuction,
      meta: mockMeta(
        "CreateAuction fallback is active until Trading command routing and read model are exposed end-to-end.",
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
  const auction = listAuctionsStore().find((item) => item.id === input.auctionId);
  const lot =
    listLotsStore().find((item) => item.auctionId === input.auctionId) ??
    (auction ? listLotsStore().find((item) => item.id === auction.lotId) : undefined);
  const actorCompanyId = normalizeCompanyId(session?.companyId);
  const sellerCompanyId = normalizeCompanyId(
    input.sellerCompanyId ?? auction?.sellerCompanyId ?? lot?.sellerCompanyId,
  );
  const existingBids = listBidsStore(input.auctionId);
  const hasOwnBid = existingBids.some(
    (bid) => normalizeCompanyId(bid.bidderCompanyId) === actorCompanyId,
  );
  const isLeader = normalizeCompanyId(auction?.leaderCompanyId) === actorCompanyId;

  if (actorCompanyId && sellerCompanyId && actorCompanyId === sellerCompanyId) {
    throw new ApiError("нельзя ставить ставки на свой товар", 400, "OWN_LOT_BID_FORBIDDEN");
  }

  if (actorCompanyId && (hasOwnBid || isLeader)) {
    throw new ApiError("нельзя перебивать свою же ставку", 400, "SELF_OUTBID_FORBIDDEN");
  }

  const fallbackBid: BidRecord = {
    auctionId: input.auctionId,
    bidderCompanyId: session?.companyId ?? "unknown-company",
    amount: input.amount,
    placedAt: input.placedAt,
    source: "mock",
  };

  try {
    await apiRequest("trading", `/auctions/${input.auctionId}/bids`, {
      method: "POST",
      session,
      body: {
        amount: input.amount,
        placed_at: input.placedAt,
      },
    });

    const storedBid = appendBidStore({ ...fallbackBid, source: "mixed" });
    addActivity("Ставка отправлена", `${storedBid.auctionId} · ${storedBid.amount}`);
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

    const storedBid = appendBidStore(fallbackBid);
    addActivity("Ставка отправлена", `${storedBid.auctionId} · local placeholder`);
    return {
      data: storedBid,
      meta: mockMeta(
        "PlaceBid fallback is active. Real bidding currently requires a reachable Trading auction endpoint and a known auction id.",
      ),
    };
  }
}
