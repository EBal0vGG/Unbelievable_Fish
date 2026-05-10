"use client";

import { useDeferredValue, useMemo, useState } from "react";

import { useAuctionsQuery } from "@/entities/auction/model/hooks";
import { useFishCatalogQuery } from "@/entities/fish/model/hooks";
import { useLotsQuery, useProductsQuery } from "@/entities/lot/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { isSellerSession } from "@/shared/lib/access";
import { displayCompany } from "@/shared/lib/display";
import { formatDateTime, formatMoney, shortId } from "@/shared/lib/format";
import { auctionStateLabels } from "@/shared/lib/labels";
import { buttonStyles } from "@/shared/ui/button";
import { EmptyState } from "@/shared/ui/empty-state";
import { Field } from "@/shared/ui/field";
import { PageHeader } from "@/shared/ui/page-header";
import { SectionCard } from "@/shared/ui/section-card";
import { Select } from "@/shared/ui/select";
import { StatCard } from "@/shared/ui/stat-card";
import { StatusBadge } from "@/shared/ui/status-badge";
import Link from "next/link";

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
      <PageHeader
        compact
        eyebrow="Аукционы"
        title="Торги"
        description="Биржевая лента торговых сессий с текущими ставками, продавцами и статусами."
        metrics={
          <>
            <div>
              <span>Активные</span>
              <strong>{totals.active}</strong>
            </div>
            <div>
              <span>Продавцы</span>
              <strong>{totals.sellers}</strong>
            </div>
          </>
        }
      />

      <section className="stats-grid">
        <StatCard tone="primary" label="Идет прием ставок" value={totals.active} />
        <StatCard label="Завершены" value={totals.finished} />
        <StatCard label="Отменены" value={totals.cancelled} />
        <StatCard tone="accent" label="Продавцы" value={totals.sellers} />
      </section>

      <SectionCard eyebrow="Аукционы" title="Поиск и фильтры">
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
      </SectionCard>

      {items.length ? (
        <section className="data-panel auction-feed">
          <div className="data-list">
            {items.map((item) => {
              const lot = lotMap.get(item.lotId);
              const product = lot ? productMap.get(lot.productId) : undefined;
              const rowTone =
                item.state === "PUBLISHED"
                  ? "data-row-active"
                  : item.state === "CANCELLED"
                    ? "data-row-cancelled"
                    : "data-row-complete";

              return (
                <div className={`data-row ${rowTone}`} key={item.id}>
                  <div className="auction-thumb">
                    {lot?.photo ? (
                      <img alt={lot.productLabel} src={lot.photo} />
                    ) : (
                      <span>{product?.fishName?.slice(0, 2).toUpperCase() ?? "UF"}</span>
                    )}
                  </div>
                  <div className="data-row-main">
                    <h3>{lot?.productLabel ?? product?.fishName ?? `Аукцион ${shortId(item.id)}`}</h3>
                    <p>Сессия #{shortId(item.id)} · лот {shortId(item.lotId)}</p>
                  </div>
                  <div className="data-cell">
                    <span>Статус</span>
                    <strong>
                      <StatusBadge status={item.state} label={auctionStateLabels[item.state]} />
                    </strong>
                  </div>
                  <div className="data-cell">
                    <span>Цена</span>
                    <strong>{formatMoney(item.currentPrice ?? item.finalPrice)}</strong>
                  </div>
                  <div className="data-cell">
                    <span>Продавец / окончание</span>
                    <strong>
                      {displayCompany(item.sellerCompanyId ?? lot?.sellerCompanyId)} · {formatDateTime(item.endsAt)}
                    </strong>
                  </div>
                  <div className="data-actions">
                    <Link
                      className={buttonStyles({ variant: "secondary", size: "sm" })}
                      href={`/auctions/${item.id}`}
                    >
                      Открыть аукцион
                    </Link>
                  </div>
                </div>
              );
            })}
          </div>
        </section>
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
