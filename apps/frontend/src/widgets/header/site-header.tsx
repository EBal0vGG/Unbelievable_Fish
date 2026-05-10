"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { useAuth } from "@/entities/session/model/auth-context";
import { getMainInterfaceMode, isAdminSession, isSellerSession } from "@/shared/lib/access";
import { env } from "@/shared/config/env";
import { cn } from "@/shared/lib/cn";
import { roleLabels } from "@/shared/lib/labels";
import { buttonStyles } from "@/shared/ui/button";

const navItems = [
  { href: "/", label: "Биржа" },
  { href: "/catalog", label: "Каталог" },
  { href: "/products", label: "Продукты" },
  { href: "/lots", label: "Лоты" },
  { href: "/auctions", label: "Торги" },
  { href: "/deals", label: "Ваши сделки" },
  { href: "/create/lot", label: "Новый лот" },
  { href: "/me", label: "Профиль" },
];

export function SiteHeader() {
  const pathname = usePathname();
  const { session, logout } = useAuth();
  const interfaceMode = getMainInterfaceMode(session);
  const canManageFish = isAdminSession(session);
  const canCreateSupply = isSellerSession(session);
  const visibleNavItems = navItems.filter((item) => {
    if (interfaceMode === "buyer" && (item.href === "/products" || item.href === "/lots")) {
      return false;
    }
    if (item.href === "/deals" && !session?.companyId) {
      return false;
    }
    if (item.href === "/create/lot" && !canCreateSupply) {
      return false;
    }
    return true;
  });
  const adminBillingNav =
    canManageFish && env.enableBillingAdminUI
      ? [
          { href: "/admin/billing/invoices", label: "Счета (admin)" },
          { href: "/admin/billing/payouts", label: "Выплаты (admin)" },
        ]
      : [];

  const extendedNavItems = canManageFish
    ? [
        ...visibleNavItems.slice(0, 5),
        { href: "/create/fish", label: "Новая рыба" },
        ...adminBillingNav,
        ...visibleNavItems.slice(5),
      ]
    : visibleNavItems;

  return (
    <header className="site-header">
      <div className="brand-block">
        <Link href="/" className="brand-mark" aria-label="Рыбная биржа">
          <Image alt="Рыбная биржа" height={54} priority src="/fish-exchange-logo.svg" width={54} />
        </Link>
        <div>
          <p className="brand-title">Рыбная биржа</p>
          <p className="brand-subtitle">Seafood commodity trading</p>
        </div>
      </div>

      <nav aria-label="Основная навигация" className="main-nav">
        {extendedNavItems.map((item) => (
          <Link
            key={item.href}
            aria-current={
              pathname === item.href || (item.href !== "/" && pathname.startsWith(`${item.href}/`))
                ? "page"
                : undefined
            }
            className={cn(
              "nav-link",
              (pathname === item.href || (item.href !== "/" && pathname.startsWith(`${item.href}/`))) &&
                "nav-link-active",
            )}
            href={item.href}
          >
            {item.label}
          </Link>
        ))}
      </nav>

      <div className="header-actions">
        {session ? (
          <>
            <Link className="session-chip" href="/me">
              <span>{session.name}</span>
              <span>{session.companyId ? "Профиль компании" : "Профиль пользователя"} · {roleLabels[session.role]}</span>
            </Link>
            <button className={buttonStyles({ variant: "ghost", size: "sm" })} onClick={logout} type="button">
              Выйти
            </button>
          </>
        ) : (
          <>
            <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/login">
              Вход
            </Link>
            <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href="/register">
              Регистрация
            </Link>
          </>
        )}
      </div>
    </header>
  );
}
