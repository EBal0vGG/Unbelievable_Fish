"use client";

import { useDeferredValue, useMemo, useState } from "react";

import { useDealsQuery } from "@/entities/deal/model/hooks";
import { DealCard, dealStatusLabels } from "@/entities/deal/ui/deal-card";
import { useAuth } from "@/entities/session/model/auth-context";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { EmptyState } from "@/shared/ui/empty-state";
import { Field } from "@/shared/ui/field";
import { Select } from "@/shared/ui/select";
import type { DealStatus } from "@/shared/types/domain";

const statuses: DealStatus[] = [
  "pending",
  "confirmed",
  "contract_prepared",
  "contract_signed",
  "payment_requested",
  "paid",
  "shipment_requested",
  "shipped",
  "completed",
  "cancelled",
];

export default function DealsPage() {
  const { session } = useAuth();
  const dealsQuery = useDealsQuery(session);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [side, setSide] = useState("all");
  const deferredSearch = useDeferredValue(search);

  const items = useMemo(() => {
    return (dealsQuery.data?.data ?? []).filter((item) => {
      const haystack = `${item.id} ${item.auctionId} ${item.customerId} ${item.supplierId} ${item.status} ${item.productSnapshot.name} ${item.productSnapshot.processingType}`
        .toLowerCase()
        .trim();
      const searchMatch = !deferredSearch || haystack.includes(deferredSearch.toLowerCase());
      const statusMatch = status === "all" || item.status === status;
      const sideMatch =
        side === "all" ||
        (side === "supplier" && item.supplierId === session?.companyId) ||
        (side === "customer" && item.customerId === session?.companyId);

      return searchMatch && statusMatch && sideMatch;
    });
  }, [dealsQuery.data?.data, deferredSearch, session?.companyId, side, status]);

  const totals = useMemo(() => {
    return items.reduce(
      (acc, item) => ({
        amount: acc.amount + item.totalAmount,
        active: acc.active + (item.status !== "completed" && item.status !== "cancelled" ? 1 : 0),
      }),
      { amount: 0, active: 0 },
    );
  }, [items]);

  return (
    <div className="page-stack">
      <section className="page-hero compact-hero">
        <div>
          <p className="eyebrow">Твои сделки</p>
          <h1>Контракты после торгов</h1>
          <p className="hero-copy">
            Подтверждение победителя, контракт, оплата и отгрузка в одном рабочем контуре.
          </p>
        </div>
        <div className="hero-metrics">
          <div>
            <span>Активные</span>
            <strong>{totals.active}</strong>
          </div>
          <div>
            <span>В выборке</span>
            <strong>{items.length}</strong>
          </div>
          <div>
            <span>Оборот</span>
            <strong>{new Intl.NumberFormat("ru-RU", { notation: "compact" }).format(totals.amount)} ₽</strong>
          </div>
        </div>
      </section>

      <FilterBar
        search={search}
        onSearchChange={setSearch}
        status={status}
        onStatusChange={setStatus}
        statusOptions={[
          { label: "Все статусы", value: "all" },
          ...statuses.map((item) => ({ label: dealStatusLabels[item], value: item })),
        ]}
        source="all"
        onSourceChange={() => undefined}
        showSource={false}
        searchPlaceholder="Твоя сделка, аукцион, продукт или компания"
        extraFilters={
          <Field label="Сторона">
            <Select value={side} onChange={(event) => setSide(event.target.value)}>
              <option value="all">Все</option>
              <option value="supplier">Мы поставщик</option>
              <option value="customer">Мы покупатель</option>
            </Select>
          </Field>
        }
      />

      {items.length ? (
        <div className="card-grid card-grid-3">
          {items.map((item) => (
            <DealCard key={item.id} deal={item} />
          ))}
        </div>
      ) : !session ? (
        <EmptyState
          title="Войдите, чтобы увидеть свои сделки"
          description="Сделки доступны только поставщику и покупателю, которые участвуют в контракте."
          actionHref="/login?next=/deals"
          actionLabel="Войти"
        />
      ) : (
        <EmptyState
          title="Твои сделки не найдены"
          description="Сделка появляется после завершения аукциона и выбора победителя."
          actionHref="/auctions"
          actionLabel="Открыть аукционы"
        />
      )}
    </div>
  );
}
