import type { LotRecord, ProductRecord, UserRole, UserSession } from "@/shared/types/domain";

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
