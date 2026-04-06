"use client";

import type { ReactNode } from "react";

import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";

interface Option {
  label: string;
  value: string;
}

export function FilterBar({
  search,
  onSearchChange,
  status,
  onStatusChange,
  statusOptions,
  source,
  onSourceChange,
  searchLabel = "Поиск",
  searchPlaceholder = "Продукт, лот, аукцион, компания",
  showSource = true,
  showStatus = true,
  extraFilters,
}: {
  search: string;
  onSearchChange: (value: string) => void;
  status: string;
  onStatusChange: (value: string) => void;
  statusOptions: Option[];
  source: string;
  onSourceChange: (value: string) => void;
  searchLabel?: string;
  searchPlaceholder?: string;
  showSource?: boolean;
  showStatus?: boolean;
  extraFilters?: ReactNode;
}) {
  return (
    <div className="filters">
      <div className="filters-search">
        <Field label={searchLabel}>
          <Input
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder={searchPlaceholder}
          />
        </Field>
      </div>
      {showStatus ? (
        <Field label="Статус">
          <Select value={status} onChange={(event) => onStatusChange(event.target.value)}>
            {statusOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </Select>
        </Field>
      ) : null}
      {showSource ? (
        <Field label="Источник">
          <Select value={source} onChange={(event) => onSourceChange(event.target.value)}>
            <option value="all">Все</option>
            <option value="api">API</option>
            <option value="mock">Mock / Local</option>
            <option value="mixed">Mixed</option>
          </Select>
        </Field>
      ) : null}
      {extraFilters}
    </div>
  );
}
