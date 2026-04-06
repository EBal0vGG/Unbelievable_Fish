export const env = {
  appName: "Unbelievable Fish Exchange",
  catalogApiUrl: process.env.NEXT_PUBLIC_CATALOG_API_URL ?? "http://localhost:8081",
  tradingApiUrl: process.env.NEXT_PUBLIC_TRADING_API_URL ?? "http://localhost:8082",
  dealsApiUrl: process.env.NEXT_PUBLIC_DEALS_API_URL ?? "http://localhost:8083",
  enableApiFallback: process.env.NEXT_PUBLIC_ENABLE_API_FALLBACK !== "false",
  // Command fallback is intentionally disabled by default: write operations must hit real backend.
  enableCommandFallback: process.env.NEXT_PUBLIC_ENABLE_COMMAND_FALLBACK === "true",
} as const;

export const serviceBaseUrls = {
  catalog: env.catalogApiUrl,
  trading: env.tradingApiUrl,
  deals: env.dealsApiUrl,
} as const;
