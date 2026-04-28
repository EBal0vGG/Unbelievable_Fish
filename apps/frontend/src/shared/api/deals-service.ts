import { ApiError, apiRequest, isRecoverableApiGap } from "@/shared/api/http-client";
import { mixedMeta, mockMeta } from "@/shared/api/service-helpers";
import {
  addActivity,
  listAuctionsStore,
  listDealProjectionsStore,
  listDealsStore,
  upsertDealProjectionStore,
  upsertDealStore,
} from "@/shared/api/mock-store";
import type {
  DealContractRecord,
  DealProjectionRecord,
  DealRecord,
  DealStatus,
  ProductSnapshot,
  ServiceMeta,
  ServiceResult,
  UserSession,
} from "@/shared/types/domain";

interface ProductSnapshotDTO {
  product_id: string;
  name: string;
  description: string;
  category: string;
  weight: number;
  unit: string;
  size: string;
  processing_type: string;
  volume: number;
  origin_country: string;
}

interface ContractDTO {
  number?: string;
  prepared_at?: string;
  signed_at?: string;
  signed_by?: string;
  signature_ref?: string;
  document_url?: string;
}

interface DealDTO {
  id: string;
  customer_id: string;
  supplier_id: string;
  auction_id: string;
  quantity: number;
  unit_price: number;
  total_amount: number;
  status: string;
  type: string;
  created_at: string;
  confirmed_at?: string;
  contract?: ContractDTO;
  product_snapshot: ProductSnapshotDTO;
}

interface ProjectionDTO {
  auction_id: string;
  supplier_id: string;
  start_price: number;
  published_at: string;
  status: string;
  product_snapshot: ProductSnapshotDTO;
}

export type DealActionInput =
  | { type: "confirm" }
  | { type: "prepareContract"; contractNumber?: string; documentUrl?: string }
  | { type: "signContract"; signatureRef: string }
  | { type: "requestPayment"; invoiceNumber: string; dueDate?: string }
  | { type: "markPaid"; paymentId: string; paymentType: string }
  | { type: "requestShipment" }
  | { type: "markShipped"; trackingNumber: string; carrier: string }
  | { type: "complete" }
  | { type: "cancel"; reason: string }
  | { type: "updatePrice"; newPrice: number };

const dealStatuses: DealStatus[] = [
  "pending",
  "confirmed",
  "contract_prepared",
  "contract_signed",
  "payment_requested",
  "paid",
  "shipment_requested",
  "shipped",
  "completed",
  "cancelled",
];

function normalizeDealStatus(value: string): DealStatus {
  return dealStatuses.includes(value as DealStatus) ? (value as DealStatus) : "pending";
}

function mapSnapshot(dto: ProductSnapshotDTO): ProductSnapshot {
  return {
    productId: dto.product_id,
    name: dto.name,
    description: dto.description,
    category: dto.category,
    weight: dto.weight,
    unit: dto.unit,
    size: dto.size,
    processingType: dto.processing_type,
    volume: dto.volume,
    originCountry: dto.origin_country,
  };
}

function mapContract(dto?: ContractDTO): DealContractRecord | undefined {
  if (!dto) {
    return undefined;
  }

  return {
    number: dto.number,
    preparedAt: dto.prepared_at,
    signedAt: dto.signed_at,
    signedBy: dto.signed_by,
    signatureRef: dto.signature_ref,
    documentUrl: dto.document_url,
  };
}

function mapDeal(dto: DealDTO, source: DealRecord["source"] = "api"): DealRecord {
  return {
    id: dto.id,
    customerId: dto.customer_id,
    supplierId: dto.supplier_id,
    auctionId: dto.auction_id,
    quantity: dto.quantity,
    unitPrice: dto.unit_price,
    totalAmount: dto.total_amount,
    status: normalizeDealStatus(dto.status),
    type: dto.type,
    createdAt: dto.created_at,
    confirmedAt: dto.confirmed_at,
    contract: mapContract(dto.contract),
    productSnapshot: mapSnapshot(dto.product_snapshot),
    source,
  };
}

function mapProjection(dto: ProjectionDTO, source: DealProjectionRecord["source"] = "api"): DealProjectionRecord {
  return {
    auctionId: dto.auction_id,
    supplierId: dto.supplier_id,
    startPrice: dto.start_price,
    publishedAt: dto.published_at,
    status: dto.status,
    productSnapshot: mapSnapshot(dto.product_snapshot),
    source,
  };
}

function getDealFallbackNote(): string {
  return "Deals service exposes get-by-id and get-by-auction, but no list endpoint yet. The UI merges API reads by known auctions with a local deal mirror.";
}

function canViewDeal(deal: DealRecord, session: UserSession | null): boolean {
  if (!session?.companyId) {
    return false;
  }

  return deal.customerId === session.companyId || deal.supplierId === session.companyId;
}

export async function getProjectionByAuctionId(
  auctionId: string,
  session: UserSession | null,
): Promise<ServiceResult<DealProjectionRecord | null>> {
  try {
    const data = await apiRequest<ProjectionDTO>("deals", `/deal-projections/${auctionId}`, {
      session,
    });
    const projection = upsertDealProjectionStore(mapProjection(data));
    return {
      data: projection,
      meta: { source: "api" },
    };
  } catch (error) {
    if (!isRecoverableApiGap(error)) {
      throw error;
    }
    const fallback = listDealProjectionsStore(auctionId)[0] ?? null;
    return {
      data: fallback,
      meta: mockMeta(
        fallback
          ? "Deal projection API is unavailable for this auction, using local projection mirror."
          : "Deal projection is not available for this auction yet or the backend projection endpoint is not reachable.",
      ),
    };
  }
}

export async function getDealByAuctionId(
  auctionId: string,
  session: UserSession | null,
): Promise<ServiceResult<DealRecord | null>> {
  if (!session?.companyId) {
    return {
      data: null,
      meta: mockMeta("Deal data is visible only to authenticated deal participants."),
    };
  }

  try {
    const data = await apiRequest<DealDTO>("deals", `/deals/by-auction/${auctionId}`, {
      session,
    });
    const mappedDeal = mapDeal(data);
    const deal = canViewDeal(mappedDeal, session) ? upsertDealStore(mappedDeal) : null;
    return {
      data: deal,
      meta: { source: "api" },
    };
  } catch (error) {
    if (!isRecoverableApiGap(error)) {
      throw error;
    }
    const fallback = listDealsStore().find((item) => item.auctionId === auctionId && canViewDeal(item, session)) ?? null;
    return {
      data: fallback,
      meta: mockMeta(
        fallback
          ? "Deal by auction API is unavailable, using local deal mirror."
          : "Deal by auction is not available yet. Showing auction-only context.",
      ),
    };
  }
}

export async function getDealById(
  dealId: string,
  session: UserSession | null,
): Promise<ServiceResult<DealRecord | null>> {
  if (!session?.companyId) {
    return {
      data: null,
      meta: mockMeta("Deal data is visible only to authenticated deal participants."),
    };
  }

  try {
    const data = await apiRequest<DealDTO>("deals", `/deals/${dealId}`, { session });
    const mappedDeal = mapDeal(data);
    const deal = canViewDeal(mappedDeal, session) ? upsertDealStore(mappedDeal) : null;
    return {
      data: deal,
      meta: { source: "api" },
    };
  } catch (error) {
    if (!isRecoverableApiGap(error)) {
      throw error;
    }
    const fallback = listDealsStore().find((item) => item.id === dealId && canViewDeal(item, session)) ?? null;
    return {
      data: fallback,
      meta: mockMeta(
        fallback
          ? "Deal details API is unavailable, using local deal mirror."
          : "Deal details are not available yet.",
      ),
    };
  }
}

export async function listDeals(session: UserSession | null): Promise<ServiceResult<DealRecord[]>> {
  if (!session?.companyId) {
    return {
      data: [],
      meta: mockMeta("Deal list is visible only to authenticated deal participants."),
    };
  }

  const byId = new Map<string, DealRecord>();
  for (const deal of listDealsStore().filter((item) => canViewDeal(item, session))) {
    byId.set(deal.id, deal);
  }

  const auctionIds = Array.from(
    new Set([...listAuctionsStore().map((auction) => auction.id), ...listDealsStore().map((deal) => deal.auctionId)]),
  );
  let sawApi = false;

  await Promise.all(
    auctionIds.map(async (auctionId) => {
      try {
        const data = await apiRequest<DealDTO>("deals", `/deals/by-auction/${auctionId}`, { session });
        const mappedDeal = mapDeal(data);
        if (canViewDeal(mappedDeal, session)) {
          const deal = upsertDealStore(mappedDeal);
          byId.set(deal.id, deal);
        }
        sawApi = true;
      } catch (error) {
        if (!isRecoverableApiGap(error)) {
          throw error;
        }
      }
    }),
  );

  return {
    data: Array.from(byId.values()).sort((left, right) => right.createdAt.localeCompare(left.createdAt)),
    meta: sawApi ? mixedMeta(getDealFallbackNote()) : mockMeta(getDealFallbackNote()),
  };
}

function actionToRequest(input: DealActionInput): { pathSuffix: string; body?: unknown } {
  switch (input.type) {
    case "confirm":
      return { pathSuffix: "confirm" };
    case "prepareContract":
      return {
        pathSuffix: "contract/prepare",
        body: {
          contract_number: input.contractNumber ?? "",
          document_url: input.documentUrl ?? "",
        },
      };
    case "signContract":
      return {
        pathSuffix: "contract/sign",
        body: {
          signature_ref: input.signatureRef,
        },
      };
    case "requestPayment":
      return {
        pathSuffix: "payment/request",
        body: {
          invoice_number: input.invoiceNumber,
          due_date: input.dueDate ? new Date(input.dueDate).toISOString() : undefined,
        },
      };
    case "markPaid":
      return {
        pathSuffix: "payment/mark-paid",
        body: {
          payment_id: input.paymentId,
          payment_type: input.paymentType,
        },
      };
    case "requestShipment":
      return { pathSuffix: "shipment/request" };
    case "markShipped":
      return {
        pathSuffix: "shipment/mark-shipped",
        body: {
          tracking_number: input.trackingNumber,
          carrier: input.carrier,
        },
      };
    case "complete":
      return { pathSuffix: "complete" };
    case "cancel":
      return {
        pathSuffix: "cancel",
        body: {
          reason: input.reason,
        },
      };
    case "updatePrice":
      return {
        pathSuffix: "price",
        body: {
          new_price: input.newPrice,
        },
      };
  }
}

function applyLocalDealAction(
  deal: DealRecord,
  input: DealActionInput,
  session: UserSession | null,
): DealRecord {
  const now = new Date().toISOString();

  switch (input.type) {
    case "confirm":
      return { ...deal, status: "confirmed", confirmedAt: now, source: "mock" };
    case "prepareContract":
      return {
        ...deal,
        status: "contract_prepared",
        contract: {
          ...deal.contract,
          number: input.contractNumber ?? deal.contract?.number,
          documentUrl: input.documentUrl,
          preparedAt: now,
        },
        source: "mock",
      };
    case "signContract":
      return {
        ...deal,
        status: "contract_signed",
        contract: {
          ...deal.contract,
          signedAt: now,
          signedBy: session?.companyId,
          signatureRef: input.signatureRef,
        },
        source: "mock",
      };
    case "requestPayment":
      return { ...deal, status: "payment_requested", source: "mock" };
    case "markPaid":
      return { ...deal, status: "paid", source: "mock" };
    case "requestShipment":
      return { ...deal, status: "shipment_requested", source: "mock" };
    case "markShipped":
      return { ...deal, status: "shipped", source: "mock" };
    case "complete":
      return { ...deal, status: "completed", source: "mock" };
    case "cancel":
      return { ...deal, status: "cancelled", source: "mock" };
    case "updatePrice":
      return {
        ...deal,
        unitPrice: input.newPrice,
        totalAmount: deal.quantity * input.newPrice,
        source: "mock",
      };
  }
}

export async function runDealAction(
  dealId: string,
  input: DealActionInput,
  session: UserSession | null,
): Promise<ServiceResult<DealRecord>> {
  if (!session?.companyId || !session.userId) {
    throw new ApiError("Войдите в профиль, чтобы управлять сделкой", 400, "MISSING_SESSION");
  }

  const locallyKnownDeal = listDealsStore().find((item) => item.id === dealId);
  if (locallyKnownDeal && !canViewDeal(locallyKnownDeal, session)) {
    throw new ApiError("Сделка доступна только ее участникам", 403, "DEAL_FORBIDDEN");
  }

  const request = actionToRequest(input);
  let meta: ServiceMeta = { source: "api" };

  try {
    await apiRequest("deals", `/deals/${dealId}/${request.pathSuffix}`, {
      method: "POST",
      session,
      body: request.body,
    });
  } catch (error) {
    if (!isRecoverableApiGap(error)) {
      throw error;
    }

    const existing = listDealsStore().find((item) => item.id === dealId);
    if (!existing) {
      throw error;
    }
    if (!canViewDeal(existing, session)) {
      throw new ApiError("Сделка доступна только ее участникам", 403, "DEAL_FORBIDDEN");
    }

    const next = upsertDealStore(applyLocalDealAction(existing, input, session));
    addActivity("Сделка обновлена", `${next.id} · ${next.status}`, session);
    return {
      data: next,
      meta: mockMeta("Deal command API is unavailable, local lifecycle mirror was updated."),
    };
  }

  const refreshed = await getDealById(dealId, session);
  if (!refreshed.data) {
    const existing = listDealsStore().find((item) => item.id === dealId);
    if (!existing) {
      throw new ApiError("Сделка обновлена, но read-model еще не вернул запись", 202, "DEAL_READ_MODEL_PENDING");
    }
    if (!canViewDeal(existing, session)) {
      throw new ApiError("Сделка доступна только ее участникам", 403, "DEAL_FORBIDDEN");
    }
    const next = upsertDealStore({ ...applyLocalDealAction(existing, input, session), source: "mixed" });
    meta = mixedMeta("Deal command was accepted, but the read-model is not refreshed yet.");
    addActivity("Сделка обновлена", `${next.id} · ${next.status}`, session);
    return { data: next, meta };
  }

  addActivity("Сделка обновлена", `${refreshed.data.id} · ${refreshed.data.status}`, session);
  return {
    data: refreshed.data,
    meta,
  };
}
