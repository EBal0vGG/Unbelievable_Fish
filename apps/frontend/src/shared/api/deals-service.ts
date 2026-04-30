import { ApiError, apiRequest, isRecoverableApiGap } from "@/shared/api/http-client";
import { mixedMeta, mockMeta } from "@/shared/api/service-helpers";
import {
  addActivity,
  listAuctionsStore,
  listDealConfirmationsStore,
  listDealProjectionsStore,
  listDealsStore,
  upsertDealConfirmationStore,
  upsertDealProjectionStore,
  upsertDealStore,
} from "@/shared/api/mock-store";
import type {
  DealConfirmationRecord,
  DealConfirmationStage,
  DealConfirmationStatus,
  DealContractRecord,
  DealProjectionRecord,
  DealRecord,
  DealStatus,
  DealVerificationMethod,
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

interface DealConfirmationDTO {
  id: string;
  deal_id: string;
  stage: string;
  requested_by_user_id: string;
  requested_by_company_id: string;
  counterparty_company_id: string;
  status: string;
  verification_method: string;
  verification_token_hash?: string;
  signature_ref?: string;
  requested_at: string;
  approved_at?: string;
  rejected_at?: string;
  expires_at?: string;
  comment?: string;
  reason?: string;
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
  | { type: "requestConfirmation"; stage: DealConfirmationStage; verificationMethod: DealVerificationMethod; comment?: string }
  | { type: "approveConfirmation"; confirmationId: string }
  | { type: "rejectConfirmation"; confirmationId: string; reason?: string }
  | { type: "prepareContract"; contractNumber?: string; documentUrl?: string }
  | { type: "signContract"; signatureRef: string }
  | { type: "requestPayment"; invoiceNumber: string; dueDate?: string }
  | { type: "requestShipment" }
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

function mapConfirmationStatus(value: string): DealConfirmationStatus {
  switch (value) {
    case "pending":
    case "approved":
    case "rejected":
    case "expired":
      return value;
    default:
      return "pending";
  }
}

function mapConfirmationStage(value: string): DealConfirmationStage {
  switch (value) {
    case "confirmed":
    case "paid":
    case "shipped":
    case "completed":
    case "cancelled":
      return value;
    default:
      return "confirmed";
  }
}

function mapVerificationMethod(value: string): DealVerificationMethod {
  switch (value) {
    case "manual":
    case "email":
    case "signature":
      return value;
    default:
      return "manual";
  }
}

function mapConfirmation(
  dto: DealConfirmationDTO,
  source: DealConfirmationRecord["source"] = "api",
): DealConfirmationRecord {
  return {
    id: dto.id,
    dealId: dto.deal_id,
    stage: mapConfirmationStage(dto.stage),
    requestedByUserId: dto.requested_by_user_id,
    requestedByCompanyId: dto.requested_by_company_id,
    counterpartyCompanyId: dto.counterparty_company_id,
    status: mapConfirmationStatus(dto.status),
    verificationMethod: mapVerificationMethod(dto.verification_method),
    verificationTokenHash: dto.verification_token_hash,
    signatureRef: dto.signature_ref,
    requestedAt: dto.requested_at,
    approvedAt: dto.approved_at,
    rejectedAt: dto.rejected_at,
    expiresAt: dto.expires_at,
    comment: dto.comment,
    reason: dto.reason,
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

export async function listDealConfirmations(
  dealId: string,
  session: UserSession | null,
): Promise<ServiceResult<DealConfirmationRecord[]>> {
  if (!session?.companyId) {
    return {
      data: [],
      meta: mockMeta("Подтверждения доступны только участникам сделки."),
    };
  }

  try {
    const data = await apiRequest<DealConfirmationDTO[]>("deals", `/deals/${dealId}/confirmations`, { session });
    const confirmations = data.map((item) => upsertDealConfirmationStore(mapConfirmation(item)));
    return {
      data: confirmations,
      meta: { source: "api" },
    };
  } catch (error) {
    if (!isRecoverableApiGap(error)) {
      throw error;
    }
    return {
      data: listDealConfirmationsStore(dealId),
      meta: mockMeta("Confirmation API is unavailable, using local approval mirror."),
    };
  }
}

function actionToRequest(input: DealActionInput): { pathSuffix: string; body?: unknown } {
  switch (input.type) {
    case "requestConfirmation":
      return {
        pathSuffix: "confirmations",
        body: {
          stage: input.stage,
          verification_method: input.verificationMethod,
          comment: input.comment,
        },
      };
    case "approveConfirmation":
      return { pathSuffix: `confirmations/${input.confirmationId}/approve` };
    case "rejectConfirmation":
      return {
        pathSuffix: `confirmations/${input.confirmationId}/reject`,
        body: {
          reason: input.reason,
        },
      };
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
    case "requestShipment":
      return { pathSuffix: "shipment/request" };
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
    case "requestConfirmation":
      return { ...deal, source: "mock" };
    case "approveConfirmation":
      switch (listDealConfirmationsStore(deal.id).find((item) => item.id === input.confirmationId)?.stage) {
        case "confirmed":
          return { ...deal, status: "confirmed", confirmedAt: now, source: "mock" };
        case "paid":
          return { ...deal, status: "paid", source: "mock" };
        case "shipped":
          return { ...deal, status: "shipped", source: "mock" };
        case "completed":
          return { ...deal, status: "completed", source: "mock" };
        case "cancelled":
          return { ...deal, status: "cancelled", source: "mock" };
        default:
          return { ...deal, source: "mock" };
      }
    case "rejectConfirmation":
      return { ...deal, source: "mock" };
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
    case "requestShipment":
      return { ...deal, status: "shipment_requested", source: "mock" };
    case "updatePrice":
      return {
        ...deal,
        unitPrice: input.newPrice,
        totalAmount: deal.quantity * input.newPrice,
        source: "mock",
      };
  }
}

function applyLocalConfirmationAction(
  deal: DealRecord,
  input: DealActionInput,
  session: UserSession | null,
): void {
  const now = new Date().toISOString();

  switch (input.type) {
    case "requestConfirmation":
      upsertDealConfirmationStore({
        id: `confirmation-${Date.now()}`,
        dealId: deal.id,
        stage: input.stage,
        requestedByUserId: session?.userId ?? "local-user",
        requestedByCompanyId: session?.companyId ?? "local-company",
        counterpartyCompanyId:
          session?.companyId === deal.supplierId ? deal.customerId : deal.supplierId,
        status: "pending",
        verificationMethod: input.verificationMethod,
        requestedAt: now,
        comment: input.comment,
        source: "mock",
      });
      return;
    case "approveConfirmation": {
      const confirmation = listDealConfirmationsStore(deal.id).find((item) => item.id === input.confirmationId);
      if (!confirmation) {
        return;
      }
      upsertDealConfirmationStore({
        ...confirmation,
        status: "approved",
        approvedAt: now,
        source: "mock",
      });
      return;
    }
    case "rejectConfirmation": {
      const confirmation = listDealConfirmationsStore(deal.id).find((item) => item.id === input.confirmationId);
      if (!confirmation) {
        return;
      }
      upsertDealConfirmationStore({
        ...confirmation,
        status: "rejected",
        rejectedAt: now,
        reason: input.reason,
        source: "mock",
      });
      return;
    }
    default:
      return;
  }
}

function actionActivityText(input: DealActionInput, deal: DealRecord): string {
  switch (input.type) {
    case "requestConfirmation":
      return `Создан запрос подтверждения этапа ${input.stage} по сделке ${deal.id}`;
    case "approveConfirmation":
      return `Подтверждение по сделке ${deal.id} одобрено`;
    case "rejectConfirmation":
      return `Подтверждение по сделке ${deal.id} отклонено`;
    default:
      return `${deal.id} · ${deal.status}`;
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

    applyLocalConfirmationAction(existing, input, session);
    const next = upsertDealStore(applyLocalDealAction(existing, input, session));
    addActivity("Сделка обновлена", actionActivityText(input, next), session);
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
    applyLocalConfirmationAction(existing, input, session);
    const next = upsertDealStore({ ...applyLocalDealAction(existing, input, session), source: "mixed" });
    meta = mixedMeta("Deal command was accepted, but the read-model is not refreshed yet.");
    addActivity("Сделка обновлена", actionActivityText(input, next), session);
    return { data: next, meta };
  }

  addActivity("Сделка обновлена", actionActivityText(input, refreshed.data), session);
  return {
    data: refreshed.data,
    meta,
  };
}
