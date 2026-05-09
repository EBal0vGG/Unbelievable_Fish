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
