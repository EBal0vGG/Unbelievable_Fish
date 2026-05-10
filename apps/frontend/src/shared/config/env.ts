export const env = {
  appName: "Unbelievable Fish Exchange",
  catalogApiUrl: process.env.NEXT_PUBLIC_CATALOG_API_URL ?? "http://localhost:8081",
  tradingApiUrl: process.env.NEXT_PUBLIC_TRADING_API_URL ?? "http://localhost:8082",
  dealsApiUrl: process.env.NEXT_PUBLIC_DEALS_API_URL ?? "http://localhost:8083",
  identityApiUrl: process.env.NEXT_PUBLIC_IDENTITY_API_URL ?? "http://localhost:8084",
  billingApiUrl: process.env.NEXT_PUBLIC_BILLING_URL ?? "http://localhost:8085/billing",
  enableApiFallback: process.env.NEXT_PUBLIC_ENABLE_API_FALLBACK !== "false",
  enableCommandFallback: process.env.NEXT_PUBLIC_ENABLE_COMMAND_FALLBACK === "true",
  /** Strict opt-in only when set to "true"; UI also uses `isFakeBillingUiAllowed` from shared/lib (follows balance API when unset). */
  enableFakeBillingUI: process.env.NEXT_PUBLIC_ENABLE_FAKE_BILLING === "true",
  /** Show billing admin operator tools (requires admin JWT + BILLING_ENABLE_ADMIN_ACTIONS on server). */
  enableBillingAdminUI: process.env.NEXT_PUBLIC_ENABLE_BILLING_ADMIN === "true",
} as const;

export const serviceBaseUrls = {
  catalog: env.catalogApiUrl,
  trading: env.tradingApiUrl,
  deals: env.dealsApiUrl,
  identity: env.identityApiUrl,
  billing: env.billingApiUrl,
} as const;
