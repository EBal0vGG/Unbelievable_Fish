import { shortId } from "@/shared/lib/format";
import type { UserSession } from "@/shared/types/domain";

const TECHNICAL_ID_PREFIXES = ["company-", "user-", "auction-", "lot-", "deal-"];

export function isTechnicalId(value?: string | null): boolean {
  if (!value) {
    return false;
  }

  const normalized = value.trim().toLowerCase();
  return (
    TECHNICAL_ID_PREFIXES.some((prefix) => normalized.startsWith(prefix)) ||
    /^[0-9a-f]{12,}$/i.test(normalized) ||
    /^[a-z]+-[0-9a-f]{10,}$/i.test(normalized)
  );
}

export function displayText(value?: string | number | null, fallback = "—"): string {
  if (value === null || value === undefined || value === "") {
    return fallback;
  }

  return String(value);
}

export function displayPerson(session?: UserSession | null): string {
  const candidates = [session?.name, session?.login]
    .map((value) => value?.trim())
    .filter(Boolean) as string[];
  const readable = candidates.find((value) => !isTechnicalId(value));
  if (!readable) {
    return "Пользователь";
  }

  return readable.includes("@") ? readable.split("@")[0] : readable;
}

export function initialsFromName(value: string): string {
  const parts = value
    .trim()
    .split(/[\s._-]+/)
    .filter(Boolean);

  return (parts.length > 1 ? `${parts[0][0]}${parts[1][0]}` : value.slice(0, 2)).toUpperCase();
}

export function displayCompany(value?: string | null): string {
  if (!value) {
    return "Компания не указана";
  }

  return isTechnicalId(value) ? `Компания #${shortId(value)}` : value;
}

export function displayId(value?: string | null, label = "#"): string {
  if (!value) {
    return "—";
  }

  return `${label}${shortId(value)}`;
}
