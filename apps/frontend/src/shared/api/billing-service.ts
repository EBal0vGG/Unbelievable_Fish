import { apiRequest } from "@/shared/api/http-client";
import type { UserSession } from "@/shared/types/domain";

export interface BillingBalance {
  company_id: string;
  currency: string;
  available: number;
  held: number;
  total: number;
  /** Present when billing exposes POST …/invoices/{id}/fake-confirm (BILLING_ENABLE_FAKE_PROVIDER). */
  deal_invoice_fake_confirm_enabled?: boolean;
  /** Same flag scope for POST …/top-ups/{id}/fake-confirm. */
  top_up_fake_confirm_enabled?: boolean;
}

export interface DealInvoiceDTO {
  id: string;
  deal_id: string;
  auction_id: string;
  buyer_company_id: string;
  seller_company_id: string;
  goods_amount: number;
  platform_fee_due_amount: number;
  total_amount: number;
  currency: string;
  status: string;
  payment_url: string;
  due_at: string;
  created_at: string;
  paid_at?: string;
}

export interface SellerPayoutDTO {
  payout_id: string;
  deal_id: string;
  invoice_id: string;
  auction_id: string;
  seller_company_id: string;
  buyer_company_id: string;
  amount: number;
  currency: string;
  status: string;
  created_at: string;
  ready_at?: string;
  paid_at?: string;
}

export async function getBillingBalance(session: UserSession): Promise<BillingBalance> {
  return apiRequest<BillingBalance>("billing", "/accounts/me", { session });
}

export async function getDealInvoiceByDeal(dealId: string, session: UserSession): Promise<DealInvoiceDTO> {
  return apiRequest<DealInvoiceDTO>("billing", `/invoices/by-deal/${encodeURIComponent(dealId)}`, { session });
}

export async function fakeConfirmDealInvoice(invoiceId: string, session: UserSession): Promise<void> {
  await apiRequest<void>("billing", `/invoices/${encodeURIComponent(invoiceId)}/fake-confirm`, {
    method: "POST",
    session,
  });
}

export async function listMySellerPayouts(session: UserSession): Promise<{ payouts: SellerPayoutDTO[] }> {
  return apiRequest<{ payouts: SellerPayoutDTO[] }>("billing", "/payouts/me", { session });
}

export interface TopUpDTO {
  id: string;
  company_id: string;
  amount: number;
  currency: string;
  status: string;
  provider?: string;
  confirmation_url?: string;
  created_at: string;
  confirmed_at?: string;
}

export async function createTopUp(
  amount: number,
  session: UserSession,
  currency = "RUB",
): Promise<{ top_up_id: string; status: string; amount: number; currency: string; confirmation_url?: string }> {
  return apiRequest("billing", "/top-ups", {
    method: "POST",
    session,
    body: { amount, currency },
  });
}

export async function listTopUps(session: UserSession): Promise<{ top_ups: TopUpDTO[] }> {
  return apiRequest<{ top_ups: TopUpDTO[] }>("billing", "/top-ups", { session });
}

export async function fakeConfirmTopUp(topUpId: string, session: UserSession): Promise<void> {
  await apiRequest<void>("billing", `/top-ups/${encodeURIComponent(topUpId)}/fake-confirm`, {
    method: "POST",
    session,
  });
}

export async function adminConfirmDealInvoice(invoiceId: string, session: UserSession): Promise<void> {
  await apiRequest<void>("billing", `/admin/invoices/${encodeURIComponent(invoiceId)}/confirm`, {
    method: "POST",
    session,
  });
}

export async function adminExpireDealInvoice(invoiceId: string, session: UserSession): Promise<void> {
  await apiRequest<void>("billing", `/admin/invoices/${encodeURIComponent(invoiceId)}/expire`, {
    method: "POST",
    session,
  });
}

export interface AdminPendingDealInvoiceRow {
  id: string;
  deal_id: string;
  auction_id: string;
  buyer_company_id: string;
  seller_company_id: string;
  status: string;
  goods_amount: number;
  total_amount: number;
  currency: string;
  due_at: string;
  created_at: string;
  provider: string;
  provider_invoice_id?: string;
}

export async function listAdminPendingDealInvoices(
  session: UserSession,
): Promise<{ invoices: AdminPendingDealInvoiceRow[] }> {
  return apiRequest<{ invoices: AdminPendingDealInvoiceRow[] }>("billing", "/admin/invoices/pending", { session });
}

export async function adminMarkPayoutReady(payoutId: string, session: UserSession): Promise<void> {
  await apiRequest("billing", `/admin/payouts/${encodeURIComponent(payoutId)}/ready`, {
    method: "POST",
    session,
  });
}

export async function adminMarkPayoutPaid(payoutId: string, session: UserSession): Promise<void> {
  await apiRequest("billing", `/admin/payouts/${encodeURIComponent(payoutId)}/paid`, {
    method: "POST",
    session,
  });
}

export interface AdminPayoutQueueRow {
  payout_id: string;
  deal_id: string;
  invoice_id: string;
  auction_id: string;
  seller_company_id: string;
  buyer_company_id: string;
  seller_company_name: string;
  buyer_company_name: string;
  amount: number;
  currency: string;
  status: string;
  invoice_status: string;
  created_at: string;
  ready_at?: string;
  paid_at?: string;
  failed_at?: string;
  cancelled_at?: string;
}

export async function listAdminPayoutQueue(session: UserSession): Promise<{ payouts: AdminPayoutQueueRow[] }> {
  return apiRequest<{ payouts: AdminPayoutQueueRow[] }>("billing", "/admin/payouts", { session });
}

export async function adminMarkPayoutFailed(payoutId: string, session: UserSession): Promise<void> {
  await apiRequest("billing", `/admin/payouts/${encodeURIComponent(payoutId)}/failed`, {
    method: "POST",
    session,
  });
}
