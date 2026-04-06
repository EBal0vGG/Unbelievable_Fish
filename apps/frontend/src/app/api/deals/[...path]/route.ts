import { NextRequest } from "next/server";

import { proxyRequest } from "@/shared/api/proxy";

export async function GET(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> },
) {
  const { path } = await context.params;
  return proxyRequest(request, "deals", path);
}

export async function POST(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> },
) {
  const { path } = await context.params;
  return proxyRequest(request, "deals", path);
}
