"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { useAuth } from "@/entities/session/model/auth-context";
import { isAdminSession, isSellerSession } from "@/shared/lib/access";
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
  { href: "/create/auction", label: "Новый аукцион" },
  { href: "/me", label: "Профиль" },
];

export function SiteHeader() {
  const pathname = usePathname();
  const { session, logout } = useAuth();
  const canManageFish = isAdminSession(session);
  const canCreateSupply = isSellerSession(session);
  const visibleNavItems = navItems.filter((item) => {
    if (item.href === "/deals" && !session) {
      return false;
    }
    if ((item.href === "/create/lot" || item.href === "/create/auction") && !canCreateSupply) {
      return false;
    }
    return true;
  });
  const extendedNavItems = canManageFish
    ? [...visibleNavItems.slice(0, 5), { href: "/create/fish", label: "Новая рыба" }, ...visibleNavItems.slice(5)]
    : visibleNavItems;

  return (
    <header className="site-header">
      <div className="brand-block">
        <Link href="/" className="brand-mark">
          <Image alt="Рыбная биржа" height={54} priority src="/fish-exchange-logo.svg" width={54} />
        </Link>
        <div>
          <p className="brand-title">Рыбная биржа</p>
          <p className="brand-subtitle">B2B marketplace</p>
        </div>
      </div>

      <nav className="main-nav">
        {extendedNavItems.map((item) => (
          <Link
            key={item.href}
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
              <span>Профиль компании · {roleLabels[session.role]}</span>
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
