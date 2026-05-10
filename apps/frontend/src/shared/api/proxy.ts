import { NextRequest } from "next/server";

import { serviceBaseUrls } from "@/shared/config/env";

/** Upstream fetch budget — without this, a stuck Go handler leaves the browser spinner running with no completed access log. */
const UPSTREAM_FETCH_TIMEOUT_MS = Number(process.env.API_PROXY_UPSTREAM_TIMEOUT_MS ?? 45_000);

type ServiceName = keyof typeof serviceBaseUrls;

function stripTrailingSlash(value: string): string {
  return value.endsWith("/") ? value.slice(0, -1) : value;
}

async function readBody(request: NextRequest): Promise<ArrayBuffer | undefined> {
  if (request.method === "GET" || request.method === "HEAD") {
    return undefined;
  }

  return request.arrayBuffer();
}

export async function proxyRequest(
  request: NextRequest,
  service: ServiceName,
  pathSegments: string[],
): Promise<Response> {
  const baseUrl = stripTrailingSlash(serviceBaseUrls[service]);
  const path = pathSegments.join("/");
  const search = request.nextUrl.search;
  const targetUrl = `${baseUrl}/${path}${search}`;

  if (process.env.NODE_ENV === "development") {
    console.info("[api-proxy] forward", { service, method: request.method, path: `/${path}` });
  }

  const headers = new Headers();
  for (const [key, value] of request.headers.entries()) {
    if (["host", "connection", "content-length"].includes(key.toLowerCase())) {
      continue;
    }
    headers.set(key, value);
  }

  let response: Response;
  try {
    response = await fetch(targetUrl, {
      method: request.method,
      headers,
      body: await readBody(request),
      cache: "no-store",
      signal: AbortSignal.timeout(UPSTREAM_FETCH_TIMEOUT_MS),
    });
  } catch (error) {
    const aborted = error instanceof Error && error.name === "AbortError";
    console.error("[api-proxy] upstream unavailable", {
      service,
      targetUrl,
      method: request.method,
      timeout_ms: aborted ? UPSTREAM_FETCH_TIMEOUT_MS : undefined,
      error: error instanceof Error ? error.message : String(error),
    });
    return Response.json(
      {
        code: aborted ? "UPSTREAM_TIMEOUT" : "UPSTREAM_UNAVAILABLE",
        message: aborted
          ? `upstream did not respond within ${UPSTREAM_FETCH_TIMEOUT_MS}ms`
          : error instanceof Error
            ? error.message
            : "upstream service unavailable",
      },
      { status: aborted ? 504 : 502 },
    );
  }

  const nextHeaders = new Headers(response.headers);
  nextHeaders.delete("content-encoding");
  nextHeaders.delete("content-length");
  nextHeaders.delete("transfer-encoding");

  // 4xx are often expected domain responses (validation, not-found while async read-model catches up).
  // Keep proxy logs focused on real infrastructure failures.
  if (response.status >= 500) {
    console.warn("[api-proxy] upstream error", {
      service,
      targetUrl,
      method: request.method,
      status: response.status,
    });
  }

  return new Response(response.body, {
    status: response.status,
    headers: nextHeaders,
  });
}
