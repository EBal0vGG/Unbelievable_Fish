import type { DealRecord, DealStatus } from "@/shared/types/domain";

/** Deal stages where billing UI (invoice / payout) is meaningful. Prefer API capabilities when available. */
const DEAL_STATUSES_WITH_BILLING_UI: ReadonlySet<DealStatus> = new Set([
  "payment_requested",
  "paid",
  "shipment_requested",
]);

export function dealShowsBillingPanel(deal: DealRecord): boolean {
  return DEAL_STATUSES_WITH_BILLING_UI.has(deal.status);
}
