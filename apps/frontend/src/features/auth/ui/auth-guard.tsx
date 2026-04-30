"use client";

import { useEffect, type ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";

import { useAuth } from "@/entities/session/model/auth-context";
import { hasRequiredRole } from "@/shared/lib/access";
import type { UserRole } from "@/shared/types/domain";

export function AuthGuard({
  children,
  roles,
}: {
  children: ReactNode;
  roles?: UserRole[];
}) {
  const pathname = usePathname();
  const router = useRouter();
  const { isReady, session } = useAuth();

  useEffect(() => {
    if (!isReady) {
      return;
    }

    if (!session) {
      router.replace(`/login?next=${encodeURIComponent(pathname)}`);
      return;
    }

    if (roles && !hasRequiredRole(session, roles)) {
      router.replace("/");
    }
  }, [isReady, pathname, roles, router, session]);

  if (!isReady || !session) {
    return null;
  }

  if (roles && !hasRequiredRole(session, roles)) {
    return null;
  }

  return <>{children}</>;
}
