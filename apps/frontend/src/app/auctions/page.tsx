"use client";

import { useDeferredValue, useMemo, useState } from "react";

import { useAuctionsQuery } from "@/entities/auction/model/hooks";
import { AuctionCard } from "@/entities/auction/ui/auction-card";
import { useFishCatalogQuery } from "@/entities/fish/model/hooks";
import { useLotsQuery, useProductsQuery } from "@/entities/lot/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { isSellerSession } from "@/shared/lib/access";
import { auctionStateLabels } from "@/shared/lib/labels";
import { EmptyState } from "@/shared/ui/empty-state";
import { Field } from "@/shared/ui/field";
import { Select } from "@/shared/ui/select";

export default function AuctionsPage() {
  const { session } = useAuth();
  const canCreateAuction = isSellerSession(session);
  const auctionsQuery = useAuctionsQuery();
  const lotsQuery = useLotsQuery();
  const productsQuery = useProductsQuery();
  const fishQuery = useFishCatalogQuery(session);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [sellerFilter, setSellerFilter] = useState("all");
  const deferredSearch = useDeferredValue(search);

  const productMap = useMemo(
    () => new Map((productsQuery.data?.data ?? []).map((product) => [product.id, product])),
    [productsQuery.data?.data],
  );
  const fishMap = useMemo(
    () => new Map((fishQuery.data?.data ?? []).map((fish) => [fish.id, fish])),
    [fishQuery.data?.data],
  );
  const lotMap = useMemo(
    () => new Map((lotsQuery.data?.data ?? []).map((lot) => [lot.id, lot])),
    [lotsQuery.data?.data],
  );
  const sellerOptions = useMemo(() => {
    const sellerIds = new Set<string>();
    for (const item of auctionsQuery.data?.data ?? []) {
      const lot = lotMap.get(item.lotId);
      const sellerId = item.sellerCompanyId ?? lot?.sellerCompanyId;
      if (sellerId) {
        sellerIds.add(sellerId);
      }
    }
    return Array.from(sellerIds).sort();
  }, [auctionsQuery.data?.data, lotMap]);

  const items = useMemo(() => {
    return (auctionsQuery.data?.data ?? []).filter((item) => {
      if (item.state === "DRAFT") {
        return false;
      }
      const lot = lotMap.get(item.lotId);
      const product = lot ? productMap.get(lot.productId) : undefined;
      const fish = product ? fishMap.get(product.fishId) : undefined;
      const searchMatch = `${item.id} ${item.lotId} ${item.state} ${item.sellerCompanyId ?? lot?.sellerCompanyId ?? ""} ${lot?.productLabel ?? ""} ${product?.fishName ?? ""} ${fish?.description ?? ""}`
        .toLowerCase()
        .includes(deferredSearch.toLowerCase());
      const statusMatch = status === "all" || item.state === status;
      const sellerMatch =
        sellerFilter === "all" ||
        (item.sellerCompanyId ?? lot?.sellerCompanyId ?? "") === sellerFilter;
      return searchMatch && statusMatch && sellerMatch;
    });
  }, [auctionsQuery.data?.data, deferredSearch, fishMap, lotMap, productMap, sellerFilter, status]);

  return (
    <div className="page-stack">
      <div className="page-heading">
        <p className="eyebrow">Аукционы</p>
        <h1>Аукционы</h1>
      </div>

      <FilterBar
        search={search}
        onSearchChange={setSearch}
        status={status}
        onStatusChange={setStatus}
        statusOptions={[
          { label: "Все статусы", value: "all" },
          { label: auctionStateLabels.DRAFT, value: "DRAFT" },
          { label: auctionStateLabels.PUBLISHED, value: "PUBLISHED" },
          { label: auctionStateLabels.CLOSED, value: "CLOSED" },
          { label: auctionStateLabels.WON, value: "WON" },
          { label: auctionStateLabels.CANCELLED, value: "CANCELLED" },
        ]}
        source="all"
        onSourceChange={() => undefined}
        showSource={false}
        searchPlaceholder="Номер аукциона, лот, продукт или компания"
        extraFilters={
          <Field label="Продавец">
            <Select value={sellerFilter} onChange={(event) => setSellerFilter(event.target.value)}>
              <option value="all">Все</option>
              {sellerOptions.map((sellerId) => (
                <option key={sellerId} value={sellerId}>
                  {sellerId}
                </option>
              ))}
            </Select>
          </Field>
        }
      />

      {items.length ? (
        <div className="card-grid card-grid-2">
          {items.map((item) => {
            const lot = lotMap.get(item.lotId);
            const product = lot ? productMap.get(lot.productId) : undefined;

            return (
              <AuctionCard
                key={item.id}
                auction={item}
                fishName={product?.fishName}
                productLabel={lot?.productLabel}
                photo={lot?.photo}
                sellerCompanyId={item.sellerCompanyId ?? lot?.sellerCompanyId}
              />
            );
          })}
        </div>
      ) : (
        <EmptyState
          title="Аукционы не найдены"
          description="Создайте аукцион или снимите фильтры."
          actionHref={canCreateAuction ? "/create/auction" : undefined}
          actionLabel={canCreateAuction ? "Создать аукцион" : undefined}
        />
      )}
    </div>
  );
}
