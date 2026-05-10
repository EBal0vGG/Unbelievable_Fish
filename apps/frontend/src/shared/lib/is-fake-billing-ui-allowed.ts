/**
 * Fake pay / fake top-up controls in the browser.
 * - `NEXT_PUBLIC_ENABLE_FAKE_BILLING=false` — never show.
 * - `true` or unset — show when billing balance payload sets the matching `*_fake_confirm_enabled` flag.
 */
export function isFakeBillingUiAllowed(apiFeatureEnabled: boolean | undefined): boolean {
  if (process.env.NEXT_PUBLIC_ENABLE_FAKE_BILLING === "false") {
    return false;
  }
  return apiFeatureEnabled === true;
}
