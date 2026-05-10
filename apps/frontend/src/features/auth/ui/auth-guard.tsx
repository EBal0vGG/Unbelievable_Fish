"use client";

import Link from "next/link";
import { useEffect, type ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";

import { useAuth } from "@/entities/session/model/auth-context";
import { hasRequiredRole } from "@/shared/lib/access";
import type { UserRole } from "@/shared/types/domain";
import { Notice } from "@/shared/ui/notice";

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

  const loginHref = `/login?next=${encodeURIComponent(pathname)}`;

  if (!isReady) {
    return (
      <div className="page-stack">
        <Notice title="Загрузка сессии">
          Проверяем авторизацию… Если identity-сервис не отвечает, страница освободится через несколько секунд.
        </Notice>
      </div>
    );
  }

  if (!session) {
    return (
      <div className="page-stack">
        <Notice title="Нужен вход">
          Перенаправляем на страницу входа. Если этого не произошло, откройте ссылку вручную.
        </Notice>
        <p className="muted">
          <Link href={loginHref}>Перейти ко входу</Link>
        </p>
      </div>
    );
  }

  if (roles && !hasRequiredRole(session, roles)) {
    return (
      <div className="page-stack">
        <Notice tone="warning" title="Недостаточно прав">
          Эта страница доступна другой роли.
        </Notice>
        <p className="muted">
          <Link href="/">На главную</Link>
        </p>
      </div>
    );
  }

  return <>{children}</>;
}
