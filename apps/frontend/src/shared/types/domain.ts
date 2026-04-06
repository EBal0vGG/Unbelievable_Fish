export type DataSource = "api" | "mock" | "mixed";

export interface UserSession {
  companyId: string;
  userId: string;
  mode: "login" | "register";
  updatedAt: string;
}

export interface ServiceMeta {
  source: DataSource;
  placeholder?: boolean;
  note?: string;
}

export interface ServiceResult<T> {
  data: T;
  meta: ServiceMeta;
}

export interface ProductSnapshot {
  productId: string;
  name: string;
  description: string;
  category: string;
  weight: number;
  unit: string;
  size: string;
  processingType: string;
  volume: number;
  originCountry: string;
}

export interface FishRecord {
  id: string;
  name: string;
  description: string;
  source: DataSource;
}

export interface ProductRecord {
  id: string;
  fishId: string;
  fishName: string;
  weight: number;
  unit: string;
  size: string;
  processingType: string;
  status: "DRAFT" | "PUBLISHED" | "UNPUBLISHED";
  source: DataSource;
}

export interface LotRecord {
  id: string;
  productId: string;
  productLabel: string;
  sellerCompanyId: string;
  photo?: string;
  quantity: number;
  startPrice: number;
  currentPrice?: number;
  finalPrice?: number;
  status: "DRAFT" | "PUBLISHED" | "CLOSED" | "CANCELLED";
  auctionStartsAt: string;
  auctionDurationMinutes: number;
  auctionId?: string;
  source: DataSource;
  notes?: string;
}

export interface AuctionRecord {
  id: string;
  lotId: string;
  sellerCompanyId?: string;
  state: "DRAFT" | "PUBLISHED" | "CLOSED" | "WON" | "CANCELLED";
  startsAt: string;
  endsAt: string;
  currentPrice?: number;
  finalPrice?: number;
  leaderCompanyId?: string;
  winnerCompanyId?: string;
  source: DataSource;
  statusNote?: string;
}

export interface BidRecord {
  auctionId: string;
  bidderCompanyId: string;
  amount: number;
  placedAt: string;
  source: DataSource;
}

export interface DealProjectionRecord {
  auctionId: string;
  supplierId: string;
  startPrice: number;
  publishedAt: string;
  status: string;
  productSnapshot: ProductSnapshot;
}

export interface DealRecord {
  id: string;
  customerId: string;
  supplierId: string;
  auctionId: string;
  quantity: number;
  unitPrice: number;
  totalAmount: number;
  status: string;
  type: string;
  createdAt: string;
}

export interface ActivityRecord {
  id: string;
  title: string;
  description: string;
  at: string;
}

export interface FrontendStore {
  fish: FishRecord[];
  products: ProductRecord[];
  lots: LotRecord[];
  auctions: AuctionRecord[];
  bids: BidRecord[];
  activities: ActivityRecord[];
}
