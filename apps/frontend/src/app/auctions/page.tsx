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
import { Card } from "@/shared/ui/card";
import { EmptyState } from "@/shared/ui/empty-state";
import { Field } from "@/shared/ui/field";
import { Select } from "@/shared/ui/select";

export default function AuctionsPage() {
  const { session } = useAuth();
  const canCreateAuction = isSellerSession(session);
  const auctionsQuery = useAuctionsQuery(session);
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

  const totals = useMemo(() => {
    const auctions = auctionsQuery.data?.data ?? [];
    return {
      active: auctions.filter((item) => item.state === "PUBLISHED").length,
      finished: auctions.filter((item) => item.state === "WON" || item.state === "CLOSED").length,
      cancelled: auctions.filter((item) => item.state === "CANCELLED").length,
      sellers: sellerOptions.length,
    };
  }, [auctionsQuery.data?.data, sellerOptions.length]);

  return (
    <div className="page-stack">
      <section className="page-hero compact-hero">
        <div>
          <p className="eyebrow">Аукционы</p>
          <h1>Торги</h1>
          <p className="hero-copy">
            Актуальные торговые сессии с синхронизацией статуса по времени завершения и backend read-model.
          </p>
        </div>
      </section>

      <section className="stats-grid">
        <Card className="stat-card stat-card-primary">
          <span>Идет прием ставок</span>
          <strong>{totals.active}</strong>
        </Card>
        <Card className="stat-card">
          <span>Завершены</span>
          <strong>{totals.finished}</strong>
        </Card>
        <Card className="stat-card">
          <span>Отменены</span>
          <strong>{totals.cancelled}</strong>
        </Card>
        <Card className="stat-card">
          <span>Продавцы</span>
          <strong>{totals.sellers}</strong>
        </Card>
      </section>

      <div className="section-heading">
        <p className="eyebrow">Аукционы</p>
        <h2>Поиск и фильтры</h2>
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
          description="Опубликуйте лот в Catalog или снимите фильтры."
          actionHref={canCreateAuction ? "/create/lot" : undefined}
          actionLabel={canCreateAuction ? "Создать лот" : undefined}
        />
      )}
    </div>
  );
}
