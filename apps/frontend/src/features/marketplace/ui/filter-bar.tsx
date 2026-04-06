"use client";

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
  showSource = true,
}: {
  search: string;
  onSearchChange: (value: string) => void;
  status: string;
  onStatusChange: (value: string) => void;
  statusOptions: Option[];
  source: string;
  onSourceChange: (value: string) => void;
  showSource?: boolean;
}) {
  return (
    <div className="filters">
      <Field label="Поиск">
        <Input value={search} onChange={(event) => onSearchChange(event.target.value)} placeholder="Рыба, lot, auction, company" />
      </Field>
      <Field label="Статус">
        <Select value={status} onChange={(event) => onStatusChange(event.target.value)}>
          {statusOptions.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </Select>
      </Field>
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
    </div>
  );
}
