export type DataSource = "api" | "mock" | "mixed";
export type UserRole = "admin" | "seller" | "buyer" | "buyer_seller";

export interface UserSession {
  accessToken: string;
  companyId: string;
  userId: string;
  role: UserRole;
  name: string;
  login: string;
  emailVerified?: boolean;
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

export type ProductStatus = "DRAFT" | "PUBLISHED";
export type LotStatus = "DRAFT" | "PUBLISHED" | "CLOSED" | "CANCELLED";
export type AuctionState = "DRAFT" | "PUBLISHED" | "CLOSED" | "WON" | "CANCELLED";
export type DealStatus =
  | "pending"
  | "confirmed"
  | "contract_prepared"
  | "contract_signed"
  | "payment_requested"
  | "paid"
  | "shipment_requested"
  | "shipped"
  | "completed"
  | "cancelled";

export type DealConfirmationStage = "confirmed" | "paid" | "shipped" | "completed" | "cancelled";
export type DealConfirmationStatus = "pending" | "approved" | "rejected" | "expired";
export type DealVerificationMethod = "manual" | "email" | "signature";

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
  ownerCompanyId: string;
  ownerUserId: string;
  weight: number;
  unit: string;
  size: string;
  processingType: string;
  status: ProductStatus;
  source: DataSource;
}

export interface LotRecord {
  id: string;
  productId: string;
  productLabel: string;
  sellerCompanyId: string;
  creatorUserId: string;
  photo?: string;
  quantity: number;
  startPrice: number;
  minBidStep: number;
  currentPrice?: number;
  finalPrice?: number;
  status: LotStatus;
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
  state: AuctionState;
  startsAt: string;
  endsAt: string;
  currentPrice?: number;
  minBidStep?: number;
  finalPrice?: number;
  leaderCompanyId?: string;
  winnerCompanyId?: string;
  chainResultHash?: string;
  chainFinalizeTxHash?: string;
  chainFinalizeStatus?: string;
  chainFinalizeWalletAddress?: string;
  chainFinalizeBlockNumber?: number;
  source: DataSource;
  statusNote?: string;
}

export interface BidRecord {
  auctionId: string;
  bidderCompanyId: string;
  amount: number;
  placedAt: string;
  chainBidHash?: string;
  chainTxHash?: string;
  chainStatus?: string;
  chainWalletAddress?: string;
  chainBlockNumber?: number;
  source: DataSource;
}

export interface DealProjectionRecord {
  auctionId: string;
  supplierId: string;
  startPrice: number;
  publishedAt: string;
  status: string;
  productSnapshot: ProductSnapshot;
  source: DataSource;
}

export interface DealContractRecord {
  number?: string;
  preparedAt?: string;
  signedAt?: string;
  signedBy?: string;
  signatureRef?: string;
  documentUrl?: string;
}

export interface DealRecord {
  id: string;
  customerId: string;
  supplierId: string;
  auctionId: string;
  quantity: number;
  unitPrice: number;
  totalAmount: number;
  status: DealStatus;
  type: string;
  createdAt: string;
  confirmedAt?: string;
  contract?: DealContractRecord;
  productSnapshot: ProductSnapshot;
  source: DataSource;
}

export interface DealConfirmationRecord {
  id: string;
  dealId: string;
  stage: DealConfirmationStage;
  requestedByUserId: string;
  requestedByCompanyId: string;
  counterpartyCompanyId: string;
  status: DealConfirmationStatus;
  verificationMethod: DealVerificationMethod;
  verificationTokenHash?: string;
  signatureRef?: string;
  requestedAt: string;
  approvedAt?: string;
  rejectedAt?: string;
  expiresAt?: string;
  comment?: string;
  reason?: string;
  source: DataSource;
}

export interface ActivityRecord {
  id: string;
  title: string;
  description: string;
  at: string;
  companyId?: string;
  userId?: string;
}

export interface FrontendStore {
  fish: FishRecord[];
  products: ProductRecord[];
  lots: LotRecord[];
  auctions: AuctionRecord[];
  bids: BidRecord[];
  dealProjections: DealProjectionRecord[];
  deals: DealRecord[];
  dealConfirmations: DealConfirmationRecord[];
  activities: ActivityRecord[];
}
