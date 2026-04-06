import type { LotRecord, ProductRecord, UserRole, UserSession } from "@/shared/types/domain";

export function normalizeRole(role?: string | null): UserRole {
  return role === "admin" ? "admin" : "user";
}

export function isAdminSession(session: UserSession | null): boolean {
  return normalizeRole(session?.role) === "admin";
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
