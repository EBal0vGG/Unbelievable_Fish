"use client";

import { useQuery } from "@tanstack/react-query";

import { getDealById, listDealConfirmations, listDeals } from "@/shared/api/deals-service";
import type { UserSession } from "@/shared/types/domain";

export function useDealsQuery(session: UserSession | null) {
  return useQuery({
    queryKey: ["deals", session?.companyId, session?.userId],
    queryFn: () => listDeals(session),
    staleTime: 15_000,
  });
}

export function useDealDetailsQuery(dealId: string, session: UserSession | null) {
  return useQuery({
    queryKey: ["deal", dealId, session?.companyId, session?.userId],
    queryFn: () => getDealById(dealId, session),
    enabled: Boolean(dealId),
    refetchInterval: 20_000,
  });
}

export function useDealConfirmationsQuery(dealId: string, session: UserSession | null) {
  return useQuery({
    queryKey: ["deal-confirmations", dealId, session?.companyId, session?.userId],
    queryFn: () => listDealConfirmations(dealId, session),
    enabled: Boolean(dealId),
    refetchInterval: 20_000,
  });
}
