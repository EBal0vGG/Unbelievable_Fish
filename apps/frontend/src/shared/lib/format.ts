const currencyFormatter = new Intl.NumberFormat("ru-RU", {
  style: "currency",
  currency: "RUB",
  maximumFractionDigits: 0,
});

const dateFormatter = new Intl.DateTimeFormat("ru-RU", {
  dateStyle: "medium",
  timeStyle: "short",
});

export function formatMoney(value?: number | null): string {
  if (value === null || value === undefined) {
    return "н/д";
  }

  return currencyFormatter.format(value);
}

export function formatDateTime(value?: string | null): string {
  if (!value) {
    return "н/д";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return dateFormatter.format(date);
}

export function shortId(value: string): string {
  if (value.length <= 14) {
    return value;
  }

  return `${value.slice(0, 6)}…${value.slice(-4)}`;
}

export function toDateTimeLocalValue(value: Date): string {
  const offsetMs = value.getTimezoneOffset() * 60_000;
  return new Date(value.getTime() - offsetMs).toISOString().slice(0, 16);
}
