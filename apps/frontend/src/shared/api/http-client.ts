import { makeClientId } from "@/shared/lib/id";
import type { UserSession } from "@/shared/types/domain";

export class ApiError extends Error {
  status: number;
  code?: string;
  details?: unknown;

  constructor(message: string, status: number, code?: string, details?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

type ServiceName = "catalog" | "trading" | "deals" | "identity";

interface RequestOptions {
  method?: "GET" | "POST" | "PUT";
  session?: UserSession | null;
  body?: unknown;
  signal?: AbortSignal;
}

export function isRecoverableApiGap(error: unknown): boolean {
  if (!(error instanceof ApiError)) {
    return true;
  }

  return [0, 404, 405, 500, 501, 502, 503].includes(error.status);
}

export async function apiRequest<T>(
  service: ServiceName,
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const headers = new Headers();
  headers.set("Accept", "application/json");
  headers.set("X-Correlation-ID", makeClientId("corr"));
  headers.set("X-Causation-ID", makeClientId("cause"));

  if (options.session) {
    headers.set("Authorization", `Bearer ${options.session.accessToken}`);
    headers.set("X-Company-ID", options.session.companyId);
    headers.set("X-User-ID", options.session.userId);
  }

  if (options.body !== undefined) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`/api/${service}${path}`, {
    method: options.method ?? "GET",
    headers,
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
    signal: options.signal,
  }).catch((error: Error) => {
    throw new ApiError(error.message, 0);
  });

  if (!response.ok) {
    let payload: unknown;
    try {
      payload = await response.json();
    } catch {
      payload = await response.text();
    }

    const errorBody = payload as { code?: string; message?: string };
    throw new ApiError(errorBody.message ?? "Request failed", response.status, errorBody.code, payload);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    return (await response.json()) as T;
  }

  return undefined as T;
}
