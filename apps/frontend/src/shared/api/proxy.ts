import { NextRequest } from "next/server";

import { serviceBaseUrls } from "@/shared/config/env";

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
    });
  } catch (error) {
    return Response.json(
      {
        code: "UPSTREAM_UNAVAILABLE",
        message: error instanceof Error ? error.message : "upstream service unavailable",
      },
      { status: 502 },
    );
  }

  const nextHeaders = new Headers(response.headers);
  nextHeaders.delete("content-encoding");
  nextHeaders.delete("content-length");
  nextHeaders.delete("transfer-encoding");

  return new Response(response.body, {
    status: response.status,
    headers: nextHeaders,
  });
}
