export const env = {
  appName: "Unbelievable Fish Exchange",
  catalogApiUrl: process.env.NEXT_PUBLIC_CATALOG_API_URL ?? "http://localhost:8081",
  tradingApiUrl: process.env.NEXT_PUBLIC_TRADING_API_URL ?? "http://localhost:8082",
  dealsApiUrl: process.env.NEXT_PUBLIC_DEALS_API_URL ?? "http://localhost:8083",
  identityApiUrl: process.env.NEXT_PUBLIC_IDENTITY_API_URL ?? "http://localhost:8084",
  billingApiUrl: process.env.NEXT_PUBLIC_BILLING_URL ?? "http://localhost:8085/billing",
  enableApiFallback: process.env.NEXT_PUBLIC_ENABLE_API_FALLBACK !== "false",
  enableCommandFallback: process.env.NEXT_PUBLIC_ENABLE_COMMAND_FALLBACK === "true",
  /**
   * Explicit opt-in for fake billing UI (top-up / invoice). Prefer {@link isFakeBillingUiAllowed}:
   * when this env is unset, UI follows `deal_invoice_fake_confirm_enabled` from GET /accounts/me.
   */
  enableFakeBillingUI: process.env.NEXT_PUBLIC_ENABLE_FAKE_BILLING === "true",
  /** Show billing admin operator tools (requires admin JWT + BILLING_ENABLE_ADMIN_ACTIONS on server). */
  enableBillingAdminUI: process.env.NEXT_PUBLIC_ENABLE_BILLING_ADMIN === "true",
} as const;

/**
 * Fake pay / fake top-up controls.
 * - `NEXT_PUBLIC_ENABLE_FAKE_BILLING=false` — never show (prod bundle talking to any API).
 * - `true` or unset — show when billing includes the flag on the balance payload (requires BILLING_ENABLE_FAKE_PROVIDER).
 */
export function isFakeBillingUiAllowed(apiFeatureEnabled: boolean | undefined): boolean {
  if (process.env.NEXT_PUBLIC_ENABLE_FAKE_BILLING === "false") {
    return false;
  }
  return apiFeatureEnabled === true;
}

export const serviceBaseUrls = {
  catalog: env.catalogApiUrl,
  trading: env.tradingApiUrl,
  deals: env.dealsApiUrl,
  identity: env.identityApiUrl,
  billing: env.billingApiUrl,
} as const;
