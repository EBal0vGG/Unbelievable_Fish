/**
 * Catalog API: мутации и read-модели. `listProducts` / `listLots` ходят в `GET /products`, `GET /lots`
 * (JWT обязателен); без токена возвращается `mock-store`. `POST /fish` — только admin на сервере и во фронте.
 */
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
import type {
  FishRecord,
  LotRecord,
  LotStatus,
  ProductRecord,
  ServiceResult,
  UserSession,
} from "@/shared/types/domain";

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

interface CatalogProductRow {
  product_id: string;
  fish_id: string;
  seller_company_id: string;
  weight: number;
  unit: string;
  size: string;
  processing_type: string;
  status: string;
}

interface CatalogLotRow {
  lot_id: string;
  product_id: string;
  seller_company_id: string;
  auction_id?: string;
  photo?: string;
  quantity: number;
  start_price: number;
  min_bid_step: number;
  current_price: number;
  final_price?: number;
  status: string;
  auction_starts_at: string;
  auction_ends_at: string;
}

function mapLotStatusFromAPI(value: string): LotStatus {
  const upper = value.toUpperCase();
  if (upper === "DRAFT" || upper === "PUBLISHED" || upper === "CLOSED" || upper === "CANCELLED") {
    return upper;
  }
  return "DRAFT";
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

export async function listProducts(session: UserSession | null): Promise<ServiceResult<ProductRecord[]>> {
  if (!session?.accessToken) {
    return {
      data: listProductsStore(),
      meta: mockMeta("Войдите в систему, чтобы загрузить продукты с catalog API."),
    };
  }

  try {
    const rows = await apiRequest<CatalogProductRow[]>("catalog", "/products", { session });
    const fishPack = await listFish(session);
    const fishById = new Map(fishPack.data.map((f) => [f.id, f.name]));
    const mapped: ProductRecord[] = rows.map((row) => ({
      id: row.product_id,
      fishId: row.fish_id,
      fishName: fishById.get(row.fish_id) ?? row.fish_id,
      ownerCompanyId: row.seller_company_id,
      ownerUserId: session.userId,
      weight: row.weight,
      unit: row.unit,
      size: row.size,
      processingType: row.processing_type,
      status: row.status?.toUpperCase() === "PUBLISHED" ? "PUBLISHED" : "DRAFT",
      source: "api",
    }));
    for (const p of mapped) {
      upsertProductStore(p);
    }
    return { data: mapped, meta: { source: "api" } };
  } catch (error) {
    if (!isRecoverableApiGap(error) || !canFallbackCommand()) {
      throw error;
    }
    return {
      data: listProductsStore(),
      meta: mockMeta("Catalog GET /products недоступен — показаны локальные продукты."),
    };
  }
}

export async function listLots(session: UserSession | null): Promise<ServiceResult<LotRecord[]>> {
  if (!session?.accessToken) {
    return {
      data: listLotsStore(),
      meta: mockMeta("Войдите в систему, чтобы загрузить лоты с catalog API."),
    };
  }

  try {
    const rows = await apiRequest<CatalogLotRow[]>("catalog", "/lots", { session });
    const productPack = await listProducts(session);
    const productById = new Map(productPack.data.map((p) => [p.id, p]));
    const mapped: LotRecord[] = rows.map((row) => {
      const prod = productById.get(row.product_id);
      const startMs = new Date(row.auction_starts_at).getTime();
      const endMs = new Date(row.auction_ends_at).getTime();
      const durationMin =
        !Number.isNaN(startMs) && !Number.isNaN(endMs) && endMs > startMs
          ? Math.max(1, Math.round((endMs - startMs) / 60_000))
          : 60;
      return {
        id: row.lot_id,
        productId: row.product_id,
        productLabel: prod?.fishName ?? row.product_id,
        sellerCompanyId: row.seller_company_id,
        creatorUserId: session.userId,
        photo: row.photo,
        quantity: row.quantity,
        startPrice: row.start_price,
        minBidStep: row.min_bid_step,
        currentPrice: row.current_price,
        finalPrice: row.final_price,
        status: mapLotStatusFromAPI(row.status),
        auctionStartsAt: row.auction_starts_at,
        auctionDurationMinutes: durationMin,
        auctionId: row.auction_id,
        source: "api",
      };
    });
    for (const lot of mapped) {
      upsertLotStore(lot);
    }
    return { data: mapped, meta: { source: "api" } };
  } catch (error) {
    if (!isRecoverableApiGap(error) || !canFallbackCommand()) {
      throw error;
    }
    return {
      data: listLotsStore(),
      meta: mockMeta("Catalog GET /lots недоступен — показаны локальные лоты."),
    };
  }
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
