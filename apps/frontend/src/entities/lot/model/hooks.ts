"use client";

import { useQuery } from "@tanstack/react-query";

import { useAuth } from "@/entities/session/model/auth-context";
import { listLots, listProducts } from "@/shared/api/catalog-service";

export function useLotsQuery() {
  const { session } = useAuth();
  return useQuery({
    queryKey: ["lots", session?.companyId],
    queryFn: () => listLots(session ?? null),
    enabled: Boolean(session?.accessToken),
    staleTime: 15_000,
  });
}

export function useProductsQuery() {
  const { session } = useAuth();
  return useQuery({
    queryKey: ["products", session?.companyId],
    queryFn: () => listProducts(session ?? null),
    enabled: Boolean(session?.accessToken),
    staleTime: 30_000,
  });
}
