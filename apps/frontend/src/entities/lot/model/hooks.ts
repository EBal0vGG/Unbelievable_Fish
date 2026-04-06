"use client";

import { useQuery } from "@tanstack/react-query";

import { listLots, listProducts } from "@/shared/api/catalog-service";

export function useLotsQuery() {
  return useQuery({
    queryKey: ["lots"],
    queryFn: () => listLots(),
    staleTime: 15_000,
  });
}

export function useProductsQuery() {
  return useQuery({
    queryKey: ["products"],
    queryFn: () => listProducts(),
    staleTime: 30_000,
  });
}
