# ADR-0009: Invoice payment timeout forfeits winner deposit

## Context

After an auction, the winning buyer enters the deal flow (confirm → contract → payment request → invoice). If the **deal invoice** is not paid by `due_at`, the platform expires the invoice and must decide what happens to the buyer’s **held auction deposit** and to **winner selection**.

Marketplaces differ: some use grace periods, manual review, or do not forfeit deposits after a signed contract.

---

После аукциона покупатель-победитель проходит сделку до выставления инвойса. Если инвойс не оплачен к `due_at`, платформа помечает инвойс просроченным и должна определить судьбу **удержанного депозита** и **winner selection**.

## Decision

When a deal invoice expires unpaid (`billing.DealInvoiceExpired` → `HandleDealInvoiceExpired`):

1. The current deal is cancelled with reason **`PAYMENT_TIMEOUT`**.
2. **`WinnerRejected`** is emitted with that reason; integration runs **`CaptureAuctionDeposit`**, which **captures the full HELD deposit** of that buyer for the auction (same path as other winner-forfeit reasons from billing’s perspective).
3. Winner selection is **reopened** and **advanced** to the next ranked candidate; a **new deal** is created when candidates remain, or **`WinnerSelectionFailed`** if none.

This is an explicit **business policy**: **invoice payment timeout forfeits the winner’s deposit** and triggers automatic fallback to the next candidate—**not** a grace period or manual review in this codebase.

Operational note: idempotent no-ops in `HandleDealInvoiceExpired` (e.g. selection already `active`, or `selection.DealID` ≠ event deal) emit **structured warnings** so replay vs corruption can be investigated in logs.

## Consequences

- **Pros**: Predictable automation; aligns deposit capture with “buyer failed to complete payment after committing through the deal flow.”
- **Cons**: Stricter than some marketplaces; product/legal should confirm this matches intended terms of service.
- **Docs**: Billing/deals integration behavior is described here; HTTP or product copy may need to reference payment deadlines and deposit rules separately.
