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
    return "—";
  }

  return currencyFormatter.format(value);
}

export function formatDateTime(value?: string | null): string {
  if (!value) {
    return "—";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return dateFormatter.format(date);
}

export function shortId(value: string, head = 6, tail = 4): string {
  const minLength = head + tail + 4;
  if (value.length <= minLength) {
    return value;
  }

  return `${value.slice(0, head)}…${value.slice(-tail)}`;
}

export function toDateTimeLocalValue(value: Date): string {
  const offsetMs = value.getTimezoneOffset() * 60_000;
  return new Date(value.getTime() - offsetMs).toISOString().slice(0, 16);
}
