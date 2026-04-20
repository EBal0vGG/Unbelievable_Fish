import { makeClientId } from "@/shared/lib/id";
import { readLocalStorage, writeLocalStorage } from "@/shared/lib/storage";
import type {
  ActivityRecord,
  AuctionRecord,
  BidRecord,
  DealProjectionRecord,
  DealRecord,
  FishRecord,
  FrontendStore,
  LotRecord,
  ProductRecord,
  UserSession,
} from "@/shared/types/domain";

const STORE_KEY = "uf:frontend-store";

const seedFish: FishRecord[] = [
  {
    id: "fish-salmon-atlantic",
    name: "Атлантический лосось",
    description: "Филе для B2B-поставок, охлажденная линия.",
    source: "mock",
  },
  {
    id: "fish-pollock-far-east",
    name: "Минтай дальневосточный",
    description: "Экспортный поток, фасовка под оптовый канал.",
    source: "mock",
  },
  {
    id: "fish-herring-north",
    name: "Сельдь северная",
    description: "Сырье для переработки и HoReCa.",
    source: "mock",
  },
];

const seedProducts: ProductRecord[] = [
  {
    id: "product-salmon-fillet",
    fishId: "fish-salmon-atlantic",
    fishName: "Атлантический лосось",
    ownerCompanyId: "north-sea-llc",
    ownerUserId: "manager-01",
    weight: 22,
    unit: "kg",
    size: "2-4",
    processingType: "chilled",
    status: "PUBLISHED",
    source: "mock",
  },
  {
    id: "product-pollock-block",
    fishId: "fish-pollock-far-east",
    fishName: "Минтай дальневосточный",
    ownerCompanyId: "arctic-export",
    ownerUserId: "seller-02",
    weight: 1000,
    unit: "kg",
    size: "block",
    processingType: "frozen",
    status: "PUBLISHED",
    source: "mock",
  },
];

const seedLots: LotRecord[] = [
  {
    id: "lot-salmon-apr-20",
    productId: "product-salmon-fillet",
    productLabel: "Лосось chilled 2-4 / 22 kg",
    sellerCompanyId: "north-sea-llc",
    creatorUserId: "manager-01",
    photo: "https://images.unsplash.com/photo-1519708227418-c8fd9a32b7a2?auto=format&fit=crop&w=900&q=80",
    quantity: 12,
    startPrice: 540000,
    currentPrice: 590000,
    status: "PUBLISHED",
    auctionStartsAt: "2026-04-20T08:00:00.000Z",
    auctionDurationMinutes: 1680,
    auctionId: "auction-salmon-apr-20",
    source: "mock",
  },
  {
    id: "lot-pollock-apr-18",
    productId: "product-pollock-block",
    productLabel: "Минтай frozen block / 1000 kg",
    sellerCompanyId: "arctic-export",
    creatorUserId: "seller-02",
    photo: "https://main-cdn.sbermegamarket.ru/big2/hlr-system/187/133/181/810/182/232/100061076386b0.jpg",
    quantity: 28,
    startPrice: 260000,
    finalPrice: 315000,
    status: "CLOSED",
    auctionStartsAt: "2026-04-18T08:00:00.000Z",
    auctionDurationMinutes: 120,
    auctionId: "auction-pollock-apr-18",
    source: "mock",
  },
  {
    id: "lot-herring-apr-03",
    productId: "product-salmon-fillet",
    productLabel: "Сельдь переработка / тестовый драфт",
    sellerCompanyId: "north-sea-llc",
    creatorUserId: "manager-01",
    photo: "https://images.unsplash.com/photo-1506368249639-73a05d6f6488?auto=format&fit=crop&w=900&q=80",
    quantity: 18,
    startPrice: 180000,
    status: "DRAFT",
    auctionStartsAt: "2026-04-07T10:00:00.000Z",
    auctionDurationMinutes: 60,
    source: "mock",
    notes: "Temporary placeholder until Catalog listing/query endpoints are ready.",
  },
];

const seedAuctions: AuctionRecord[] = [
  {
    id: "auction-salmon-apr-20",
    lotId: "lot-salmon-apr-20",
    sellerCompanyId: "north-sea-llc",
    state: "PUBLISHED",
    startsAt: "2026-04-20T08:00:00.000Z",
    endsAt: "2026-04-21T12:00:00.000Z",
    currentPrice: 590000,
    leaderCompanyId: "vostok-trade",
    source: "mock",
  },
  {
    id: "auction-pollock-apr-18",
    lotId: "lot-pollock-apr-18",
    sellerCompanyId: "arctic-export",
    state: "WON",
    startsAt: "2026-04-18T08:00:00.000Z",
    endsAt: "2026-04-18T10:00:00.000Z",
    currentPrice: 315000,
    finalPrice: 315000,
    winnerCompanyId: "omega-foods",
    source: "mock",
  },
];

const seedBids: BidRecord[] = [
  {
    auctionId: "auction-salmon-apr-20",
    bidderCompanyId: "borealis-food",
    amount: 560000,
    placedAt: "2026-04-20T09:20:00.000Z",
    source: "mock",
  },
  {
    auctionId: "auction-salmon-apr-20",
    bidderCompanyId: "vostok-trade",
    amount: 590000,
    placedAt: "2026-04-20T09:48:00.000Z",
    source: "mock",
  },
  {
    auctionId: "auction-pollock-apr-18",
    bidderCompanyId: "omega-foods",
    amount: 315000,
    placedAt: "2026-04-18T09:55:00.000Z",
    source: "mock",
  },
];

const seedDealProjections: DealProjectionRecord[] = [
  {
    auctionId: "auction-salmon-apr-20",
    supplierId: "north-sea-llc",
    startPrice: 540000,
    publishedAt: "2026-04-20T07:45:00.000Z",
    status: "active",
    productSnapshot: {
      productId: "product-salmon-fillet",
      name: "Атлантический лосось",
      description: "Филе для B2B-поставок, охлажденная линия.",
      category: "premium chilled",
      weight: 22,
      unit: "kg",
      size: "2-4",
      processingType: "chilled",
      volume: 12,
      originCountry: "NO",
    },
    source: "mock",
  },
  {
    auctionId: "auction-pollock-apr-18",
    supplierId: "arctic-export",
    startPrice: 260000,
    publishedAt: "2026-04-18T07:40:00.000Z",
    status: "converted",
    productSnapshot: {
      productId: "product-pollock-block",
      name: "Минтай дальневосточный",
      description: "Экспортный поток, фасовка под оптовый канал.",
      category: "frozen block",
      weight: 1000,
      unit: "kg",
      size: "block",
      processingType: "frozen",
      volume: 28,
      originCountry: "RU",
    },
    source: "mock",
  },
];

const seedDeals: DealRecord[] = [
  {
    id: "deal-pollock-apr-18",
    customerId: "omega-foods",
    supplierId: "arctic-export",
    auctionId: "auction-pollock-apr-18",
    quantity: 28,
    unitPrice: 315000,
    totalAmount: 8820000,
    status: "contract_signed",
    type: "auction",
    createdAt: "2026-04-18T10:01:00.000Z",
    confirmedAt: "2026-04-18T10:08:00.000Z",
    contract: {
      number: "CNT-2026-0418",
      preparedAt: "2026-04-18T11:00:00.000Z",
      signedAt: "2026-04-18T12:35:00.000Z",
      signedBy: "omega-foods",
      signatureRef: "SIG-OF-0418",
      documentUrl: "https://contracts.example/deal-pollock-apr-18",
    },
    productSnapshot: seedDealProjections[1].productSnapshot,
    source: "mock",
  },
];

const seedActivities: ActivityRecord[] = [
  {
    id: "activity-seed-1",
    title: "Платформа готова к работе",
    description: "Данные витрины обновлены.",
    at: "2026-04-06T07:50:00.000Z",
  },
];

const seedStore: FrontendStore = {
  fish: seedFish,
  products: seedProducts,
  lots: seedLots,
  auctions: seedAuctions,
  bids: seedBids,
  dealProjections: seedDealProjections,
  deals: seedDeals,
  activities: seedActivities,
};

const seedLotPhotos = new Map(seedLots.map((lot) => [lot.id, lot.photo]));

function cloneStore(store: FrontendStore): FrontendStore {
  return JSON.parse(JSON.stringify(store)) as FrontendStore;
}

function migrateStore(store: FrontendStore): FrontendStore {
  return {
    ...store,
    products: store.products.map((product) => ({
      ...product,
      ownerCompanyId: product.ownerCompanyId ?? "legacy-company",
      ownerUserId: product.ownerUserId ?? "legacy-user",
    })),
    lots: store.lots.map((lot) => ({
      ...lot,
      creatorUserId: lot.creatorUserId ?? "legacy-user",
      photo: lot.photo ?? seedLotPhotos.get(lot.id),
    })),
    dealProjections: (store.dealProjections ?? seedDealProjections).map((projection) => ({
      ...projection,
      source: projection.source ?? "mock",
    })),
    deals: (store.deals ?? seedDeals).map((deal) => ({
      ...deal,
      source: deal.source ?? "mock",
      productSnapshot:
        deal.productSnapshot ??
        seedDealProjections.find((projection) => projection.auctionId === deal.auctionId)?.productSnapshot ??
        {
          productId: "unknown-product",
          name: "Продукт",
          description: "",
          category: "",
          weight: 0,
          unit: "",
          size: "",
          processingType: "",
          volume: deal.quantity ?? 0,
          originCountry: "",
        },
    })),
    activities: store.activities.map((activity) => ({
      ...activity,
      companyId: activity.companyId,
      userId: activity.userId,
    })),
  };
}

function upsertById<T extends { id: string }>(items: T[], nextItem: T): T[] {
  const existingIndex = items.findIndex((item) => item.id === nextItem.id);
  if (existingIndex === -1) {
    return [nextItem, ...items];
  }

  const copy = [...items];
  copy[existingIndex] = nextItem;
  return copy;
}

export function getFrontendStore(): FrontendStore {
  return migrateStore(readLocalStorage(STORE_KEY, cloneStore(seedStore)));
}

export function saveFrontendStore(store: FrontendStore): void {
  writeLocalStorage(STORE_KEY, store);
}

export function listFishStore(): FishRecord[] {
  return getFrontendStore().fish;
}

export function listProductsStore(): ProductRecord[] {
  return getFrontendStore().products;
}

export function listLotsStore(): LotRecord[] {
  return getFrontendStore().lots;
}

export function listAuctionsStore(): AuctionRecord[] {
  return getFrontendStore().auctions;
}

export function listBidsStore(auctionId?: string): BidRecord[] {
  const bids = getFrontendStore().bids;
  if (!auctionId) {
    return bids;
  }
  return bids.filter((item) => item.auctionId === auctionId);
}

export function listDealProjectionsStore(auctionId?: string): DealProjectionRecord[] {
  const projections = getFrontendStore().dealProjections;
  if (!auctionId) {
    return projections;
  }
  return projections.filter((item) => item.auctionId === auctionId);
}

export function listDealsStore(): DealRecord[] {
  return getFrontendStore().deals;
}

export function listActivitiesStore(session?: UserSession | null): ActivityRecord[] {
  return getFrontendStore()
    .activities
    .filter((activity) => {
      if (!session) {
        return false;
      }

      return activity.companyId === session.companyId && activity.userId === session.userId;
    })
    .sort((left, right) => right.at.localeCompare(left.at));
}

export function addActivity(
  title: string,
  description: string,
  session?: UserSession | null,
): void {
  const store = getFrontendStore();
  store.activities = [
    {
      id: makeClientId("activity"),
      title,
      description,
      at: new Date().toISOString(),
      companyId: session?.companyId,
      userId: session?.userId,
    },
    ...store.activities,
  ].slice(0, 40);
  saveFrontendStore(store);
}

export function upsertFishStore(item: FishRecord): FishRecord {
  const store = getFrontendStore();
  store.fish = upsertById(store.fish, item);
  saveFrontendStore(store);
  return item;
}

export function upsertProductStore(item: ProductRecord): ProductRecord {
  const store = getFrontendStore();
  store.products = upsertById(store.products, item);
  saveFrontendStore(store);
  return item;
}

export function upsertLotStore(item: LotRecord): LotRecord {
  const store = getFrontendStore();
  store.lots = upsertById(store.lots, item);
  saveFrontendStore(store);
  return item;
}

export function upsertAuctionStore(item: AuctionRecord): AuctionRecord {
  const store = getFrontendStore();
  store.auctions = upsertById(store.auctions, item);
  store.lots = store.lots.map((lot) =>
    lot.id === item.lotId && lot.auctionId !== item.id
      ? {
          ...lot,
          auctionId: item.id,
        }
      : lot,
  );
  saveFrontendStore(store);
  return item;
}

export function upsertDealProjectionStore(item: DealProjectionRecord): DealProjectionRecord {
  const store = getFrontendStore();
  store.dealProjections = upsertById(
    store.dealProjections.map((projection) => ({
      ...projection,
      id: projection.auctionId,
    })),
    {
      ...item,
      id: item.auctionId,
    },
  ).map(({ id: _id, ...projection }) => projection);
  saveFrontendStore(store);
  return item;
}

export function upsertDealStore(item: DealRecord): DealRecord {
  const store = getFrontendStore();
  store.deals = upsertById(store.deals, item);
  saveFrontendStore(store);
  return item;
}

export function appendBidStore(item: BidRecord, options?: { endsAt?: string }): BidRecord {
  const store = getFrontendStore();
  const auction = store.auctions.find((entry) => entry.id === item.auctionId);

  store.bids = [item, ...store.bids];
  store.auctions = store.auctions.map((auction) =>
    auction.id === item.auctionId
      ? {
          ...auction,
          currentPrice: item.amount,
          leaderCompanyId: item.bidderCompanyId,
          endsAt: options?.endsAt ?? auction.endsAt,
        }
      : auction,
  );
  if (auction) {
    store.lots = store.lots.map((lot) =>
      lot.id === auction.lotId || lot.auctionId === item.auctionId
        ? {
            ...lot,
            auctionId: item.auctionId,
            currentPrice: item.amount,
          }
        : lot,
    );
  }
  saveFrontendStore(store);
  return item;
}
