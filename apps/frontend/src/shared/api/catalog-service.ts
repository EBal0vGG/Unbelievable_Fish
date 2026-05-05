import { ApiError, apiRequest, isRecoverableApiGap } from "@/shared/api/http-client";
import { canFallbackCommand, mixedMeta, mockMeta, withFallback } from "@/shared/api/service-helpers";
import {
  addActivity,
  upsertAuctionStore,
  listFishStore,
  listLotsStore,
  listProductsStore,
  upsertFishStore,
  upsertLotStore,
  upsertProductStore,
} from "@/shared/api/mock-store";
import { isAdminSession } from "@/shared/lib/access";
import { makeClientId } from "@/shared/lib/id";
import type { FishRecord, LotRecord, ProductRecord, ServiceResult, UserSession } from "@/shared/types/domain";

interface CreateFishInput {
  name: string;
  description: string;
}

interface FishListItem {
  id?: string;
  fish_id?: string;
  name: string;
  description: string;
}

interface CreateProductInput {
  fishId: string;
  fishName: string;
  weight: number;
  unit: string;
  size: string;
  processingType: string;
}

interface CreateLotInput {
  productId: string;
  productLabel: string;
  photo?: string;
  quantity: number;
  startPrice: number;
  minBidStep: number;
  auctionStartsAt: string;
  auctionDurationMinutes: number;
}
interface TradingAuctionByLotResponse {
  auction_id?: string;
  lot_id?: string;
  state?: string;
}

const junkFishNames = new Set(["asdasd", "q3123123", "stressfish", "demo fish", "live demo fish", "щука"]);

function normalizedFishName(value: string): string {
  return value.trim().toLowerCase();
}

function normalizeFishList(items: FishRecord[]): FishRecord[] {
  const byName = new Map<string, FishRecord>();
  for (const item of items) {
    const normalized = normalizedFishName(item.name);
    if (!normalized || junkFishNames.has(normalized)) {
      continue;
    }
    if (!byName.has(normalized) || item.source !== "mock") {
      byName.set(normalized, item);
    }
  }
  return Array.from(byName.values()).sort((left, right) => left.name.localeCompare(right.name, "ru"));
}

async function waitAuctionIDByLot(
  lotId: string,
  session: UserSession | null,
  attempts = 12,
  delayMs = 500,
): Promise<string | undefined> {
  for (let i = 0; i < attempts; i += 1) {
    try {
      const result = await apiRequest<TradingAuctionByLotResponse>("trading", `/auctions/by-lot/${lotId}`, {
        method: "GET",
        session,
      });
      if (result.auction_id) {
        return result.auction_id;
      }
    } catch (error) {
      if (!isRecoverableApiGap(error)) {
        throw error;
      }
    }
    await new Promise((resolve) => setTimeout(resolve, delayMs));
  }
  return undefined;
}

export async function listFish(session: UserSession | null): Promise<ServiceResult<FishRecord[]>> {
  // TODO: switch to a real read-model endpoint once GET /fish is exposed by Catalog.
  return withFallback(
    async () => {
      const data = await apiRequest<FishListItem[]>("catalog", "/fish", { session });
      return data.map((item) => ({
        id: item.id ?? item.fish_id ?? makeClientId("fish"),
        name: item.name,
        description: item.description,
        source: "api" as const,
      }));
    },
    () => normalizeFishList(listFishStore()),
    "Catalog list query is not wired in the current backend build, using local fish catalog fallback.",
  ).then((result) => ({ ...result, data: normalizeFishList(result.data) }));
}

export async function listProducts(): Promise<ServiceResult<ProductRecord[]>> {
  return {
    data: listProductsStore(),
    meta: mockMeta(
      "Product list is derived from frontend session storage until Catalog query endpoints are exposed.",
    ),
  };
}

export async function listLots(): Promise<ServiceResult<LotRecord[]>> {
  return {
    data: listLotsStore(),
    meta: mockMeta(
      "Lot list is served from local UI storage because GET /lots list endpoint is not available yet.",
    ),
  };
}

export async function createFish(
  input: CreateFishInput,
  session: UserSession | null,
): Promise<ServiceResult<FishRecord>> {
  if (!isAdminSession(session)) {
    throw new ApiError("Только администратор может создавать рыбу", 403, "ADMIN_ONLY");
  }

  const fallbackFish: FishRecord = {
    id: makeClientId("fish"),
    name: input.name,
    description: input.description,
    source: "mock",
  };

  try {
    const response = await apiRequest<{ fish_id?: string }>("catalog", "/fish", {
      method: "POST",
      session,
      body: input,
    });

    const createdFish = upsertFishStore({
      id: response.fish_id ?? fallbackFish.id,
      name: input.name,
      description: input.description,
      source: response.fish_id ? "api" : "mixed",
    });
    addActivity("Создана рыба", `${createdFish.name} · ${createdFish.id}`, session);

    return {
      data: createdFish,
      meta: response.fish_id
        ? { source: "api" }
        : mixedMeta("Catalog accepted CreateFish but did not return fish_id, local mirror was created."),
    };
  } catch (error) {
    if (!isRecoverableApiGap(error) || !canFallbackCommand()) {
      throw error;
    }

    const createdFish = upsertFishStore(fallbackFish);
    addActivity("Создана рыба", createdFish.name, session);
    return {
      data: createdFish,
      meta: mockMeta("CreateFish fallback is active until the backend route is reachable."),
    };
  }
}

export async function createProduct(
  input: CreateProductInput,
  session: UserSession | null,
): Promise<ServiceResult<ProductRecord>> {
  if (!session?.companyId || !session.userId) {
    throw new ApiError("Сначала сохраните пользовательский контекст", 400, "MISSING_SESSION");
  }

  const fallbackProduct: ProductRecord = {
    id: makeClientId("product"),
    fishId: input.fishId,
    fishName: input.fishName,
    ownerCompanyId: session?.companyId ?? "unknown-company",
    ownerUserId: session?.userId ?? "unknown-user",
    weight: input.weight,
    unit: input.unit,
    size: input.size,
    processingType: input.processingType,
    status: "DRAFT",
    source: "mock",
  };

  try {
    const response = await apiRequest<{ product_id?: string }>("catalog", "/products", {
      method: "POST",
      session,
      body: {
        fish_id: input.fishId,
        weight: input.weight,
        unit: input.unit,
        size: input.size,
        processing_type: input.processingType,
      },
    });

    const createdProduct = upsertProductStore({
      ...fallbackProduct,
      id: response.product_id ?? fallbackProduct.id,
      source: response.product_id ? "api" : "mixed",
    });
    addActivity("Создан продукт", `${createdProduct.fishName} · ${createdProduct.id}`, session);

    return {
      data: createdProduct,
      meta: response.product_id
        ? { source: "api" }
        : mixedMeta("Catalog accepted CreateProduct without returning product_id, local mirror was created."),
    };
  } catch (error) {
    if (!isRecoverableApiGap(error) || !canFallbackCommand()) {
      throw error;
    }

    const createdProduct = upsertProductStore(fallbackProduct);
    addActivity("Создан продукт", createdProduct.fishName, session);
    return {
      data: createdProduct,
      meta: mockMeta("CreateProduct is currently using a local fallback."),
    };
  }
}

export async function publishProduct(
  productId: string,
  session: UserSession | null,
): Promise<ServiceResult<ProductRecord | null>> {
  const existing = listProductsStore().find((item) => item.id === productId) ?? null;

  try {
    await apiRequest("catalog", `/products/${productId}/publish`, {
      method: "POST",
      session,
    });
    if (!existing) {
      return { data: null, meta: { source: "api" } };
    }

    const published = upsertProductStore({ ...existing, status: "PUBLISHED", source: "mixed" });
    addActivity("Продукт опубликован", published.id, session);
    return {
      data: published,
      meta: { source: "api" },
    };
  } catch (error) {
    if (!isRecoverableApiGap(error) || !canFallbackCommand()) {
      throw error;
    }

    if (!existing) {
      return {
        data: null,
        meta: mockMeta("PublishProduct placeholder was used because the API route is unavailable."),
      };
    }

    const published = upsertProductStore({ ...existing, status: "PUBLISHED", source: "mock" });
    addActivity("Продукт опубликован", published.id, session);
    return {
      data: published,
      meta: mockMeta("PublishProduct fallback is active until backend route exposure is completed."),
    };
  }
}

export async function createLot(
  input: CreateLotInput,
  session: UserSession | null,
): Promise<ServiceResult<LotRecord>> {
  if (!session?.companyId || !session.userId) {
    throw new ApiError("Сначала сохраните пользовательский контекст", 400, "MISSING_SESSION");
  }

  const fallbackLot: LotRecord = {
    id: makeClientId("lot"),
    productId: input.productId,
    productLabel: input.productLabel,
    sellerCompanyId: session?.companyId ?? "unknown-company",
    creatorUserId: session?.userId ?? "unknown-user",
    photo: input.photo,
    quantity: input.quantity,
    startPrice: input.startPrice,
    minBidStep: input.minBidStep,
    currentPrice: input.startPrice,
    status: "DRAFT",
    auctionStartsAt: input.auctionStartsAt,
    auctionDurationMinutes: input.auctionDurationMinutes,
    source: "mock",
  };

  try {
    const response = await apiRequest<{ lot_id?: string }>("catalog", "/lots", {
      method: "POST",
      session,
      body: {
        product_id: input.productId,
        photo: input.photo,
        quantity: input.quantity,
        start_price: input.startPrice,
        min_bid_step: input.minBidStep,
        auction_starts_at: input.auctionStartsAt,
        auction_duration_minutes: input.auctionDurationMinutes,
      },
    });

    const createdLot = upsertLotStore({
      ...fallbackLot,
      id: response.lot_id ?? fallbackLot.id,
      source: response.lot_id ? "api" : "mixed",
    });
    addActivity("Создан лот", `${createdLot.productLabel} · ${createdLot.id}`, session);

    return {
      data: createdLot,
      meta: response.lot_id
        ? { source: "api" }
        : mixedMeta("Catalog accepted CreateLot without returning lot_id, local mirror was created."),
    };
  } catch (error) {
    if (!isRecoverableApiGap(error) || !canFallbackCommand()) {
      throw error;
    }

    const createdLot = upsertLotStore(fallbackLot);
    addActivity("Создан лот", createdLot.productLabel, session);
    return {
      data: createdLot,
      meta: mockMeta("Catalog create-lot command is unavailable from the frontend proxy. A local lot mirror was created."),
    };
  }
}

export async function publishLot(
  lotId: string,
  session: UserSession | null,
): Promise<ServiceResult<LotRecord | null>> {
  const existing = listLotsStore().find((item) => item.id === lotId) ?? null;

  try {
    await apiRequest("catalog", `/lots/${lotId}/publish`, {
      method: "POST",
      session,
    });

    if (!existing) {
      return { data: null, meta: { source: "api" } };
    }

    const linkedAuctionId = (await waitAuctionIDByLot(lotId, session)) ?? existing?.auctionId;

    const published = upsertLotStore({
      ...existing,
      status: "PUBLISHED",
      currentPrice: existing.currentPrice ?? existing.startPrice,
      auctionId: linkedAuctionId,
      source: "mixed",
    });
    if (linkedAuctionId) {
      const startsAt = new Date(published.auctionStartsAt);
      const endsAt = new Date(startsAt.getTime() + published.auctionDurationMinutes * 60_000);
      upsertAuctionStore({
        id: linkedAuctionId,
        lotId: published.id,
        sellerCompanyId: published.sellerCompanyId,
        state: "PUBLISHED",
        startsAt: published.auctionStartsAt,
        endsAt: Number.isNaN(endsAt.getTime()) ? new Date().toISOString() : endsAt.toISOString(),
        currentPrice: published.currentPrice ?? published.startPrice,
        source: "mixed",
        statusNote: "Derived from Trading read-model by lot link.",
      });
    }
    addActivity("Лот опубликован", published.id, session);
    return {
      data: published,
      meta: {
        source: "api",
      },
    };
  } catch (error) {
    if (!isRecoverableApiGap(error) || !canFallbackCommand()) {
      throw error;
    }

    if (!existing) {
      return {
        data: null,
        meta: mockMeta("PublishLot placeholder was used because the API route is unavailable."),
      };
    }

    const published = upsertLotStore({
      ...existing,
      status: "PUBLISHED",
      currentPrice: existing.currentPrice ?? existing.startPrice,
      source: "mock",
    });
    addActivity("Лот опубликован", published.id, session);
    return {
      data: published,
      meta: mockMeta("PublishLot fallback is active until backend route exposure is completed."),
    };
  }
}
