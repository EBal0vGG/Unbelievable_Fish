"use client";

import { useDeferredValue, useMemo, useState } from "react";

import { useDealsQuery } from "@/entities/deal/model/hooks";
import { dealStatusLabels } from "@/entities/deal/ui/deal-card";
import { useAuth } from "@/entities/session/model/auth-context";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { displayCompany } from "@/shared/lib/display";
import { formatDateTime, formatMoney, shortId } from "@/shared/lib/format";
import { Badge } from "@/shared/ui/badge";
import { buttonStyles } from "@/shared/ui/button";
import { EmptyState } from "@/shared/ui/empty-state";
import { Field } from "@/shared/ui/field";
import { PageHeader } from "@/shared/ui/page-header";
import { SectionCard } from "@/shared/ui/section-card";
import { Select } from "@/shared/ui/select";
import type { DealStatus } from "@/shared/types/domain";
import Link from "next/link";

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

function dealTone(status: DealStatus) {
  switch (status) {
    case "completed":
    case "paid":
      return "success";
    case "cancelled":
      return "danger";
    case "pending":
    case "payment_requested":
      return "warning";
    default:
      return "info";
  }
}

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
      <PageHeader
        compact
        eyebrow="Ваши сделки"
        title="Контракты после торгов"
        description="Подтверждение победителя, контракт, оплата и отгрузка в одном рабочем контуре."
        metrics={
          <>
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
          </>
        }
      />

      <SectionCard eyebrow="Фильтры" title="Рабочий список сделок">
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
          searchPlaceholder="Ваша сделка, аукцион, продукт или компания"
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
      </SectionCard>

      {items.length ? (
        <section className="data-panel">
          <div className="data-list">
          {items.map((item) => (
            <div className="data-row" key={item.id}>
              <div className="data-row-main">
                <h3>{item.productSnapshot.name || `Сделка ${shortId(item.id)}`}</h3>
                <p>Сделка #{shortId(item.id)} · аукцион {shortId(item.auctionId)}</p>
              </div>
              <div className="data-cell">
                <span>Статус</span>
                <strong>
                  <Badge tone={dealTone(item.status)}>{dealStatusLabels[item.status]}</Badge>
                </strong>
              </div>
              <div className="data-cell">
                <span>Сумма / объем</span>
                <strong>{formatMoney(item.totalAmount)} · {item.quantity} {item.productSnapshot.unit || "ед."}</strong>
              </div>
              <div className="data-cell">
                <span>Стороны / дата</span>
                <strong>
                  {item.supplierId === session?.companyId ? "Продажа" : "Покупка"} ·{" "}
                  {displayCompany(item.supplierId === session?.companyId ? item.customerId : item.supplierId)} ·{" "}
                  {formatDateTime(item.createdAt)}
                </strong>
              </div>
              <div className="data-actions">
                <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href={`/deals/${item.id}`}>
                  Открыть
                </Link>
              </div>
            </div>
          ))}
          </div>
        </section>
      ) : !session ? (
        <EmptyState
          title="Войдите, чтобы увидеть ваши сделки"
          description="Сделки доступны только поставщику и покупателю, которые участвуют в контракте."
          actionHref="/login?next=/deals"
          actionLabel="Войти"
        />
      ) : !session.companyId ? (
        <EmptyState
          title="Привяжите компанию, чтобы увидеть сделки"
          description="Сделки становятся доступны после регистрации или выбора компании в рабочем профиле."
        />
      ) : (
        <EmptyState
          title="Ваши сделки не найдены"
          description="Сделка появляется после завершения аукциона и выбора победителя."
          actionHref="/auctions"
          actionLabel="Открыть аукционы"
        />
      )}
    </div>
  );
}
