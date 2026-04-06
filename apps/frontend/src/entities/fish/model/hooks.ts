"use client";

import { useQuery } from "@tanstack/react-query";

import { listFish } from "@/shared/api/catalog-service";
import type { UserSession } from "@/shared/types/domain";

export function useFishCatalogQuery(session: UserSession | null) {
  return useQuery({
    queryKey: ["fish-catalog", session?.companyId, session?.userId],
    queryFn: () => listFish(session),
    staleTime: 30_000,
  });
}
