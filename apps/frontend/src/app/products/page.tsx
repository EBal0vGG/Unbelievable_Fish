"use client";

import { useDeferredValue, useMemo, useState } from "react";

import { useProductsQuery } from "@/entities/lot/model/hooks";
import { ProductCard } from "@/entities/product/ui/product-card";
import { useAuth } from "@/entities/session/model/auth-context";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { isOwnedProduct } from "@/shared/lib/access";
import { EmptyState } from "@/shared/ui/empty-state";
import { Field } from "@/shared/ui/field";
import { Select } from "@/shared/ui/select";

export default function ProductsPage() {
  const { session } = useAuth();
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
      <div className="page-heading">
        <p className="eyebrow">Продукты</p>
        <h1>Мои продукты</h1>
      </div>

      <FilterBar
        search={search}
        onSearchChange={setSearch}
        status={status}
        onStatusChange={setStatus}
        statusOptions={[
          { label: "Все статусы", value: "all" },
          { label: "DRAFT", value: "DRAFT" },
          { label: "PUBLISHED", value: "PUBLISHED" },
        ]}
        source="all"
        onSourceChange={() => undefined}
        showSource={false}
        extraFilters={
          <Field label="Обработка">
            <Select value={processingType} onChange={(event) => setProcessingType(event.target.value)}>
              <option value="all">Все</option>
              <option value="chilled">chilled</option>
              <option value="frozen">frozen</option>
              <option value="live">live</option>
            </Select>
          </Field>
        }
      />

      {items.length ? (
        <div className="card-grid card-grid-3">
          {items.map((item) => (
            <ProductCard key={item.id} product={item} />
          ))}
        </div>
      ) : (
        <EmptyState
          title="Продукты не найдены"
          description="У вас пока нет продуктов. Создайте новую позицию или измените фильтры."
          actionHref="/create/lot"
          actionLabel="Создать продукт"
        />
      )}
    </div>
  );
}
