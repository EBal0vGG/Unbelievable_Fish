import { NextRequest } from "next/server";

import { proxyRequest } from "@/shared/api/proxy";

export async function GET(
  request: NextRequest,
  context: { params: { path: string[] } },
) {
  const { path } = context.params;
  return proxyRequest(request, "trading", path);
}

export async function POST(
  request: NextRequest,
  context: { params: { path: string[] } },
) {
  const { path } = context.params;
  return proxyRequest(request, "trading", path);
}
