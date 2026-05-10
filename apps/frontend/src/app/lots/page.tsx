"use client";

import { useDeferredValue, useMemo, useState } from "react";

import { useLotsQuery } from "@/entities/lot/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { isOwnedLot, isSellerSession } from "@/shared/lib/access";
import { displayCompany } from "@/shared/lib/display";
import { formatDateTime, formatMoney, shortId } from "@/shared/lib/format";
import { lotStatusLabels } from "@/shared/lib/labels";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { buttonStyles } from "@/shared/ui/button";
import { EmptyState } from "@/shared/ui/empty-state";
import { Field } from "@/shared/ui/field";
import { PageHeader } from "@/shared/ui/page-header";
import { SectionCard } from "@/shared/ui/section-card";
import { Select } from "@/shared/ui/select";
import { StatusBadge } from "@/shared/ui/status-badge";
import Link from "next/link";

export default function LotsPage() {
  const { session } = useAuth();
  const canCreateSupply = isSellerSession(session);
  const lotsQuery = useLotsQuery();
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [auctionLink, setAuctionLink] = useState("all");
  const deferredSearch = useDeferredValue(search);

  const items = useMemo(() => {
    return (lotsQuery.data?.data ?? []).filter((item) => {
      if (!isOwnedLot(item, session)) {
        return false;
      }
      const searchMatch = `${item.productLabel} ${item.sellerCompanyId} ${item.id}`
        .toLowerCase()
        .includes(deferredSearch.toLowerCase());
      const statusMatch = status === "all" || item.status === status;
      const auctionMatch =
        auctionLink === "all" ||
        (auctionLink === "linked" ? Boolean(item.auctionId) : !item.auctionId);
      return searchMatch && statusMatch && auctionMatch;
    });
  }, [auctionLink, deferredSearch, lotsQuery.data?.data, session, status]);

  return (
    <div className="page-stack">
      <PageHeader
        compact
        eyebrow="Лоты"
        title="Лоты"
        description="Партии продукции, подготовленные к публикации и запуску торгов."
        metrics={
          <>
            <div>
              <span>Найдено</span>
              <strong>{items.length}</strong>
            </div>
            <div>
              <span>С аукционом</span>
              <strong>{items.filter((item) => Boolean(item.auctionId)).length}</strong>
            </div>
          </>
        }
      />

      <SectionCard eyebrow="Фильтры" title="Рабочий список лотов">
        <FilterBar
          search={search}
          onSearchChange={setSearch}
          status={status}
          onStatusChange={setStatus}
          statusOptions={[
            { label: "Все статусы", value: "all" },
            { label: lotStatusLabels.DRAFT, value: "DRAFT" },
            { label: lotStatusLabels.PUBLISHED, value: "PUBLISHED" },
            { label: lotStatusLabels.CLOSED, value: "CLOSED" },
            { label: lotStatusLabels.CANCELLED, value: "CANCELLED" },
          ]}
          source="all"
          onSourceChange={() => undefined}
          showSource={false}
          searchPlaceholder="Название продукта, компания или номер лота"
          extraFilters={
            <Field label="Аукцион">
              <Select value={auctionLink} onChange={(event) => setAuctionLink(event.target.value)}>
                <option value="all">Все</option>
                <option value="linked">Есть аукцион</option>
                <option value="free">Без аукциона</option>
              </Select>
            </Field>
          }
        />
      </SectionCard>

      {items.length ? (
        <section className="data-panel">
          <div className="data-list">
          {items.map((item) => (
            <div className="data-row" key={item.id}>
              <div className="data-row-main">
                <h3>{item.productLabel}</h3>
                <p>Лот #{shortId(item.id)} · продавец {displayCompany(item.sellerCompanyId)}</p>
              </div>
              <div className="data-cell">
                <span>Статус</span>
                <strong>
                  <StatusBadge status={item.status} label={lotStatusLabels[item.status]} />
                </strong>
              </div>
              <div className="data-cell">
                <span>Цена / шаг</span>
                <strong>
                  {formatMoney(item.currentPrice ?? item.startPrice)} · {formatMoney(item.minBidStep)}
                </strong>
              </div>
              <div className="data-cell">
                <span>Старт торгов</span>
                <strong>{formatDateTime(item.auctionStartsAt)}</strong>
              </div>
              <div className="data-actions">
                {item.auctionId ? (
                  <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href={`/auctions/${item.auctionId}`}>
                    Аукцион
                  </Link>
                ) : (
                  <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/create/lot">
                    Настроить
                  </Link>
                )}
              </div>
            </div>
          ))}
          </div>
        </section>
      ) : (
        <EmptyState
          title="Лоты не найдены"
          description="У вас пока нет доступных лотов. Создайте свой лот или снимите фильтры."
          actionHref={canCreateSupply ? "/create/lot" : undefined}
          actionLabel={canCreateSupply ? "Создать лот" : undefined}
        />
      )}
    </div>
  );
}
