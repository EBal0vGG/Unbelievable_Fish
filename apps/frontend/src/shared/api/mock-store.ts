import { makeClientId } from "@/shared/lib/id";
import { readLocalStorage, writeLocalStorage } from "@/shared/lib/storage";
import type {
  ActivityRecord,
  AuctionRecord,
  BidRecord,
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
    id: "lot-salmon-apr-01",
    productId: "product-salmon-fillet",
    productLabel: "Лосось chilled 2-4 / 22 kg",
    sellerCompanyId: "north-sea-llc",
    creatorUserId: "manager-01",
    quantity: 12,
    startPrice: 540000,
    currentPrice: 590000,
    status: "PUBLISHED",
    auctionStartsAt: "2026-04-06T09:00:00.000Z",
    auctionDurationMinutes: 180,
    auctionId: "auction-salmon-apr-01",
    source: "mock",
  },
  {
    id: "lot-pollock-apr-02",
    productId: "product-pollock-block",
    productLabel: "Минтай frozen block / 1000 kg",
    sellerCompanyId: "arctic-export",
    creatorUserId: "seller-02",
    quantity: 28,
    startPrice: 260000,
    finalPrice: 315000,
    status: "CLOSED",
    auctionStartsAt: "2026-04-05T08:00:00.000Z",
    auctionDurationMinutes: 120,
    auctionId: "auction-pollock-apr-02",
    source: "mock",
  },
  {
    id: "lot-herring-apr-03",
    productId: "product-salmon-fillet",
    productLabel: "Сельдь переработка / тестовый драфт",
    sellerCompanyId: "north-sea-llc",
    creatorUserId: "manager-01",
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
    id: "auction-salmon-apr-01",
    lotId: "lot-salmon-apr-01",
    sellerCompanyId: "north-sea-llc",
    state: "PUBLISHED",
    startsAt: "2026-04-06T09:00:00.000Z",
    endsAt: "2026-04-06T12:00:00.000Z",
    currentPrice: 590000,
    leaderCompanyId: "vostok-trade",
    source: "mock",
  },
  {
    id: "auction-pollock-apr-02",
    lotId: "lot-pollock-apr-02",
    sellerCompanyId: "arctic-export",
    state: "WON",
    startsAt: "2026-04-05T08:00:00.000Z",
    endsAt: "2026-04-05T10:00:00.000Z",
    currentPrice: 315000,
    finalPrice: 315000,
    winnerCompanyId: "omega-foods",
    source: "mock",
  },
];

const seedBids: BidRecord[] = [
  {
    auctionId: "auction-salmon-apr-01",
    bidderCompanyId: "borealis-food",
    amount: 560000,
    placedAt: "2026-04-06T09:20:00.000Z",
    source: "mock",
  },
  {
    auctionId: "auction-salmon-apr-01",
    bidderCompanyId: "vostok-trade",
    amount: 590000,
    placedAt: "2026-04-06T09:48:00.000Z",
    source: "mock",
  },
  {
    auctionId: "auction-pollock-apr-02",
    bidderCompanyId: "omega-foods",
    amount: 315000,
    placedAt: "2026-04-05T09:55:00.000Z",
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
  activities: seedActivities,
};

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
