"use client";

import { useQuery } from "@tanstack/react-query";

import {
  getBillingBalance,
  getDealInvoiceByDeal,
  listMySellerPayouts,
  listTopUps,
} from "@/shared/api/billing-service";
import { ApiError } from "@/shared/api/http-client";
import type { UserSession } from "@/shared/types/domain";

/** Polling interval while waiting on async billing (webhook confirm, invoice creation). */
export const BILLING_POLL_MS = 3000;

export function useBillingBalanceQuery(session: UserSession | null) {
  return useQuery({
    queryKey: ["billing-balance", session?.companyId, session?.userId],
    queryFn: () => getBillingBalance(session!),
    enabled: Boolean(session),
    staleTime: 10_000,
  });
}

export function useDealInvoiceBillingQuery(dealId: string, session: UserSession | null, enabled: boolean) {
  return useQuery({
    queryKey: ["billing-invoice", dealId, session?.companyId],
    queryFn: () => getDealInvoiceByDeal(dealId, session!),
    enabled: Boolean(session && dealId && enabled),
    staleTime: 10_000,
    retry: false,
    refetchInterval: (query) => {
      if (!enabled) {
        return false;
      }
      const err = query.state.error;
      if (err instanceof ApiError && err.status === 404) {
        return BILLING_POLL_MS;
      }
      const inv = query.state.data;
      if (inv?.status === "PAYMENT_PENDING") {
        return BILLING_POLL_MS;
      }
      return false;
    },
  });
}

export function useTopUpsQuery(session: UserSession | null) {
  return useQuery({
    queryKey: ["billing-topups", session?.companyId],
    queryFn: () => listTopUps(session!),
    enabled: Boolean(session),
    staleTime: 10_000,
    refetchInterval: (query) => {
      const list = query.state.data?.top_ups ?? [];
      if (list.some((t) => t.status === "PENDING")) {
        return BILLING_POLL_MS;
      }
      return false;
    },
  });
}

export function useSellerPayoutsQuery(
  session: UserSession | null,
  enabled: boolean,
  options?: { dealId?: string; pollUntilPaidForDeal?: boolean },
) {
  const dealId = options?.dealId;
  const poll = Boolean(options?.pollUntilPaidForDeal && dealId);

  return useQuery({
    queryKey: ["billing-payouts", session?.companyId],
    queryFn: () => listMySellerPayouts(session!),
    enabled: Boolean(session && enabled),
    staleTime: 10_000,
    refetchInterval: (query) => {
      if (!enabled || !poll || !dealId) {
        return false;
      }
      const row = query.state.data?.payouts.find((p) => p.deal_id === dealId);
      if (!row || row.status !== "PAID") {
        return BILLING_POLL_MS;
      }
      return false;
    },
  });
}
