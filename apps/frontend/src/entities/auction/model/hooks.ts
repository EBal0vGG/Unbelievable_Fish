"use client";

import { useQuery } from "@tanstack/react-query";

import { getAuctionDetails, listAuctions } from "@/shared/api/trading-service";
import type { UserSession } from "@/shared/types/domain";

export function useAuctionsQuery() {
  return useQuery({
    queryKey: ["auctions"],
    queryFn: () => listAuctions(),
    staleTime: 10_000,
  });
}

export function useAuctionDetailsQuery(auctionId: string, session: UserSession | null) {
  return useQuery({
    queryKey: ["auction", auctionId, session?.companyId, session?.userId],
    queryFn: () => getAuctionDetails(auctionId, session),
    enabled: Boolean(auctionId),
    refetchInterval: 15_000,
  });
}
