import { env } from "@/shared/config/env";
import type { ServiceMeta, ServiceResult } from "@/shared/types/domain";

export function apiMeta(note?: string): ServiceMeta {
  return { source: "api", note };
}

export function mockMeta(note: string): ServiceMeta {
  return { source: "mock", placeholder: true, note };
}

export function mixedMeta(note: string): ServiceMeta {
  return { source: "mixed", placeholder: true, note };
}

export async function withFallback<T>(
  runApi: () => Promise<T>,
  runFallback: () => T,
  note: string,
): Promise<ServiceResult<T>> {
  try {
    return { data: await runApi(), meta: apiMeta() };
  } catch (error) {
    if (!env.enableApiFallback) {
      throw error;
    }

    return { data: runFallback(), meta: mockMeta(note) };
  }
}
