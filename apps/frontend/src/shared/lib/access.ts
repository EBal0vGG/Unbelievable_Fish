import type { DealRecord, LotRecord, ProductRecord, UserRole, UserSession } from "@/shared/types/domain";

export type MainInterfaceMode = "buyer" | "seller" | "all";
export type DealParticipantSide = "customer" | "supplier" | "outsider";

export function normalizeRole(role?: string | null): UserRole {
  if (role === "admin" || role === "seller" || role === "buyer" || role === "buyer_seller") {
    return role;
  }

  return "buyer";
}

export function isAdminSession(session: UserSession | null): boolean {
  return normalizeRole(session?.role) === "admin";
}

export function isSellerSession(session: UserSession | null): boolean {
  const role = normalizeRole(session?.role);
  return role === "seller" || role === "buyer_seller";
}

export function isBuyerSession(session: UserSession | null): boolean {
  const role = normalizeRole(session?.role);
  return role === "buyer" || role === "buyer_seller";
}

export function getMainInterfaceMode(session: UserSession | null): MainInterfaceMode {
  if (!session) {
    return "all";
  }
  const role = normalizeRole(session?.role);
  if (role === "buyer") {
    return "buyer";
  }
  if (role === "seller") {
    return "seller";
  }
  return "all";
}

export function hasRequiredRole(session: UserSession | null, roles: UserRole[]): boolean {
  if (!session) {
    return false;
  }
  return roles.some((role) => {
    if (role === "seller") {
      return isSellerSession(session);
    }
    if (role === "buyer") {
      return isBuyerSession(session);
    }
    return normalizeRole(session.role) === role;
  });
}

export function isOwnedProduct(product: ProductRecord, session: UserSession | null): boolean {
  if (!session) {
    return false;
  }

  return product.ownerCompanyId === session.companyId && product.ownerUserId === session.userId;
}

export function isOwnedLot(lot: LotRecord, session: UserSession | null): boolean {
  if (!session) {
    return false;
  }

  return lot.sellerCompanyId === session.companyId && lot.creatorUserId === session.userId;
}

export function getDealParticipantSide(deal: DealRecord, session: UserSession | null): DealParticipantSide {
  if (!session?.companyId) {
    return "outsider";
  }
  if (deal.supplierId === session.companyId) {
    return "supplier";
  }
  if (deal.customerId === session.companyId) {
    return "customer";
  }
  return "outsider";
}
