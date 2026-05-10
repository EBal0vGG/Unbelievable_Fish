"use client";

import { useDeferredValue, useMemo, useState } from "react";

import { useProductsQuery } from "@/entities/lot/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { isOwnedProduct, isSellerSession } from "@/shared/lib/access";
import { displayCompany } from "@/shared/lib/display";
import { shortId } from "@/shared/lib/format";
import { productStatusLabels } from "@/shared/lib/labels";
import { buttonStyles } from "@/shared/ui/button";
import { EmptyState } from "@/shared/ui/empty-state";
import { Field } from "@/shared/ui/field";
import { PageHeader } from "@/shared/ui/page-header";
import { SectionCard } from "@/shared/ui/section-card";
import { Select } from "@/shared/ui/select";
import { StatusBadge } from "@/shared/ui/status-badge";
import Link from "next/link";

export default function ProductsPage() {
  const { session } = useAuth();
  const canCreateSupply = isSellerSession(session);
  const productsQuery = useProductsQuery();
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [processingType, setProcessingType] = useState("all");
  const deferredSearch = useDeferredValue(search);

  const items = useMemo(() => {
    return (productsQuery.data?.data ?? []).filter((item) => {
      if (!isOwnedProduct(item, session)) {
        return false;
      }

      const searchMatch = `${item.fishName} ${item.processingType} ${item.size} ${item.id}`
        .toLowerCase()
        .includes(deferredSearch.toLowerCase());
      const statusMatch = status === "all" || item.status === status;
      const processingMatch = processingType === "all" || item.processingType === processingType;

      return searchMatch && statusMatch && processingMatch;
    });
  }, [deferredSearch, processingType, productsQuery.data?.data, session, status]);

  return (
    <div className="page-stack">
      <PageHeader
        compact
        eyebrow="Продукты"
        title="Мои продукты"
        description="Позиции вашей компании, из которых формируются лоты для торгов."
        metrics={
          <>
            <div>
              <span>Найдено</span>
              <strong>{items.length}</strong>
            </div>
            <div>
              <span>Опубликовано</span>
              <strong>{items.filter((item) => item.status === "PUBLISHED").length}</strong>
            </div>
          </>
        }
      />

      <SectionCard eyebrow="Фильтры" title="Рабочий список продуктов">
        <FilterBar
          search={search}
          onSearchChange={setSearch}
          status={status}
          onStatusChange={setStatus}
          statusOptions={[
            { label: "Все статусы", value: "all" },
            { label: productStatusLabels.DRAFT, value: "DRAFT" },
            { label: productStatusLabels.PUBLISHED, value: "PUBLISHED" },
          ]}
          source="all"
          onSourceChange={() => undefined}
          showSource={false}
          extraFilters={
            <Field label="Обработка">
              <Select value={processingType} onChange={(event) => setProcessingType(event.target.value)}>
                <option value="all">Все</option>
                <option value="chilled">Охлажденная</option>
                <option value="frozen">Замороженная</option>
                <option value="live">Живая</option>
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
                <h3>{item.fishName}</h3>
                <p>Продукт #{shortId(item.id)} · {displayCompany(item.ownerCompanyId)}</p>
              </div>
              <div className="data-cell">
                <span>Статус</span>
                <strong>
                  <StatusBadge status={item.status} label={productStatusLabels[item.status]} />
                </strong>
              </div>
              <div className="data-cell">
                <span>Обработка</span>
                <strong>{item.processingType}</strong>
              </div>
              <div className="data-cell">
                <span>Вес / размер</span>
                <strong>
                  {item.weight} {item.unit} · {item.size}
                </strong>
              </div>
              <div className="data-actions">
                <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href="/create/lot">
                  В лот
                </Link>
              </div>
            </div>
          ))}
          </div>
        </section>
      ) : (
        <EmptyState
          title="Продукты не найдены"
          description="У вас пока нет продуктов под текущими фильтрами. Создание продукта доступно в сценарии нового лота."
          actionHref={canCreateSupply ? "/create/lot" : undefined}
          actionLabel={canCreateSupply ? "Создать продукт" : undefined}
        />
      )}
    </div>
  );
}
